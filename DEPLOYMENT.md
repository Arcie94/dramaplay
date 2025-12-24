# Deployment Guide for DramaBang (Proxmox/Docker)

This guide covers how to deploy the DramaBang application on a Proxmox server using Docker and make it accessible via the internet.

## 📋 Prerequisites

1.  **Proxmox Server**: Running and accessible.
2.  **Linux Environment**: An Ubuntu/Debian VM or LXC container running on Proxmox.
3.  **Internet Access & Domain** (Optional but recommended): A domain name for public access (free from Cloudflare).

---

## 🚀 Step 1: Prepare the Environment (Ubuntu/Debian)

1.  Update the system:
    ```bash
    sudo apt update && sudo apt upgrade -y
    ```
2.  Install Docker & Docker Compose:
    ```bash
    # Install Docker
    curl -fsSL https://get.docker.com -o get-docker.sh
    sh get-docker.sh

    # Verify installation
    docker --version
    docker compose version
    ```

---

## 📂 Step 2: Deploy the App

1.  **Clone the Repository**:
    ```bash
    git clone https://github.com/Arcie94/dramaplay.git
    cd dramabang
    ```

2.  **Start the Services**:
    ```bash
    sudo docker compose up -d --build
    ```

    -   This will build the Backend and Frontend images.
    -   It will start Nginx as a reverse proxy on **Port 80**.

3.  **Verify Access**:
    -   Open your browser and visit: `http://<YOUR-SERVER-IP>`
    -   You should see the DramaBang homepage.
    -   The backend API is accessible internally, and persistent data/uploads are stored in the `./data` and `./uploads` folders on the host.

---

## 🌐 Step 3: Go Public (Cloudflare Tunnel)

The safest way to expose your local server to the internet without opening ports on your router is using **Cloudflare Tunnel**.

1.  **Sign up for Cloudflare Zero Trust** (Free).
2.  **Create a Tunnel** in the Cloudflare Dashboard (Networks > Tunnels).
3.  **Install the Connector** on your Proxmox server (copy the command provided by Cloudflare).
4.  **Configure a Public Hostname**:
    -   **Subdomain**: `tv.yourdomain.com` (or whatever you prefer).
    -   **Service**:
        -   Type: `HTTP`
        -   URL: `localhost:80` (This points to our Nginx container).

**Done!** Your app is now accessible via `https://tv.yourdomain.com` with auto HTTPS.

---

## 🛠️ Management & Updates

-   **Update App**:
    ```bash
    git pull
    sudo docker compose up -d --build
    ```

-   **View Logs**:
    ```bash
    sudo docker compose logs -f
    ```

-   **Backup Data**:
    -   Database: `./data/dramabang.db`
    -   Uploads: `./uploads` folder.
