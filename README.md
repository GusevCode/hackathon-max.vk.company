# Проект хакатона MAX

Проект хакатона «Умный город» для MAX.

## Структура

```text
src/
  backend/                 серверная часть на Go и бот «ЖКХ Контроль» для MAX
    cmd/control-bot/       точка запуска приложения
    internal/domain/       бизнес-сущности
    internal/usecase/      прикладные сценарии и порты
    internal/infrastructure/ адаптеры MAX и HTTP
    go.mod
  frontend/                статусная страница на React + TypeScript + Vite (FSD)
    src/app/               композиция приложения и глобальные стили
    src/pages/             страницы
    src/widgets/           блоки страниц
    src/features/          действия пользователя
    src/entities/          UI-модели предметной области
    src/shared/            переиспользуемый код
    dist/                  результат сборки Vite (не коммитится)
infra/
  docker/backend.Dockerfile
  docker/frontend.Dockerfile
  caddy/Caddyfile           HTTPS, статический frontend и прокси API
.github/workflows/
compose.yaml
.golangci.yml
```

## Локальная разработка

1. Скопируйте `.env.example` в `.env` и добавьте в него токен MAX-бота.
2. Запустите локальные тесты и линтеры:

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

3. Запустите бота напрямую:

   ```bash
   go run ./cmd/control-bot
   ```

4. Либо запустите бота через Docker:

   ```bash
   docker compose up --build bot
   ```

Бот использует транспорт Webhook. Профиль Compose `https` публикует React-
статусную страницу по адресу `https://max.conspiracy-team.ru/`, проксирует
`/healthz`, `/webhook` и `/api/*` в Go-сервис, а также запускает Tarantool и
SeaweedFS для данных приложения и фотографий-доказательств.

## Направление CI/CD

- Каждый push и pull request проверяет форматирование и типы Go и frontend,
  запускает тесты, линтеры и сборку приложений в GitHub Actions.
- Отдельный workflow с ручным запуском собирает образы backend и frontend,
  публикует их в GHCR и разворачивает на VPS по SSH.
- VPS запускает зафиксированные образы через `docker compose`; токен MAX
  хранится только как секрет на сервере в `.env`.

См. [`docs/MAX_DEVELOPMENT.md`](docs/MAX_DEVELOPMENT.md): API MAX, Webhook,
mini app, HTTPS и безопасность.

См. [`docs/INFRASTRUCTURE.md`](docs/INFRASTRUCTURE.md): разработка, CI/CD, VPS
и план настройки домена.

См. [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md): FSD frontend и правила
чистой архитектуры backend.

См. [`docs/DEPLOY_BOT.md`](docs/DEPLOY_BOT.md): полный чек-лист первого деплоя.

См. [`docs/MVP_TESTING.md`](docs/MVP_TESTING.md): workflow ролей, заданий и
фотографий для проверки этапов 1–4.

## Настройка деплоя через GitHub Actions

CI запускается на каждом push и pull request. Деплой выполняется отдельным
workflow вручную: GitHub собирает Docker-образы, публикует их в GHCR и
перезапускает сервисы `bot` и `caddy` на VPS по SSH.

Перед первым деплоем создайте следующие секреты репозитория или окружения GitHub:

- `VPS_HOST` — VPS hostname or IP;
- `VPS_USER` — непривилегированный SSH-пользователь с правами Docker;
- `VPS_APP_DIR` — абсолютный путь к каталогу приложения на VPS;
- `VPS_SSH_PRIVATE_KEY` — приватный ключ для GitHub runner;
- `VPS_KNOWN_HOSTS` — независимо полученный вывод `ssh-keyscan -H <host>`;
- `GHCR_USERNAME` и `GHCR_READ_TOKEN` — данные для скачивания приватного образа GHCR; для публичного пакета не нужны.

На VPS один раз вручную создайте `${VPS_APP_DIR}/.env` и храните настоящий
`MAX_BOT_TOKEN` там. Workflow деплоя не копирует секреты на машину, а только
обновляет Compose-файл и тег образа.

После получения публичного имени бота можно добавить переменную репозитория
GitHub `MAX_BOT_URL`. Значение должно быть полным URL, например
`https://max.ru/<username>`; кнопка бота появится на frontend только при наличии
этой переменной.

Для деплоя откройте **Actions → Deploy → Run workflow**, выберите ref и запустите
workflow. Для обычных обновлений подключаться к VPS по SSH не требуется.
