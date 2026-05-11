# OAuth-вход через Google, VKID и Yandex ID

Authorization Code + PKCE (S256) полностью на стороне backend. Фронт делает
top-level redirect на `/api/v1/auth/oauth/<provider>/start`, дальше всё
state/PKCE/обмен кода живёт в auth-сервисе.

## Поддерживаемые провайдеры

| Provider | Имя в URL | Auth URL                                | Token URL                          | Userinfo URL                                       | Scope                  |
|----------|-----------|-----------------------------------------|------------------------------------|----------------------------------------------------|------------------------|
| Google   | `google`  | `https://accounts.google.com/o/oauth2/v2/auth` | `https://oauth2.googleapis.com/token` | `https://openidconnect.googleapis.com/v1/userinfo` | `openid email profile` |
| Yandex   | `yandex`  | `https://oauth.yandex.ru/authorize`     | `https://oauth.yandex.ru/token`    | `https://login.yandex.ru/info?format=json`         | `login:email login:info` |
| VK ID    | `vkid`    | `https://id.vk.com/authorize`           | `https://id.vk.com/oauth2/auth`    | `https://id.vk.com/oauth2/user_info`               | `email`                |

PKCE S256 включён для всех провайдеров. VK ID требует PKCE обязательно.

## HTTP-эндпоинты

| Метод | Путь                                            | Назначение |
|-------|-------------------------------------------------|------------|
| GET   | `/api/v1/auth/oauth/{provider}/start`           | Запускает OAuth-флоу: создаёт state+PKCE, ставит cookie `oauth_state`, редиректит на провайдера. |
| GET   | `/api/v1/auth/oauth/{provider}/callback`        | Принимает редирект от провайдера, валидирует state, обменивает code → access_token → userinfo, ставит session-cookie и редиректит на `<FRONTEND_URL>/files`. |

При ошибке `callback` редиректит на `<FRONTEND_URL>/login?oauth_error=<code>`.
Коды: `invalid_state`, `email_taken`, `denied`, `invalid_request`,
`unknown_provider`, `internal` (UI переводит через `serverErrors.ts`).

## Логика linking

1. Если в `oauth_accounts` уже есть `(provider, provider_id)` — логиним связанного пользователя.
2. Иначе, если провайдер вернул верифицированный email и в БД есть пользователь с этим email + `is_verified=true` — добавляем строку в `oauth_accounts` (link) и логиним этого пользователя.
3. Если такой email есть, но `is_verified=false` — отказ `ErrConflict` (UI: `oauth_error=email_taken`).
4. Иначе создаём нового пользователя: `username` из профиля провайдера + цифровой суффикс при коллизии, `is_verified=true` (для VK ID — только если есть email), `password_hash=""`.

## Конфиг (env-vars)

| Переменная                  | Дефолт                | Назначение |
|-----------------------------|-----------------------|------------|
| `OAUTH_STATE_TTL`           | `10m`                 | TTL state+PKCE-verifier в Redis. |
| `OAUTH_FRONTEND_URL`        | значение `APP_URL`    | Куда редиректить после успешного/неуспешного callback. |
| `GOOGLE_CLIENT_ID`          | —                     | Включает Google, если непустой вместе с `GOOGLE_CLIENT_SECRET`. |
| `GOOGLE_CLIENT_SECRET`      | —                     | |
| `GOOGLE_REDIRECT_URL`       | —                     | URL, зарегистрированный в Google Cloud Console. |
| `YANDEX_CLIENT_ID`          | —                     | Включает Yandex. |
| `YANDEX_CLIENT_SECRET`      | —                     | |
| `YANDEX_REDIRECT_URL`       | —                     | URL, зарегистрированный в Яндекс OAuth. |
| `VKID_CLIENT_ID`            | —                     | Включает VK ID. |
| `VKID_CLIENT_SECRET`        | —                     | |
| `VKID_REDIRECT_URL`         | —                     | URL, зарегистрированный в VK ID. |

Провайдер «включён» когда заполнены `CLIENT_ID` и `CLIENT_SECRET`. Иначе
gateway вернёт `oauth_error=unknown_provider`.

## Redirect URLs для регистрации в консолях провайдеров

В каждой консоли провайдера зарегистрируйте redirect URL вида
`https://<host>/api/v1/auth/oauth/<provider>/callback`.

| Окружение | URL примера |
|-----------|-------------|
| Локально (`make docker-up` + `npm run dev`) | `http://localhost:8080/api/v1/auth/oauth/google/callback`, `.../yandex/callback`, `.../vkid/callback` |
| Production | `https://colkiss.ru/api/v1/auth/oauth/google/callback`, `.../yandex/callback`, `.../vkid/callback` |

## Безопасность

- `state` и PKCE `code_verifier` — `crypto/rand`, 32 байта, `base64url` без padding.
- PKCE метод — **только** `S256`.
- Redis-state одноразовый (`GETDEL`), TTL = `OAUTH_STATE_TTL`.
- `redirect_uri` подставляется ТОЛЬКО из конфига; не принимается из request/header — нет open-redirect.
- Финальный редирект на фронт — захардкоженный `<FRONTEND_URL>/files`.
- Cookie `oauth_state`: `HttpOnly`, `Secure`, `SameSite=Lax`, `Path=/api/v1/auth/oauth`. Используется как double-submit-проверка против state из query.
- Whitelist provider'ов проверяется в gateway до gRPC.

## Миграции

- `migrations/010_create_oauth_accounts.sql` — таблица `oauth_accounts(user_id, provider, provider_id)` с `UNIQUE(provider, provider_id)`.
- `migrations/020_oauth_password_nullable.sql` — `password_hash` сделан nullable + индекс `idx_users_email_lower` по `LOWER(email)` для быстрого case-insensitive поиска при link-by-email.
