# Инфраструктура разработки и деплоя

## Цель

Поток разработки должен выглядеть так:

```text
git push / Pull Request
        |
        v
GitHub Actions: gofmt -> vet -> tests (-race) -> golangci-lint -> Docker build
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

- `src/backend` — Go-модуль с echo-ботом MAX на официальном SDK.
- `src/frontend` — место для будущего mini app.
- `compose.yaml` — запуск бота и HTTPS-профиля Caddy.
- `infra/docker/backend.Dockerfile` — multi-stage образ Go 1.24.
- `infra/caddy/Caddyfile` — reverse proxy для домена; перед первым деплоем нужно заменить placeholder домена.
- `.golangci.yml` — конфигурация golangci-lint 2.x.
- `.github/workflows/ci.yml` — проверки push/PR.
- `.github/workflows/deploy.yml` — ручная сборка, публикация в GHCR и деплой на VPS.
- `/healthz` — health endpoint backend на внутреннем порту `8080`.

Пока бот получает сообщения через Long Polling, поэтому домен и публичный HTTPS не нужны для локальной разработки.

## Этапы

### Этап 1. Локальный echo-бот

1. Скопировать `.env.example` в `.env`.
2. Заполнить `MAX_BOT_TOKEN`.
3. Запустить `go test -race ./...` и `go vet ./...` из `src/backend`.
4. Запустить `go run ./cmd/echo-bot` или `docker compose up --build bot`.
5. Написать боту в MAX и проверить, что он повторяет текст.

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

1. Создать DNS `A`/`AAAA` запись на IP VPS, например `bot.example.ru`.
2. Заменить placeholder `bot.example.com` в `infra/caddy/Caddyfile`.
3. Включить профиль `https`: `docker compose --profile https up -d`.
4. Проверить `/healthz` через HTTPS.
5. Когда backend будет готов принимать webhook, переключить бота с Long Polling на webhook.

### Этап 4. Mini app (если понадобится)

- собрать frontend в `src/frontend`;
- отдавать статику через Caddy по отдельному пути/поддомену;
- добавить серверную проверку `initData`;
- подключить URL mini app к боту в MAX;
- расширить deploy workflow фронтенд-артефактом.

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

`MAX_BOT_TOKEN` не нужен GitHub Actions: он остаётся только в `.env` на VPS и локальной машине разработчика.

## Правила эксплуатации

- Не использовать `root` для deploy.
- Не отключать проверку `known_hosts` и не использовать `StrictHostKeyChecking=no`.
- Не хранить `.env`, SSH-ключи и токены в GitHub artifacts.
- Использовать immutable image tag по commit SHA; `latest` оставлять только как удобный alias.
- После deploy проверять health endpoint и логи контейнера.
- Бэкапить только появившиеся позже данные/БД; на первом этапе БД нет.
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
