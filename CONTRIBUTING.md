# Руководство по внесению изменений

Спасибо за интерес к проекту! Этот документ описывает процесс разработки и внесения изменений.

## Ветки

- **develop** — основная ветка разработки. Всегда должна содержать рабочий код, готовый к развёртыванию.
- **master** — ветка релизов. Содержит только стабильный код, прошедший тестирование.
- **feature/&lt;name&gt;** — ветки для новой функциональности.
- **fix/&lt;name&gt;** — ветки для исправлений.

## Workflow

1. Создайте ветку от `develop`:
   ```bash
   git checkout develop
   git pull origin develop
   git checkout -b feature/my-feature
   ```

2. Внесите изменения и убедитесь, что всё работает:
   ```bash
   go fmt ./...
   golangci-lint run ./...
   go test ./...
   ```

3. Создайте Pull Request в ветку `develop`.

4. Дождитесь прохождения CI и code review.

5. После одобрения изменения будут смержены в `develop`.

## Conventional Commits

Используем [Conventional Commits](https://habr.com/ru/articles/867012/) для единообразия сообщений коммитов.

**Формат:** `type(scope)?: описание`

**Типы:**
- `feat` — новая функциональность
- `fix` — исправление ошибки
- `docs` — изменения в документации
- `style` — правки стиля (форматирование, пробелы)
- `refactor` — рефакторинг без изменения функциональности
- `perf` — оптимизация производительности
- `test` — добавление или обновление тестов
- `build` — изменения сборки
- `ci` — настройка или изменение CI/CD
- `chore` — прочие задачи
- `revert` — откат предыдущего коммита

**Scope (опционально):** `auth`, `ui`, `api`, `core`, `config`, `deps`, `tests`, `db`, `build`

**Примеры:**
```
feat(auth): add Google OAuth support
fix(ui): fix button alignment issue
docs: update README
ci: add golangci-lint config
```

## Запуск локально

**1. Настроить переменные окружения:**
```bash
cp .env.example .env
# при необходимости отредактировать .env
```

**2. Через Docker (рекомендуется):**
```bash
docker-compose up -d --build
```

Миграции применяются автоматически — сервис `migrator` запускается перед `app` и завершается сам.

**3. Через Go (нужен запущенный PostgreSQL):**
```bash
go run ./cmd/migrator   # применить миграции
go run ./cmd/server     # запустить сервер
```

Или через Make:
```bash
make migrate
make run
```

Приложение будет доступно на `http://localhost:8080`.

## Полезные команды Make

| Команда | Описание |
|---------|----------|
| `make run` | Запустить сервер локально |
| `make test` | Тесты с race detector |
| `make cover` | Тесты + открыть отчёт покрытия в браузере |
| `make lint` | Запустить golangci-lint |
| `make fmt` | Форматирование кода |
| `make vet` | Статический анализ go vet |
| `make ci` | lint + test (как в CI) |
| `make migrate` | Применить миграции |
| `make docker-up` | Поднять Docker-окружение |
| `make docker-down` | Остановить Docker-окружение |
| `make docs` | Пересобрать документацию API |

## Pre-commit

Используется [Lefthook](https://github.com/evilmartians/lefthook) для автоматической проверки перед коммитом.

**Установка:**
```bash
# Через Go
go install github.com/evilmartians/lefthook/v2@v2.1.2

# Или через brew
brew install lefthook

# Или через npm
npm install lefthook --save-dev
```

**Активация хуков:**
```bash
lefthook install
```

После установки при каждом коммите будут выполняться:
- `go fmt` — форматирование кода
- `golangci-lint` — статический анализ
- Валидация сообщения коммита (Conventional Commits)

**Обход проверок** (в исключительных случаях):
```bash
SKIP=golangci-lint git commit -m "WIP: work in progress"
```
