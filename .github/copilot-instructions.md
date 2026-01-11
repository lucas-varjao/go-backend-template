# Copilot Instructions for GoSvelteKit

## Project Overview

GoSvelteKit is a template/base project with a Golang backend and SvelteKit (Svelte 5) frontend, designed to kickstart new projects with authentication and basic functionality already implemented.

## Backend Specifications

-   **Framework**: Golang with Gin
-   **Authentication**: Session-based (pluggable adapters inspired by Lucia Auth)
-   **Database**: SQLite (via GORM)
-   **ORM**: GORM for database operations
-   **API Docs**: Swaggo
-   **Logging**: Zap
-   **Config**: Viper

### Authentication Architecture

The auth system uses a pluggable adapter pattern:

```
internal/auth/
├── interfaces.go      # UserAdapter, SessionAdapter interfaces
├── auth_manager.go    # Central AuthManager
└── adapter/gorm/      # GORM implementation
```

-   **Sessions** are stored in the database (not JWTs)
-   Login returns `session_id` (not `access_token`/`refresh_token`)
-   Auth via `Authorization: Bearer {session_id}` header or `session_id` cookie

## Frontend Specifications

-   **Framework**: SvelteKit (Svelte 5)
-   **Styling**: TailwindCSS (utility-first)
-   **Runtime**: Bun
-   **UI Components**: shadcn-svelte
-   **Mode**: Dark mode only
-   Always use Svelte 5 with the new runes API (`$state`, `$derived`, `$props`)

### Auth Store

```typescript
// User fields are snake_case
interface User {
    id: string;
    identifier: string; // username
    email: string;
    display_name: string;
    role: string;
    active: boolean;
}
```

## Design Preferences

-   Professional and minimalist design
-   Basic animations for enhanced UX
-   `slate-950` as primary background color

## Code Style & Conventions

-   Backend Go code follows standard Go idioms and project layout best practices
-   Frontend TypeScript uses 4-space indentation and single quotes
-   CSS is primarily written as Tailwind utility classes
-   Component files use .svelte extension and follow SvelteKit conventions

## Development Workflow

-   Monorepo with `backend/` and `frontend/` directories
-   Dev requires running both servers: `go run cmd/server/server.go` and `bun run dev`
-   Follow Conventional Commits for commit messages

## Deployment

-   Docker containers for deployment
-   Environment variables for configuration
