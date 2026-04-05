# 2026_1_KISS go
Проект Colab команды KISS

## Структура проекта

```
cmd/server/     # точка входа приложения
internal/       # приватный код (сервисы, репозитории, хендлеры)
scripts/        # скрипты (валидация коммитов и т.д.)
```

## Запуск
На данный момент полный функционал гарантирован исключительно при запуске через docker

Перед запуском самого сервера необходимо собрать образ ранера
```shell
docker build -t kiss-python-runner -f build/runner/Dockerfile .
```

Запуск осуществляется следующей командой из корневой папки проекта:
```shell
docker compose up
```

При деплое **настоятельно рекомендуется** установить и настроить gVisor (runsc), что спасёт от container escape,
посредством куда большей изоляции выполняемого кода от пользователей.

Приложение — `http://localhost:8080`, PostgreSQL — порт 5432.

## Запуск CI локально

Проверки из GitHub Actions можно запускать на своей машине двумя способами.

**1. Те же команды, что и в CI (рекомендуется):**
```bash
make ci
```
Выполняет `golangci-lint run ./...` и `go test -race ./...` — то же, что в workflow.

**2. Полный прогон workflow в Docker (как на GitHub):**
Установите [act](https://github.com/nektos/act) и выполните:
```bash
act pull_request
# или для push
act push
```
`act` поднимает контейнеры, ставит Go и запускает все jobs из `.github/workflows/ci.yml`. Нужен установленный Docker.

* В первый раз может быть долгое выполнение

## Разработка

- **Ветки:** разработка ведётся в `develop`, релизы — в `master`. См. [CONTRIBUTING.md](CONTRIBUTING.md).
- **Коммиты:** используем [Conventional Commits](https://habr.com/ru/articles/867012/).
- **Проверки:** перед коммитом установите Lefthook и выполните `lefthook install`.

## Полезные команды

| Команда        | Описание              |
|----------------|------------------------|
| `make build`   | Сборка бинарника      |
| `make run`     | Запуск приложения     |
| `make test`     | Запуск тестов         |
| `make lint`     | Запуск golangci-lint  |
| `make ci`       | Линт + тесты (как в CI) |
| `make docker-up`   | Поднять Docker-сервисы |
| `make docker-down` | Остановить Docker-сервисы |

## Лицензия

MIT — см. [LICENSE](LICENSE).
