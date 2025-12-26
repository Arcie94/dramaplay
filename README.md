# 🎬 DramaPlay - Vertical Drama Streaming Platform

DramaPlay is a comprehensive, full-stack video streaming platform designed for vertical drama consumption (Shorts/Reels style). Built for high performance and scalability, it features a robust Go backend, a modern Astro frontend, and a complete admin management suite.

## 🚀 Key Features

### 👤 User Experience (Frontend)

- **Cinematic Video Player**:
  - Custom-built HLS player with adaptive streaming.
  - Smart controls: Autoplay, Mute Toggle, Quality Selection.
  - Touch-optimized swipe navigation (Next/Prev Episode).
- **Interactive Community**:
  - **Comments**: Users can discuss episodes in real-time.
  - **User Avatars**: Comments display user profile photos.
  - **Edit/Delete Capabilities**: Full CRUD for own comments with inline editing.
  - **Premium UI**: Three-Dot context menus and custom dark-mode confirmation modals.
- **Personalization**:
  - **Resume Playback**: Automatically remembers timestamp and episode.
  - **My List**: Bookmark favorite dramas.
  - **Watch History**: Tracks viewing progress.
- **Authentication**:
  - Google OAuth Integration.
  - Standard Email/Password Login.
  - **Security**: "Change Password" feature with strict validation.
- **PWA Ready**: Installable as a native-like app on mobile and desktop.

### 🛠️ Admin Dashboard

- **Dashboard Overview**: Real-time stats on Users, Dramas, and System Health.
- **Content Management**:
  - **Dramas**: Create, Edit, Bulk Delete, and Manage Episodes.
  - **Image Tools**: Integrated **CropperJS** for uploading optimized Posters/Covers.
- **User Management**:
  - **Search**: Real-time debounced user search.
  - **Bulk Actions**: Select and delete multiple users efficiently.
- **System Settings**:
  - **Dynamic Branding**: Upload **Site Logo** and **Favicon** directly from the Admin Panel (updates globally instantly).
  - **Configuration**: Manage Google Client ID and Analytics IDs via GUI.

## 🏗️ Technical Architecture

### Backend (`/backend`)

- **Language**: Go (Golang)
- **Framework**: [Fiber](https://gofiber.io/) (High-performance HTTP framework)
- **Database**: PostgreSQL (Production-grade relational DB)
- **ORM**: GORM
- **Key Features**:
  - JWT Authentication.
  - Static File Serving (optimized for images/video).
  - **Migrator**: Custom utility to migrate data from SQLite to PostgreSQL.

### Frontend (`/frontend`)

- **Framework**: [Astro](https://astro.build/) (Hybrid: Server-Side Rendering + Static)
- **Styling**: TailwindCSS
- **Logic**: Vanilla JS + TypeScript for lightweight client interactivity.
- **Deployment**: Dockerized Node.js environment.

### Infrastructure

- **Docker Compose**: Orchestrates the entire stack (Backend, Frontend, Postgres, Nginx).
- **Nginx**: Reverse proxy for routing, load balancing, and SSL termination.

## 📦 Installation & Deployment

### Quick Start (Docker)

The recommended way to run the project (matches Production environment).

1.  **Clone the Repository**

    ```bash
    git clone https://github.com/Arcie94/dramaplay.git
    cd dramaplay
    ```

2.  **Run Services**
    ```bash
    # Build and start Backend, Frontend, Postgres, and Nginx
    sudo docker compose up -d --build
    ```
3.  **Access**
    - Frontend: `http://localhost` (or server IP)
    - Admin Panel: `http://localhost/admin`

### Manual Development Setup

**Backend**

```bash
cd backend
go mod tidy
go run main.go
# API runs on port 3000
```

**Frontend**

```bash
cd frontend
npm install
npm run dev
# App runs on port 4321
```

## 📖 Additional Documentation

- [**DEPLOYMENT.md**](./DEPLOYMENT.md): Detailed guide for deploying to **Proxmox** or Linux servers using Cloudflare Tunnels.
- [**DEVELOPMENT_LOG.md**](./DEVELOPMENT_LOG.md): Comprehensive history of features, changes, and technical decisions.
- [**DEPLOYMENT.md**](./DEPLOYMENT.md): Detailed guide for deploying to **Proxmox** or Linux servers using Cloudflare Tunnels.
- [**DEVELOPMENT_LOG.md**](./DEVELOPMENT_LOG.md): Comprehensive history of features, changes, and technical decisions (from V1.0 to V1.4).
- [**api_research.md**](./.agent/api_research.md): Technical deep-dive into API reverse-engineering attempts.

## ⚙️ Configuration

- **Environment Variables**: Managed via `docker-compose.yml` (DB connection, API URL).
- **Runtime Settings**: Site name, Logo, and OAuth keys are managed via the **Admin Settings** page.

## 📄 License

Private Project. All rights reserved.
