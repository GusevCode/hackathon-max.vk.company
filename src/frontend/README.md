# Frontend

Статусная страница проекта на React + TypeScript + Vite с Feature-Sliced Design.
В production она
собирается в Docker-образ на базе Caddy и публикуется по
`https://max.conspiracy-team.ru/`.

## Локальный запуск

```bash
npm ci
npm run dev
```

Во время разработки `/healthz` и `/api/*` проксируются с Vite на Go-сервис
`localhost:8080`. Проверки выполняются командами `npm run typecheck`,
`npm test` и `npm run build`.

Исходники организованы по слоям `app`, `pages`, `widgets`, `features`,
`entities` и `shared`. Папка `dist` создаётся Vite автоматически после build и
не хранится в Git.

## Ссылка на бота

Если в GitHub Actions задать repository variable `MAX_BOT_URL` со ссылкой вида
`https://max.ru/<username>`, она будет встроена в сборку и появится кнопка
«Открыть бота в MAX». Без этой переменной интерфейс не показывает непроверенную
ссылку.
