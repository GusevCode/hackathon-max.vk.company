# Архитектура проекта

## Frontend: Feature-Sliced Design

Frontend живёт в `src/frontend/src` и разделён по слоям FSD:

```text
src/frontend/
  src/
    app/       точка входа приложения, глобальные стили и композиция
    pages/     страницы и их сценарии (сейчас status)
    widgets/   крупные блоки страницы (header, status-board)
    features/  действия пользователя (check-api)
    entities/  предметные сущности и их UI (service-status)
    shared/    API-клиенты, типы и переиспользуемые UI-примитивы
  public/      статические файлы
  dist/        результат Vite build, не коммитится
```

Зависимости направлены сверху вниз: `app` может собирать `pages`, `pages` —
`widgets` и `features`, а повторно используемый код располагается в `shared`.
Слои не должны импортировать код из соседнего слоя на том же уровне.

`dist` создаётся командой `npm run build` и используется только на этапе
сборки Docker-образа frontend.

## Backend: модульный монолит и чистая архитектура

```text
src/backend/
  cmd/control-bot/                       composition root
  internal/
    domain/                             слой предметной области
    usecase/control/                    меню, wizard-сценарии и workflow
    usecase/access/                     централизованная ACL-политика
    usecase/notifications/              доставка уведомлений из событий
    infrastructure/maxbot/              адаптер MAX SDK
    infrastructure/messaging/            адаптер брокера NATS
    infrastructure/tarantool/           persistence Tarantool
    infrastructure/objectstorage/       SeaweedFS/S3-compatible storage
    infrastructure/httpserver/          HTTP health delivery
```

Правила зависимостей:

1. `domain` не знает о фреймворках, HTTP и MAX SDK.
2. `usecase` содержит бизнес-сценарии контроля работ и порт `BotGateway`; он зависит
   только от `domain` и стандартной библиотеки.
3. `infrastructure` реализует порты application-слоя: MAX SDK, NATS, Tarantool,
   SeaweedFS и HTTP endpoint.
4. `cmd/control-bot` собирает зависимости и запускает приложение, но не содержит
   бизнес-логики.

Внутри `usecase/control` один файл отвечает за один связный сценарий: меню,
callbacks, приглашения, пользователи, задания, справочники и пошаговые wizard’ы
разнесены по отдельным файлам. Проверки полномочий не копируются в handlers, а
используют пакет `usecase/access`.

Такой порядок позволяет тестировать сценарии через fake gateway без сети и
без токена MAX. Webhook, Tarantool, SeaweedFS и NATS подключаются в composition
root и не переносят SDK-зависимости в use case.

### Модули и события

Приложение остаётся одним бинарником и одним контейнером, но разделено на
модули с портами между ними:

```text
control -> NotificationPublisher -> NATS -> notifications -> BotGateway(MAX)
   |
   +-> Repository/PhotoStore/Storage
```

Модуль `control` публикует события `task.assigned`, `task.submitted` и
`task.reviewed`, а модуль `notifications` подписывается на них и отправляет
личные сообщения сотрудникам и руководителям. Если NATS временно недоступен,
контур управления использует безопасный прямой fallback, чтобы бизнес-действие
не терялось. После стабилизации домена уведомления можно вынести в отдельный
worker без изменения use case-кода.

Для локального и VPS-развёртывания используется NATS с включённым JetStream
(`nats:2.11-alpine`, volume `nats_data`). На текущем MVP consumer использует
core-subscription; JetStream оставлен включённым как следующий шаг для durable
доставки, повторов и dead-letter очереди.
