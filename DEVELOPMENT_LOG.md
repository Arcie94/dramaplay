# 📜 Development Journey & Changelog

This document tracks the major development milestones, features added, and technical challenges resolved in the evolution of DramaPlay.

## 📱 Version 1.4 - Mobile Expansion & Resilience (Dec 2025)

### 1. 🤖 Mobile Application (Capacitor)

- **Goal**: Create a native-like experience for Android users.
- **Implementation**:
  - Integrated **CapacitorJS** into the Astro frontend.
  - Configured `server.url` wrapper mode for seamless updates without app store re-submissions.
  - **Features**:
    - Custom App Icon (Black/Cinematic theme).
    - Fixed touch gestures (Scrubbing/Seeking) on mobile screens.
    - "Download App" button in the Profile menu.

### 2. 📺 Persistent Fullscreen (SPA Navigation)

- **Problem**: Changing episodes caused a full page reload, exiting fullscreen mode and disrupting the binge-watching experience.
- **Solution**: Implemented a **Single Page Application (SPA)** architecture specifically for the Video Player.
- **Tech**:
  - Intercepted Next/Prev clicks and `video.ended` events.
  - Used `history.pushState` to update URLs dynamically.
  - Fetched stream data via `fetch` and updated the video source without tearing down the DOM.
- **Result**: Users can now watch an entire series in fullscreen uninterrupted.

### 3. 🛡️ API Strategy & Research

- **Context**: Moving away from fragile internal scrapers to a more robust upstream source.
- **Migration**: Successfully migrated all content endpoints (`Trending`, `Detail`, `Stream`) to use the `dramabox-api-rho` proxy.
- **Research**: Conducted deep-dive experiments to bypass the proxy and hit `sapi.dramaboxdb.com` directly.
  - **Finding**: The API uses a proprietary Signature Algorithm (embedded in the native APK).
  - **Decision**: Direct access is not feasible without advanced reverse engineering. The Proxy approach remains the most stable and maintainable solution.

## 🚀 Version 1.3 - Admin Power Ups & Monetization (Dec 2024)

### 1. 🛡️ Admin Dashboard Enhancements

The Admin Dashboard received a major overhaul to improve usability and system observability.

- **System Activity Logs**:

  - **Feature**: A real-time log panel to monitor background processes (like Mass Ingest and Deduplication).
  - **Tech**: Implemented `SystemLog` model in Go (GORM) and a `GET /api/admin/logs` endpoint.
  - **UI**: Added an auto-refreshing "System Activity" panel in the Admin Home.
  - **Challenge**: The background scripts (`go run ...`) failed in Docker because the Go toolchain is missing in the runtime container.
  - **Solution**: Modified `Dockerfile` to use a **multi-stage build**, compiling `cmd/ingest` and `cmd/dedup` into standalone binaries (`./ingest`, `./dedup`) during the build phase.

- **Advanced Filtering & Sorting**:

  - **Feature**: Admins can now filter dramas by **Genre** (fuzzy search) and Sort by:
    - Newest / Oldest
    - Title A-Z / Z-A
  - **Implementation**: Dynamic query construction in `GetAdminDramas` handler.

- **UI/UX Polish**:
  - **Toasts**: Replaced unprofessional `alert()` popups with a custom **Toast Notification System** (Slide-in animations, Success/Error states).
  - **Modals**: Replaced `confirm()` dialogs with themed CSS modals.

### 2. 💸 Monetization Layer

Added features to support the sustainability of the platform.

- **Premium Donate Button**:
  - **Location**: User Profile Page.
  - **Design**: A standout "Premium Card" design with a Golden/Amber gradient, animated pulse on hover, and distinct shadow to attract attention.
  - **Functionality**: Opens a modal displaying the **QRIS Code** for easy donations.
  - **Asset Management**: QRIS image is stored in `frontend/public/qris.jpg` and served statically.

### 3. 🔧 Technical Refactoring

- **Docker Architecture**:

  - Updated `backend/Dockerfile` to compile helper tools.
  - Ensured lightweight production images (`alpine`).

- **Codebase Organization**:
  - Centralized Admin Handlers in `backend/handlers/admin.go`.
  - Created `models/log.go` for better concern separation.

---

## 📅 Previous Versions

### Version 1.2 - Social Features

- **Comments System**: Interactive comments on episodes.
- **User Profiles**: Avatars and Name management.
- **Watch History**: Resume playback functionality.

### Version 1.0 - Core Platform

- **Vertical Player**: Custom HLS Video Player.
- **Infinite Scroll**: Homepage carousels.
- **Authentication**: JWT & Connect (Google).
