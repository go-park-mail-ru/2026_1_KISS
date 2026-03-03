# 2026_1_KISS go
Проект Colab команды KISS

## Структура проекта

```
cmd/server/     # точка входа приложения
internal/        # приватный код (сервисы, репозитории, хендлеры)
scripts/        # скрипты (валидация коммитов и т.д.)
```

## Разработка

- **Ветки:** разработка ведётся в `develop`, релизы — в `master`. См. [CONTRIBUTING.md](CONTRIBUTING.md).
- **Коммиты:** используем [Conventional Commits](https://habr.com/ru/articles/867012/).
- **Проверки:** перед коммитом установите Lefthook и выполните `lefthook install`.

## Полезные команды

| Команда        | Описание              |
|----------------|------------------------|
| `make build`   | Сборка бинарника      |
| `make run`     | Запуск приложения     |
| `make test`    | Запуск тестов         |
| `make lint`    | Запуск golangci-lint  |
| `make docker-up`   | Поднять Docker-сервисы |
| `make docker-down` | Остановить Docker-сервисы |

## Лицензия

MIT — см. [LICENSE](LICENSE).
