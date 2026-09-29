# Публичный HTTP API

Backend предоставляет небольшой публичный API для проверки состояния
развёртывания:

- `GET /api/version` — возвращает commit/revision приложения;
- `/api/swagger/` — Swagger UI;
- `/api/swagger/doc.json` — сгенерированная спецификация Swagger 2.0.

Swagger-описание генерируется из Go-аннотаций командой:

```bash
make swagger
```

Сгенерированные файлы находятся в
`src/backend/internal/infrastructure/httpserver/swagger/`. CI запускает генератор
и проверяет, что результат сохранён в репозитории и не устарел.

Публичный API не содержит методов авторизации или управления данными. Webhook
MAX (`/webhook`) остаётся отдельной защищённой точкой интеграции и не включается
в Swagger-описание.
