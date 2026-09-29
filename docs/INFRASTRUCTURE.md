# Инфраструктура разработки и деплоя

## Цель

Поток разработки должен выглядеть так:

```text
git push / Pull Request
        |
        v
GitHub Actions: Go + frontend tests/linters/builds -> Docker build
        |
        +---- merge в main
        |
        +---- Actions → Deploy → Run workflow
                         |
                         v
                 GHCR image -> SSH -> VPS -> docker compose pull/up
```

Секреты MAX и SSH-ключи не хранятся в Git и не передаются в Docker image.

## Что уже подготовлено

- `src/backend` — Go-модуль MAX-бота на чистой архитектуре.
- `src/frontend` — React + TypeScript + Vite статусная страница.
- `compose.yaml` — запуск бота и HTTPS-профиля Caddy.
- `infra/docker/backend.Dockerfile` — multi-stage образ Go 1.25.
- `infra/docker/frontend.Dockerfile` — сборка Vite и runtime-образ Caddy.
- `infra/caddy/Caddyfile` — HTTPS, статический frontend и reverse proxy `/api/*`.
- `.golangci.yml` — конфигурация golangci-lint 2.x.
- `.github/workflows/ci.yml` — проверки push/PR.
- `.github/workflows/deploy.yml` — ручная сборка двух образов, публикация в GHCR и деплой на VPS; при деплое перезапускается Tarantool для применения схемы.
- `/healthz` — health endpoint backend на внутреннем порту `8080`.
- `/webhook` — защищённый endpoint MAX Webhook на публичном HTTPS-домене.
- `tarantool` — один экземпляр БД с начальной схемой из `infra/tarantool/init.lua`.
- `seaweedfs` — локальное S3-совместимое хранилище фотографий (`chrislusf/seaweedfs:4.47`).
- `nats` — брокер событий для межмодульных уведомлений (`nats:2.11-alpine`,
  JetStream включён, данные в volume `nats_data`).

Бот получает события через Webhook. Для production обязательны DNS, HTTPS и
`PUBLIC_BASE_URL`, указывающий на домен VPS.

Перед первым запуском SeaweedFS в `.env` на VPS должны быть заданы
`OBJECT_STORAGE_ACCESS_KEY`, `OBJECT_STORAGE_SECRET_KEY`,
`OBJECT_STORAGE_ENDPOINT=seaweedfs:8333` и `OBJECT_STORAGE_BUCKET`.
Брокер доступен приложению по `MESSAGE_BROKER_URL=nats://nats:4222`.

## Этапы

### Этап 1. Локальный бот и Webhook

1. Скопировать `.env.example` в `.env`.
2. Заполнить `MAX_BOT_TOKEN`.
3. Запустить `go test -race ./...` и `go vet ./...` из `src/backend`.
4. Запустить `go run ./cmd/control-bot` или `docker compose up --build bot`.
5. Запустить Tarantool и SeaweedFS через Docker Compose.
6. Проверить регистрацию `/webhook` через MAX API и отправить боту тестовое сообщение.

### Этап 2. Первый VPS-деплой без домена

На VPS один раз вручную подготовить:

- Docker Engine и Docker Compose plugin;
- отдельного пользователя `deploy` с правом запускать Docker;
- каталог приложения, например `/opt/max-hackathon`;
- файл `/opt/max-hackathon/.env` с реальным `MAX_BOT_TOKEN`;
- SSH-ключ GitHub Actions в `authorized_keys`;
- firewall: SSH только с нужных адресов/через выбранную политику.

Дальше обычный deploy выполняется из GitHub Actions без SSH-сеанса разработчика.

### Этап 3. Домен и HTTPS

После покупки домена:

1. Создать DNS `A` запись `max.conspiracy-team.ru` на публичный IP VPS.
2. Убедиться, что `max.conspiracy-team.ru` указан в `infra/caddy/Caddyfile`.
3. Включить профиль `https`: `docker compose --profile https up -d`.
4. Проверить главную страницу и `/healthz` через HTTPS.
5. Проверить, что MAX Webhook зарегистрирован и принимает события.

### Этап 4. Расширение до mini app (если понадобится)

- развить существующий React frontend в интерфейс mini app;
- обращаться к Go через same-origin `/api/*`, уже настроенный в Caddy;
- добавить серверную проверку `initData`;
- подключить URL mini app к боту в MAX;
- зарегистрировать URL mini app в MAX.

## GitHub secrets

Рекомендуется создать отдельное GitHub Environment `production` и хранить секреты там:

| Secret | Назначение |
| --- | --- |
| `VPS_HOST` | IP или DNS VPS |
| `VPS_USER` | непривилегированный deploy-пользователь |
| `VPS_APP_DIR` | абсолютный каталог приложения |
| `VPS_SSH_PRIVATE_KEY` | приватный ключ GitHub Actions |
| `VPS_KNOWN_HOSTS` | заранее проверенная строка `ssh-keyscan` |
| `GHCR_USERNAME` | пользователь/robot account для pull из GHCR |
| `GHCR_READ_TOKEN` | token только с `read:packages` |

`MAX_BOT_TOKEN`, `MAX_WEBHOOK_SECRET`, `POLZA_AI_API_KEY`, пароли Tarantool и
SeaweedFS не нужны GitHub Actions: они остаются только в `.env` на VPS и
локальной машине разработчика.

## Правила эксплуатации

- Не использовать `root` для deploy.
- Не отключать проверку `known_hosts` и не использовать `StrictHostKeyChecking=no`.
- Не хранить `.env`, SSH-ключи и токены в GitHub artifacts.
- Использовать immutable image tag по commit SHA; `latest` оставлять только как удобный alias.
- После deploy проверять health endpoint и логи контейнера.
- Настроить резервное копирование Tarantool и SeaweedFS после появления данных.
- Для production MAX использовать webhook и доверенный TLS-сертификат.

### TLS-сертификат MAX API

`platform-api2.max.ru` использует цепочку `Russian Trusted CA`, которой нет в
стандартном Mozilla-наборе сертификатов Alpine. Корневой сертификат добавлен в
`infra/certs/russian_trusted_root_ca_pem.crt` и устанавливается в runtime-образе
через `update-ca-certificates`. SHA-256 отпечаток сертификата:
`D26D2D0231B7C39F92CC738512BA54103519E4405D68B5BD703E9788CA8ECF31`.
Промежуточный сертификат также добавлен в `infra/certs`; его SHA-1
отпечаток: `335D43F53451B781535FF3882DF713D3C14F8A01`.

Источник сертификата — официальный портал Госуслуг / `gu-st.ru`. Отключать
проверку TLS (`InsecureSkipVerify`) нельзя.

## Что нужно получить от владельца проекта перед первым deploy

- купленное доменное имя (когда решим, что нужен webhook/mini app);
- IP/домен VPS и ОС;
- способ доступа по SSH и готовый deploy-пользователь;
- решение, будет ли GHCR-образ приватным;
- токен MAX для локальной проверки и отдельно подтверждение, что он не попал в чат/репозиторий/логи.
