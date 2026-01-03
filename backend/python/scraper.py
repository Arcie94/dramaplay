import cloudscraper
from bs4 import BeautifulSoup
import json
import base64
from Crypto.Cipher import AES
from Crypto.Util.Padding import unpad
from hashlib import md5
import sys
import re

# Redirect print to stderr to keep stdout clean for JSON
def log(*args, **kwargs):
    print(*args, file=sys.stderr, **kwargs)

class IdlixScraper:
    def __init__(self):
        self.scraper = cloudscraper.create_scraper()
        self.base_url = "https://tv12.idlixku.com"
        self.ajax_url = f"{self.base_url}/wp-admin/admin-ajax.php"
        self.keys = [
            "459283", "idlix", "id123", "root", "dooplay", 
            "tv12.idlixku.com", "https://tv12.idlixku.com/", "Dooplay", "admin"
        ]
        self.update_dynamic_keys()

    def update_dynamic_keys(self):
        log("Updating dynamic keys...")
        try:
            response = self.scraper.get(self.base_url, timeout=10)
            if response.status_code == 200:
                # Try multiple patterns for nonce
                patterns = [
                    r'"nonce":"(\w+)"',
                    r'dt_nonce\s*=\s*"(\w+)"',
                    r'name="nonce"\s+value="(\w+)"',
                    r'id="result_nonce"\s+value="(\w+)"'
                ]
                
                found = False
                for pattern in patterns:
                    match = re.search(pattern, response.text)
                    if match:
                        nonce = match.group(1)
                        log(f"Found nonce with pattern '{pattern}': {nonce}")
                        if nonce not in self.keys:
                            self.keys.insert(0, nonce)
                        found = True
                
                if not found:
                    log("No nonce found in homepage.")
                    
        except Exception as e:
            log(f"Failed to update keys: {e}")

    def bytes_to_key(self, salt, password):
        dt = dt_tmp = b''
        password = password.encode('utf-8')
        while len(dt) < 48:
            dt_tmp = md5(dt_tmp + password + salt).digest()
            dt += dt_tmp
        return dt[:32], dt[32:]

    def decrypt(self, encrypted_json, password):
        try:
            data = json.loads(encrypted_json)
            ct = base64.b64decode(data['ct'])
            iv = bytes.fromhex(data['iv'])
            salt = bytes.fromhex(data['s'])
            
            key, _ = self.bytes_to_key(salt, password)
            cipher = AES.new(key, AES.MODE_CBC, iv)
            decrypted = unpad(cipher.decrypt(ct), AES.block_size)
            return decrypted.decode('utf-8')
        except Exception:
            return None

    def get_latest(self, page=1):
        url = f"{self.base_url}/page/{page}/" if page > 1 else self.base_url
        log(f"Fetching latest from {url}")
        try:
            res = self.scraper.get(url, timeout=15)
            if res.status_code != 200:
                log(f"Error status: {res.status_code}")
                return []
            
            soup = BeautifulSoup(res.text, 'html.parser')
            movies = []
            # Selectors
            items = soup.select('.item.movies, .item.tvshows')
            for item in items:
                title_tag = item.select_one('.data h3 a')
                if not title_tag: continue
                
                link = title_tag.get('href')
                title = title_tag.text.strip()
                poster_tag = item.select_one('.poster img')
                poster = poster_tag.get('src') if poster_tag else ""

                # Extract ID
                # Link: https://tv12.idlixku.com/movie-title/ -> movie-title
                if link and self.base_url in link:
                    id_part = link.replace(self.base_url, "").strip("/")
                    movies.append({
                        "id": f"movie:{id_part}",
                        "title": title,
                        "cover": poster,
                        "genre": "Movie"
                    })
            return movies
        except Exception as e:
            log(f"Exception in get_latest: {e}")
            return []

    def get_detail(self, id_raw):
        url = f"{self.base_url}/{id_raw}/"
        log(f"Fetching detail from {url}")
        try:
            res = self.scraper.get(url, timeout=15)
            if res.status_code != 200:
                return None
            
            soup = BeautifulSoup(res.text, 'html.parser')
            title = soup.select_one('.data h1').text.strip() if soup.select_one('.data h1') else "Unknown"
            synopsis = soup.select_one('.wp-content p').text.strip() if soup.select_one('.wp-content p') else ""
            poster = soup.select_one('.poster img').get('src') if soup.select_one('.poster img') else ""

            # Episodes (Server options for Movies)
            episodes = []
            # Check for players
            if soup.select_one('#playeroptionsul li'):
                episodes.append({
                    "id": f"movie:{id_raw}",
                    "index": 0,
                    "label": "Full Movie"
                })
            
            return {
                "drama": {
                    "id": f"movie:{id_raw}",
                    "title": title,
                    "description": synopsis,
                    "cover": poster,
                    "genre": "Movie",
                    "total_episodes": "1"
                },
                "episodes": episodes
            }
        except Exception as e:
            log(f"Exception in get_detail: {e}")
            return None

    def get_stream(self, id_raw):
        # 1. Get Detail Page to find player data
        url = f"{self.base_url}/{id_raw}/"
        try:
            res = self.scraper.get(url, timeout=15)
            soup = BeautifulSoup(res.text, 'html.parser')
            
            # Find first player
            option = soup.select_one('#playeroptionsul li')
            if not option:
                return None
            
            post_id = option.get('data-post')
            nume = option.get('data-nume')
            vtype = option.get('data-type')
            
            # 2. Hit Ajax
            data = {'action': 'doo_player_ajax', 'post': post_id, 'nume': nume, 'type': vtype}
            log(f"Posting to Ajax: {data}")
            
            res_ajax = self.scraper.post(self.ajax_url, data=data, timeout=10)
            txt = res_ajax.text
            final_url = ""

            # 3. Decrypt/Parse
            if '"ct":' in txt:
                # Encrypted
                decrypted_txt = None
                for key in self.keys:
                    dec = self.decrypt(txt, key)
                    if dec:
                        decrypted_txt = dec.replace('\\"', '"').strip('"')
                        log(f"Decrypted successfully with key: {key}")
                        break
                
                if decrypted_txt:
                    txt = decrypted_txt
                else:
                    log("Failed to decrypt content with any key.")
                    return None # Return None if decryption fails
            
            # Extract URL from JSON or HTML
            if txt.startswith('{'):
                try:
                    js = json.loads(txt)
                    if 'embed_url' in js: final_url = js['embed_url']
                    elif 'content' in js: txt = js['content']
                except: pass
            
            if not final_url and '<iframe' in txt:
                soup_iframe = BeautifulSoup(txt, 'html.parser')
                iframe = soup_iframe.select_one('iframe')
                if iframe: final_url = iframe.get('src')

            if not final_url: final_url = txt # Fallback

            # Final sanity check: if it still looks like encrypted json, fail
            if '{"ct":' in final_url:
                log("Final URL still looks encrypted. Failing.")
                return None

            return {
                "id": f"movie:{id_raw}",
                "chapter": {
                    "index": 0,
                    "video": {
                        "mp4": final_url
                    }
                }
            }
        except Exception as e:
            log(f"Exception in get_stream: {e}")
            return None

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print(json.dumps({"error": "No command"}))
        sys.exit(1)
    
    cmd = sys.argv[1]
    scraper = IdlixScraper()
    
    result = None
    if cmd == "latest":
        page = int(sys.argv[3]) if len(sys.argv) > 3 else 1
        result = scraper.get_latest(page)
    elif cmd == "detail":
        uid = sys.argv[3] # --url ID
        result = scraper.get_detail(uid)
    elif cmd == "stream":
        uid = sys.argv[3]
        result = scraper.get_stream(uid)
    
    print(json.dumps(result))
