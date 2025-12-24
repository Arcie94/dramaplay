export async function GET() {
    try {
        const response = await fetch('http://localhost:3000/api/sitemap');
        const json = await response.json();
        const dramas = json.data || [];
        const domain = "https://dramaplay.online";

        const xml = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url>
    <loc>${domain}/</loc>
    <changefreq>daily</changefreq>
    <priority>1.0</priority>
  </url>
  ${dramas.map((drama: any) => `
  <url>
    <loc>${domain}/detail/${drama.bookId}</loc>
    <changefreq>weekly</changefreq>
    <priority>0.8</priority>
  </url>
  `).join('')}
</urlset>`;

        return new Response(xml, {
            headers: {
                'Content-Type': 'application/xml',
            },
        });
    } catch (e) {
        return new Response('Error generating sitemap', { status: 500 });
    }
}
