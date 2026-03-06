# Нормализация схемы базы данных — Проект KISS (Jupyter Notebooks)

---

## Таблица USER_ACCOUNT

Таблица `USER_ACCOUNT` содержит информацию о зарегистрированных пользователях системы.
Хранит учётные данные, хешированный пароль (bcrypt автоматически включает соль в хеш, поэтому отдельное поле `salt` не требуется) и ссылку на аватар.

<p> Функциональные зависимости: </p>

- `{id} -> {name, email, password_hash, avatar_url, created_at, updated_at}`
- `{email} -> {id, name, password_hash, avatar_url, created_at, updated_at}`

<p> Нормальные формы: </p>

- 1 НФ: Все атрибуты являются атомарными.
- 2 НФ: Все атрибуты полностью функционально зависят от первичного ключа id.
- 3 НФ: Все атрибуты не зависят от других неключевых атрибутов.
- НФБК: 3 НФ + в таблице отсутствуют составные ключи.

```mermaid
erDiagram
    USER_ACCOUNT {
        INT id PK "Уникальный идентификатор пользователя"
        TEXT name "Имя пользователя"
        CHAR status "Статус"
        TEXT description "Пункт 'о себе'"
        TEXT email "Email пользователя (AK1)"
        TEXT password_hash "Хеш пароля (bcrypt, включает соль)"
        TEXT avatar_url "Путь к файлу аватара"
        TIMESTAMPTZ created_at "Время создания"
        TIMESTAMPTZ updated_at "Время обновления"
    }
```

---

## Таблица IPYNB_FILE

Таблица `IPYNB_FILE` содержит информацию о файлах Jupyter Notebook.
Хранит версию формата, язык программирования и владельца файла. Поле `programming_language` реализовано как ENUM с допустимыми значениями `Python` и `R`.

<p> Функциональные зависимости: </p>

- `{id} -> {owner_id, title, nbformat, nbformat_minor, programming_language, created_at, updated_at}`

<p> Нормальные формы: </p>

- 1 НФ: Все атрибуты являются атомарными.
- 2 НФ: Все атрибуты полностью функционально зависят от первичного ключа id.
- 3 НФ: Все атрибуты не зависят от других неключевых атрибутов.
- НФБК: 3 НФ + в таблице отсутствуют составные ключи.

```mermaid
erDiagram
    IPYNB_FILE {
        INT id PK "Уникальный идентификатор файла"
        INT owner_id FK "Идентификатор пользователя-владельца"
        TEXT title "Название файла"
        INT nbformat "Основная версия формата"
        INT nbformat_minor "Минорная версия формата"
        PROGRAMMING_LANGUAGE programming_language "Язык программирования (Python, R)"
        TIMESTAMPTZ created_at "Время создания"
        TIMESTAMPTZ updated_at "Время обновления"
    }
```

---

## Таблица FILE_PERMISSION

Таблица `FILE_PERMISSION` содержит информацию о правах доступа пользователей к файлам (шеринг).
Составной первичный ключ `{file_id, user_id}` гарантирует, что у одного пользователя может быть только одна запись доступа к конкретному файлу.

<p> Функциональные зависимости: </p>

- `{file_id, user_id} -> {permission_level}`

<p> Нормальные формы: </p>

- 1 НФ: Все атрибуты являются атомарными.
- 2 НФ: Атрибут permission_level полностью функционально зависит от составного ключа {file_id, user_id}.
- 3 НФ: Атрибут permission_level не зависит от других неключевых атрибутов.
- НФБК: 3 НФ + нет транзитивных зависимостей.

```mermaid
erDiagram
    FILE_PERMISSION {
        INT file_id FK "Идентификатор файла (PK1.1)"
        INT user_id FK "Идентификатор пользователя (PK1.2)"
        PERMISSION_LEVEL permission_level "Уровень доступа (edit, comment, readonly)"
    }
```

---

## Таблица IPYNB_CELL

Таблица `IPYNB_CELL` содержит информацию о ячейках (блоках) внутри файла Jupyter Notebook.
Поле `order_index` определяет порядок отображения ячеек внутри файла. Поле `cell_type` реализовано как ENUM с допустимыми значениями `code`, `markdown`, `raw`.

<p> Функциональные зависимости: </p>

- `{id} -> {file_id, order_index, cell_type, source, execution_count, created_at, updated_at}`
- `{file_id, order_index} -> {id, cell_type, source, execution_count, created_at, updated_at}`

<p> Нормальные формы: </p>

- 1 НФ: Все атрибуты являются атомарными.
- 2 НФ: Все атрибуты полностью функционально зависят от первичного ключа id.
- 3 НФ: Все атрибуты не зависят от других неключевых атрибутов.
- НФБК: 3 НФ + в таблице отсутствуют составные ключи.

