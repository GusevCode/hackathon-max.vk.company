# MAX Hackathon Project

Проект хакатона «Умный город» для MAX.

## Структура

```text
src/
  backend/                 Go backend and MAX echo bot
    cmd/echo-bot/          executable entrypoint
    internal/               application packages
    go.mod
  frontend/                reserved for MAX mini app
infra/
  docker/backend.Dockerfile
  caddy/Caddyfile           HTTPS/reverse proxy config
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

3. Run the bot directly:

   ```bash
   go run ./cmd/echo-bot
   ```

4. Or run everything through Docker:

   ```bash
   docker compose up --build bot
   ```

The first echo version deliberately uses Long Polling. The domain and HTTPS profile are already prepared for health checks and the later Webhook/mini-app stage.

## CI/CD direction

- Every push and pull request runs formatting checks, `go test -race`, coverage and `golangci-lint` in GitHub Actions.
- A separate manually triggered workflow will build the image, publish it to GHCR and deploy over SSH to the VPS.
- The VPS will run the pinned image through `docker compose`; the MAX token will exist only as a server-side secret in `.env`.

See [`docs/MAX_DEVELOPMENT.md`](docs/MAX_DEVELOPMENT.md) for MAX API, Webhook, mini app, HTTPS and security notes.

See [`docs/INFRASTRUCTURE.md`](docs/INFRASTRUCTURE.md) for the development, CI/CD, VPS and domain plan.

See [`docs/DEPLOY_ECHO_BOT.md`](docs/DEPLOY_ECHO_BOT.md) for the complete first-deployment checklist.

## GitHub Actions deployment setup

CI runs on every push and pull request. Deploy is a separate manual workflow: GitHub builds the Docker image, publishes it to GHCR and restarts the `bot` service on the VPS over SSH.

Before the first deployment, create these GitHub repository/environment secrets:

- `VPS_HOST` — VPS hostname or IP;
- `VPS_USER` — non-root SSH user with Docker permissions;
- `VPS_APP_DIR` — absolute application directory on the VPS;
- `VPS_SSH_PRIVATE_KEY` — private key used by the runner;
- `VPS_KNOWN_HOSTS` — output of `ssh-keyscan -H <host>` collected independently;
- `GHCR_USERNAME` and `GHCR_READ_TOKEN` — credentials for pulling the private GHCR image (omit if the package is public).

On the VPS, create `${VPS_APP_DIR}/.env` manually once and keep the real `MAX_BOT_TOKEN` there. The deploy workflow never copies secrets to the machine; it only updates the compose file and image tag.

To deploy, open **Actions → Deploy → Run workflow**, select a ref, and run it. The VPS does not need a shell session for ordinary updates.
