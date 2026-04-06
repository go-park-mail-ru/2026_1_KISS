# Runner Container Manager

`internal/runner/container` реализует инфраструктурный адаптер Docker для порта `internal/runner.Manager`.

## Что делает

- создает контейнер на сессию (`runner-<sessionID>`),
- переиспользует уже запущенный контейнер,
- удаляет остановленный контейнер и поднимает новый,
- ждет готовность runner-агента по `GET /health`,
- умеет останавливать контейнер сессии и массово очищать managed-контейнеры.

## Конфигурация

Используются поля `RunnerConfig` из `internal/pkg/config`:

- `RUNNER_IMAGE` (default `kiss-python-runner`)
- `RUNNER_NETWORK` (default `bridge`)
- `RUNNER_NAME_PREFIX` (default `runner-`)
- `RUNNER_AGENT_PORT` (default `8080`)
- `RUNNER_MEMORY_LIMIT_BYTES` (default `536870912`)
- `RUNNER_NANO_CPUS` (default `1000000000`)
- `RUNNER_STARTUP_TIMEOUT` (default `20s`)
- `RUNNER_HEALTHCHECK_INTERVAL` (default `300ms`)

## Тесты

Пакет покрыт unit-тестами с fake Docker API:

- `manager_test.go`
- `readiness_test.go`

