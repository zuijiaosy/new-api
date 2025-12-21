# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

New API is a next-generation large model gateway and AI asset management system based on Go (backend) and React (frontend). It's a fork of One API with additional features and improvements. The system provides API relay functionality for various AI models including OpenAI, Claude, Gemini, and many others.

## Development Commands

### Backend (Go)
- **Run backend**: `go run main.go`
- **Build**: `go build -o new-api main.go`

### Frontend (React)
- **Install dependencies**: `cd web && bun install`
- **Development server**: `cd web && bun run dev`
- **Build production**: `cd web && DISABLE_ESLINT_PLUGIN='true' bun run build`
- **Lint check**: `cd web && bun run lint`
- **Lint fix**: `cd web && bun run lint:fix`
- **ESLint**: `cd web && bun run eslint`
- **ESLint fix**: `cd web && bun run eslint:fix`

### Full Stack Development
- **Build frontend and start backend**: `make all`
- **Frontend only**: `make build-frontend`
- **Backend only**: `make start-backend`

### Testing
- **API performance test**: `./bin/time_test.sh <domain> <key> <count> [<model>]`

## Project Architecture

### Backend Structure (Go)
- **`main.go`**: Entry point with server initialization, middleware setup, and resource initialization
- **`common/`**: Shared utilities (database, Redis, crypto, validation, rate limiting)
- **`constant/`**: Application constants (API types, channels, endpoints)
- **`controller/`**: HTTP request handlers (relay, channels, users, billing)
- **`dto/`**: Data transfer objects for API requests/responses
- **`middleware/`**: HTTP middleware (auth, logging, rate limiting)
- **`model/`**: Database models and data access layer
- **`relay/`**: Core relay functionality for different AI providers
- **`router/`**: HTTP route definitions
- **`service/`**: Business logic and external service integrations

### Frontend Structure (React)
- **`web/src/`**: React application source code
- **`web/dist/`**: Built frontend assets (embedded in Go binary)
- Built with Vite, uses Semi UI components, TailwindCSS for styling

### Key Features Architecture
- **Multi-provider relay**: Supports OpenAI, Claude, Gemini, Azure, AWS Bedrock, and 40+ other providers
- **Channel management**: Weighted load balancing, failover, and health monitoring
- **User management**: Token-based authentication, quota management, group permissions
- **Billing system**: Pay-per-use, prepaid credits, Stripe integration
- **Cache layer**: Redis for performance optimization and rate limiting
- **Database**: SQLite (default), MySQL, or PostgreSQL support

### Configuration
- **Environment variables**: Set in `.env` file or system environment
- **Database**: Configured via `SQL_DSN` environment variable
- **Redis**: Optional cache layer via `REDIS_CONN_STRING`
- **Session management**: Uses `SESSION_SECRET` for multi-instance deployment

### Development Notes
- The frontend build is embedded into the Go binary using `//go:embed`
- The system supports both standalone and Docker deployment
- Channel relay logic is in `relay/` packages organized by provider
- Database migrations are handled automatically on startup
- Real-time features use WebSockets for live updates

### Important Files
- **`go.mod`**: Go dependencies and module definition
- **`Makefile`**: Build automation for frontend/backend
- **`web/package.json`**: Frontend dependencies and scripts
- **`common/init.go`**: Environment variable loading and initialization
- **`model/`**: Database schema and ORM models
- **`relay/`**: Provider-specific API integration logic

## 对话语言
一直使用中文进行对话
## 其他注意事项
- 编写的测试文件使用后记得删除
- 代码中的注释全部使用中文
- 我是 java 开发者转 go 的，尽量使用 java 开发工程师能够看懂的代码语法修改或者新增代
- 编写的测试go 代码测试完成后需要删除
- 我们自己修改或者新增功能尽量不影响原来的代码，方便我后续更新合并开源项目中的代码