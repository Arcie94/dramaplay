# 🎬 DramaPlay - Vertical Drama Streaming Platform

DramaPlay is a modern, responsive web application for streaming vertical dramas. It features a robust admin panel for content management, user authentication via Google OAuth, and a PWA-ready frontend.

## ✨ Features

### 👤 User Experience
-   **Seamless Streaming**: HLS video playback with adaptive quality.
-   **Google Login**: Secure and fast authentication using Google OAuth.
-   **Progress Tracking**: Automatically saves watch history and resumes playback.
-   **"My List"**: Bookmark favorite dramas for later watch.
-   **PWA Support**: Installable on mobile and desktop for a native app-like experience.
-   **Responsive Design**: Optimized for both mobile vertical viewing and desktop browsing.

### 🛠️ Admin Dashboard
-   **Drama Management**: content management (CRUD) for dramas and episodes.
-   **User Management**: Bulk delete, search, and view registered users.
-   **Site Settings**:
    -   **Dynamic Branding**: Upload and crop Site Logo and Favicon directly from the admin panel.
    -   **Configurable IDs**: Manage Google Client ID and GA4 Measurement ID without restarting.
-   **Security**: Admin-specific authentication and role management.

## 🏗️ Tech Stack

### Backend
-   **Language**: Go (Golang)
-   **Framework**: [Fiber](https://gofiber.io/) (Fast HTTP web framework)
-   **Database**: SQLite (via GORM)
-   **Key Libraries**:
    -   `gorm.io/gorm`: ORM for database interactions.
    -   `github.com/golang-jwt/jwt`: JSON Web Tokens for auth.

### Frontend
-   **Framework**: [Astro](https://astro.build/) (Server-Side Rendering mode)
-   **Styling**: TailwindCSS
-   **State/Logic**: Vanilla JS & TypeScript
-   **Key Libraries**:
    -   `cropperjs`: For image editing (Logo/Favicon upload).
    -   `hls.js`: For video streaming.

## 🚀 Getting Started

### Prerequisites
-   [Go 1.21+](https://go.dev/dl/)
-   [Node.js 20+](https://nodejs.org/)

### 1. Backend Setup

```bash
cd backend

# Install dependencies
go mod tidy

# Run the server (default port: 3000)
go run main.go
```

The backend API will be available at `http://localhost:3000`.

### 2. Frontend Setup

Open a new terminal:

```bash
cd frontend

# Install dependencies
npm install

# Run development server
npm run dev
```

The application will be accessible at `http://localhost:4321`.

## ⚙️ Configuration

### Admin Account
The first user or a manually seeded user can be set as Admin. 
(Check `backend/seeds/data.go` or database for initial credentials if applicable).

### Environment Variables
Currently, critical settings like `GOOGLE_CLIENT_ID` and `GA_MEASUREMENT_ID` are managed dynamically via the **Admin > Settings** page, stored in the database.

## 📱 PWA & Mobile
This project is configured as a Progressive Web App.
-   **Manifest**: Located at `frontend/public/manifest.json`.
-   **Service Worker**: `frontend/public/sw.js` handles caching strategies (Network-First for HTML, Cache-First for assets).

## 📄 License
Private Project. All rights reserved.
