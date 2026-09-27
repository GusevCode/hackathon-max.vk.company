# Backend architecture

Backend написан на Go и использует чистую архитектуру с тремя слоями:

- `internal/domain` — независимые бизнес-сущности;
- `internal/usecase` — сценарии приложения и интерфейсы-порты;
- `internal/infrastructure` — MAX SDK и HTTP-адаптеры.

`cmd/echo-bot` является composition root: читает конфигурацию, создаёт MAX
клиент, health server, Tarantool/MinIO и control use case. Use case не импортирует
MAX SDK и тестируется через fake gateway.

Транспорт MAX — Webhook. Tarantool и MinIO подключаются как отдельные сервисы
Docker Compose; начальная схема Tarantool находится в `infra/tarantool/init.lua`.