```mermaid
erDiagram
    IPYNB_CELL {
        INT id PK "Уникальный идентификатор ячейки"
        INT file_id FK "Идентификатор файла"
        INT order_index "Порядковый номер ячейки в файле (AK1.2)"
        CELL_TYPE cell_type "Тип ячейки (code, markdown, raw)"
        TEXT source "Исходный текст ячейки"
        INT execution_count "Порядковый номер запуска"
        TIMESTAMPTZ created_at "Время создания"
        TIMESTAMPTZ updated_at "Время обновления"
    }
```

---

## Таблица CELL_OUTPUT

Таблица `CELL_OUTPUT` содержит информацию о результатах исполнения ячеек с кодом.
Вместо хранения в JSONB (что запрещено на этапе нормализации), каждый выход хранится отдельной строкой. Поле `output_type` определяет тип результата, `order_index` — порядок вывода.

<p> Функциональные зависимости: </p>

- `{id} -> {cell_id, order_index, output_type, text_content, created_at}`
- `{cell_id, order_index} -> {id, output_type, text_content, created_at}`

<p> Нормальные формы: </p>

- 1 НФ: Все атрибуты являются атомарными.
- 2 НФ: Все атрибуты полностью функционально зависят от первичного ключа id.
- 3 НФ: Все атрибуты не зависят от других неключевых атрибутов.
- НФБК: 3 НФ + в таблице отсутствуют составные ключи.

```mermaid
erDiagram
    CELL_OUTPUT {
        INT id PK "Уникальный идентификатор вывода"
        INT cell_id FK "Идентификатор ячейки"
        INT order_index "Порядковый номер вывода (AK1.2)"
        TEXT output_type "Тип вывода (stream, execute_result, error, display_data)"
        TEXT text_content "Текстовое содержимое вывода"
        TIMESTAMPTZ created_at "Время создания"
    }
```

---

## Таблица COMMENT

Таблица `COMMENT` содержит информацию о комментариях пользователей к ячейкам ноутбука.

<p> Функциональные зависимости: </p>

- `{id} -> {user_id, cell_id, text, created_at}`

<p> Нормальные формы: </p>

- 1 НФ: Все атрибуты являются атомарными.
- 2 НФ: Все атрибуты полностью функционально зависят от первичного ключа id.
- 3 НФ: Все атрибуты не зависят от других неключевых атрибутов.
- НФБК: 3 НФ + в таблице отсутствуют составные ключи.

```mermaid
erDiagram
    COMMENT {
        INT id PK "Уникальный идентификатор комментария"
        INT user_id FK "Идентификатор пользователя"
        INT cell_id FK "Идентификатор ячейки"
        TEXT text "Текст комментария"
        TIMESTAMPTZ created_at "Время создания"
    }
```

---

## Таблица SUBSCRIPTION_PLAN

Таблица `SUBSCRIPTION_PLAN` содержит информацию о доступных тарифных планах подписки.
Необходима для монетизации (покупка подписок с квотой на исполнение).

<p> Функциональные зависимости: </p>

- `{id} -> {name, price, execution_quota, duration_day, created_at}`
- `{name} -> {id, price, execution_quota, duration_day, created_at}`

<p> Нормальные формы: </p>

- 1 НФ: Все атрибуты являются атомарными.
- 2 НФ: Все атрибуты полностью функционально зависят от первичного ключа id.
- 3 НФ: Все атрибуты не зависят от других неключевых атрибутов.
- НФБК: 3 НФ + в таблице отсутствуют составные ключи.

```mermaid
erDiagram
    SUBSCRIPTION_PLAN {
        INT id PK "Уникальный идентификатор плана"
        TEXT name "Название плана (AK1)"
        INT price "Цена в копейках"
        INT execution_quota "Квота на исполнение"
        INT duration_day "Длительность подписки в днях"
        TIMESTAMPTZ created_at "Время создания"
    }
```

---

## Таблица USER_SUBSCRIPTION

Таблица `USER_SUBSCRIPTION` содержит информацию о подписках, приобретённых пользователями.
Хранит оставшуюся квоту исполнения и период действия подписки.

<p> Функциональные зависимости: </p>

- `{id} -> {user_id, plan_id, execution_remaining, started_at, expires_at, created_at}`

<p> Нормальные формы: </p>

- 1 НФ: Все атрибуты являются атомарными.
- 2 НФ: Все атрибуты полностью функционально зависят от первичного ключа id.
- 3 НФ: Все атрибуты не зависят от других неключевых атрибутов.
- НФБК: 3 НФ + в таблице отсутствуют составные ключи.

```mermaid
erDiagram
    USER_SUBSCRIPTION {
        INT id PK "Уникальный идентификатор подписки"
        INT user_id FK "Идентификатор пользователя"
        INT plan_id FK "Идентификатор тарифного плана"
        INT execution_remaining "Остатки по тарифу"
        TIMESTAMPTZ started_at "Дата начала подписки"
        TIMESTAMPTZ expires_at "Дата окончания подписки"
        TIMESTAMPTZ created_at "Время создания"
    }
```
