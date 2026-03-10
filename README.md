# Streamcore


real-time market data streaming


## Getting Started

### Prerequisites

- [Docker](https://www.docker.com/) and Docker Compose
- [Node.js](https://nodejs.org/) (for frontend dev only)
- [Go 1.18+](https://go.dev/) (for backend dev only)

### Setup

1. Clone the repository:
   ```bash
   git clone <repo-url>
   cd streamcore-v2
   ```

2. Copy the environment template and fill in your values:
   ```bash
   cp .env.example .env
   ```

3. Start the full stack:
   ```bash
   docker-compose up --build
   ```

4. Open `http://localhost:<NGINX_PORT>` in your browser.

### Development

```bash
# Frontend dev server (hot reload)
cd web && npm start

# Run frontend tests
cd web && npm test
```

#
