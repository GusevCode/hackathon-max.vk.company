# MAX Hackathon Project

Проект хакатона «Умный город» для MAX.

## Структура

```text
src/
  backend/                 Go backend and MAX ЖКХ Контроль bot
    cmd/echo-bot/          executable entrypoint
    internal/domain/        business entities
    internal/usecase/       application services and ports
    internal/infrastructure/ MAX and HTTP adapters
    go.mod
  frontend/                React + TypeScript + Vite status page (FSD)
    src/app/               app composition and global styles
    src/pages/             pages
    src/widgets/           page blocks
    src/features/          user actions
    src/entities/          domain UI/models
    src/shared/            reusable code
    dist/                  generated Vite output (not committed)
infra/
  docker/backend.Dockerfile
  docker/frontend.Dockerfile
  caddy/Caddyfile           HTTPS, static frontend and API proxy
.github/workflows/
compose.yaml
.golangci.yml
```

## Local development

1. Copy `.env.example` to `.env` and put the MAX bot token in it.
2. Run tests and lint locally:

   ```bash
   cd src/backend
   go test ./...
   golangci-lint run
   ```

   ```bash
   cd src/frontend
   npm ci
   npm run typecheck
   npm run lint
   npm test
   npm run build
   ```

3. Run the bot directly:

   ```bash
   go run ./cmd/echo-bot
   ```

4. Or run the bot through Docker:

   ```bash
   docker compose up --build bot
   ```

The bot uses Webhook transport. The `https` Compose profile publishes the React
status page at `https://max.conspiracy-team.ru/`, proxies `/healthz`, `/webhook`
and `/api/*` to Go, and runs Tarantool and MinIO for application data and photo
evidence.

## CI/CD direction

- Every push and pull request checks Go and frontend formatting/types, tests,
  linters, and builds the applications in GitHub Actions.
- A separate manually triggered workflow builds the backend and frontend images,
  publishes them to GHCR, and deploys them over SSH to the VPS.
- The VPS runs pinned images through `docker compose`; the MAX token exists only
  as a server-side secret in `.env`.

See [`docs/MAX_DEVELOPMENT.md`](docs/MAX_DEVELOPMENT.md) for MAX API, Webhook, mini app, HTTPS and security notes.

See [`docs/INFRASTRUCTURE.md`](docs/INFRASTRUCTURE.md) for the development, CI/CD, VPS and domain plan.

See [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) for the frontend FSD and
backend clean architecture rules.

See [`docs/DEPLOY_ECHO_BOT.md`](docs/DEPLOY_ECHO_BOT.md) for the complete first-deployment checklist.

See [`docs/MVP_TESTING.md`](docs/MVP_TESTING.md) for the role/task/photo workflow
used to test stages 1–4.

## GitHub Actions deployment setup

CI runs on every push and pull request. Deploy is a separate manual workflow:
GitHub builds both Docker images, publishes them to GHCR and restarts the `bot`
and `caddy` services on the VPS over SSH.

Before the first deployment, create these GitHub repository/environment secrets:

- `VPS_HOST` — VPS hostname or IP;
- `VPS_USER` — non-root SSH user with Docker permissions;
- `VPS_APP_DIR` — absolute application directory on the VPS;
- `VPS_SSH_PRIVATE_KEY` — private key used by the runner;
- `VPS_KNOWN_HOSTS` — output of `ssh-keyscan -H <host>` collected independently;
- `GHCR_USERNAME` and `GHCR_READ_TOKEN` — credentials for pulling the private GHCR image (omit if the package is public).

On the VPS, create `${VPS_APP_DIR}/.env` manually once and keep the real `MAX_BOT_TOKEN` there. The deploy workflow never copies secrets to the machine; it only updates the compose file and image tag.

Optionally add the GitHub repository variable `MAX_BOT_URL` after the public bot
username is known. Its value should be the complete URL, for example
`https://max.ru/<username>`; the frontend shows the bot button only when this
variable exists.

To deploy, open **Actions → Deploy → Run workflow**, select a ref, and run it. The VPS does not need a shell session for ordinary updates.
