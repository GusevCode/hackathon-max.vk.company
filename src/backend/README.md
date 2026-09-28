# Backend architecture

Backend написан на Go и использует чистую архитектуру с тремя слоями:

- `internal/domain` — независимые бизнес-сущности;
- `internal/usecase` — сценарии приложения и интерфейсы-порты;
- `internal/infrastructure` — MAX SDK и HTTP-адаптеры.

`cmd/control-bot` является composition root: читает конфигурацию, создаёт MAX
клиент, health server, Tarantool/SeaweedFS и control use case. Use case не импортирует
MAX SDK и тестируется через fake gateway.

Control use case разделён по ответственностям: `menus.go` и `callbacks.go`
обрабатывают навигацию, `wizard.go` — пошаговые диалоги, `tasks.go` — жизненный
цикл задания, `users.go` — пользователи и назначения, `reference.go` — объекты и
виды работ, а `access/` содержит единую ACL-политику. Ни один пользовательский
сценарий не требует ручного ввода команды.

Транспорт MAX — Webhook. Tarantool, SeaweedFS и NATS подключаются как отдельные
сервисы Docker Compose; начальная схема Tarantool находится в
`infra/tarantool/init.lua`. Backend является модульным монолитом: control-модуль
публикует события заданий, а notifications-модуль доставляет их через NATS.
