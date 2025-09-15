# Gemini Project: DuckChat

This document provides a comprehensive overview of the DuckChat project, designed to be used as a context for Gemini AI.

## Project Overview

DuckChat is a minimal chat service with a unified REST API. The backend is implemented in Go and utilizes a clean architecture. The frontend is server-side rendered using `templ` components and styled with Tailwind CSS.

### Key Technologies

*   **Backend:** Go
*   **Frontend:** templ (server-side rendering), Tailwind CSS
*   **API:** REST with JSON-RPC 2.0 body
*   **Real-time Communication:** Server-Sent Events (SSE)
*   **Database:** Redis Stack (RedisJSON + RediSearch)
*   **Routing:** gorilla/mux
*   **Authentication:** JWT

### Architecture

The project follows a clean architecture pattern, separating concerns into the following layers:

*   **Presentation:** HTTP handlers, SSE endpoints, and UI components.
*   **Application:** Use cases and business logic orchestration.
*   **Domain:** Business logic and entities.
*   **Infrastructure:** Redis and other external services.

## Building and Running

### Backend

To run the backend server, execute the following command from the `backend` directory:

```bash
go run ./cmd/server/main.go
```

The server will start on port `8080` by default.

### Frontend

The frontend assets are built using Tailwind CSS. To build the CSS, run the following command from the `backend/web/static` directory:

```bash
npm run build-css
```

For development, you can use the following command to watch for changes and automatically rebuild the CSS:

```bash
npm run dev
```

## Development Conventions

*   **Go:** The Go code follows standard Go conventions.
*   **Templ:** UI components are written using the `templ` library, which combines Go code with HTML-like syntax.
*   **Styling:** Styling is done using Tailwind CSS.
*   **API:** The API follows the JSON-RPC 2.0 specification.
*   **Commits:** Commit messages should be clear and concise.
