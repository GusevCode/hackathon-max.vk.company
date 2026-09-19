# Backend architecture

Backend написан на Go и использует чистую архитектуру с тремя слоями:

- `internal/domain` — независимые бизнес-сущности;
- `internal/usecase` — сценарии приложения и интерфейсы-порты;
- `internal/infrastructure` — MAX SDK и HTTP-адаптеры.

`cmd/echo-bot` является composition root: читает конфигурацию, создаёт MAX
клиент, health server и echo use case. Use case не импортирует MAX SDK и
тестируется через fake gateway.
