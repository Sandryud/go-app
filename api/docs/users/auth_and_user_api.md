# Auth & User API

## Общие сведения

- **Базовый URL**: `http://localhost:8080`
- **Версия API**: `/api/v1`
- **Аутентификация**: JWT access‑токен в заголовке  
  `Authorization: Bearer <access_token>`

### Формат ошибок

Все ошибки возвращаются в едином формате:

```json
{
  "error": {
    "code": "invalid_request",
    "message": "Некорректное тело запроса",
    "details": "подробности (опционально)"
  }
}
```

---

## Auth

### POST `/api/v1/auth/register`

- **Описание**: регистрация нового пользователя. После регистрации на указанный email отправляется код подтверждения. Для получения токенов доступа необходимо подтвердить email через эндпоинт `/api/v1/auth/verify-email`.
- **Тело запроса**:

```json
{
  "email": "user1@example.com",
  "password": "Password123!",
  "username": "user1"
}
```

- **Успех**: `201 Created`

```json
{
  "user_id": "3691663d-0fb2-4cc4-a0c3-8ad710d00835",
  "email": "user1@example.com",
  "username": "user1",
  "message": "Verification code has been sent to your email"
}
```

- **Ошибки**:
  - `400 invalid_request` — невалидное тело.
  - `409 email_already_exists` — email занят (аккаунт уже подтверждён).
  - `409 email_unverified` — аккаунт с таким email существует, но не подтверждён. Запросите новый код подтверждения через `/api/v1/auth/resend-verification`.
  - `409 username_already_exists` — username занят.

Пример:

```bash
curl -i -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user1@example.com","password":"Password123!","username":"user1"}'
```

---

### POST `/api/v1/auth/login`

- **Описание**: вход по email/паролю. Требуется, чтобы email был подтверждён через `/api/v1/auth/verify-email`.
- **Тело**:

```json
{
  "email": "user1@example.com",
  "password": "Password123!"
}
```

- **Успех**: `200 OK`

```json
{
  "user_id": "3691663d-0fb2-4cc4-a0c3-8ad710d00835",
  "email": "user1@example.com",
  "username": "user1",
  "tokens": {
    "access_token": "...",
    "refresh_token": "..."
  }
}
```

- **Ошибки**:
  - `400 invalid_request`
  - `401 invalid_credentials` — неверный email или пароль.
  - `403 email_not_verified` — email не подтверждён. Используйте `/api/v1/auth/verify-email` для подтверждения.

Пример:

```bash
curl -i -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user1@example.com","password":"Password123!"}'
```

---

### POST `/api/v1/auth/verify-email`

- **Описание**: подтверждение email одноразовым кодом, отправленным при регистрации. После успешного подтверждения возвращает пару access/refresh токенов.
- **Тело**:

```json
{
  "email": "user1@example.com",
  "code": "123456"
}
```

- **Успех**: `200 OK`

```json
{
  "user_id": "3691663d-0fb2-4cc4-a0c3-8ad710d00835",
  "email": "user1@example.com",
  "username": "user1",
  "tokens": {
    "access_token": "...",
    "refresh_token": "..."
  }
}
```

- **Ошибки**:
  - `400 invalid_request` — невалидное тело запроса.
  - `400 verification_code_not_found` — код не найден или истёк срок действия. Запросите новый код через `/api/v1/auth/resend-verification`.
  - `400 verification_code_invalid` — неверный код подтверждения.
  - `400 verification_attempts_exceeded` — превышен лимит попыток ввода кода. Запросите новый код.
  - `409 email_already_verified` — email уже подтверждён.

Пример:

```bash
curl -i -X POST http://localhost:8080/api/v1/auth/verify-email \
  -H "Content-Type: application/json" \
  -d '{"email":"user1@example.com","code":"123456"}'
```

---

### POST `/api/v1/auth/resend-verification`

- **Описание**: повторная отправка кода подтверждения email. Используется, если код не пришёл, истёк или были исчерпаны попытки ввода.
- **Тело**:

```json
{
  "email": "user1@example.com"
}
```

- **Успех**: `200 OK`

```json
{
  "message": "If an account with this email exists, a verification code has been sent"
}
```

Если email уже подтверждён, возвращается:

```json
{
  "message": "Email is already verified"
}
```

- **Ошибки**:
  - `400 invalid_request` — невалидное тело запроса.

Пример:

```bash
curl -i -X POST http://localhost:8080/api/v1/auth/resend-verification \
  -H "Content-Type: application/json" \
  -d '{"email":"user1@example.com"}'
```

---

### POST `/api/v1/auth/refresh`

- **Описание**: обновление пары access/refresh по refresh‑токену. Требуется, чтобы email пользователя был подтверждён.
- **Тело**:

```json
{
  "refresh_token": "..."
}
```

- **Успех**: `200 OK`

```json
{
  "user_id": "3691663d-0fb2-4cc4-a0c3-8ad710d00835",
  "email": "user1@example.com",
  "username": "user1",
  "tokens": {
    "access_token": "...",
    "refresh_token": "..."
  }
}
```

- **Ошибки**:
  - `400 invalid_request`
  - `401 invalid_refresh_token` — неверный или истёкший refresh‑токен.
  - `403 email_not_verified` — email не подтверждён. Используйте `/api/v1/auth/verify-email` для подтверждения.

Пример:

```bash
curl -i -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token":"..."}'
```

---

## User (требуется JWT access‑токен)

### GET `/api/v1/users/me`

- **Описание**: получить профиль текущего пользователя.
- **Заголовок**: `Authorization: Bearer <access_token>`
- **Успех**: `200 OK`

```json
{
  "id": "3691663d-0fb2-4cc4-a0c3-8ad710d00835",
  "email": "user1@example.com",
  "username": "user1_new",
  "first_name": "Иван",
  "last_name": "Иванов",
  "gender": "male",
  "role": "user",
  "training_level": "intermediate",
  "created_at": "...",
  "updated_at": "..."
}
```

- **Ошибки**:
  - `401 unauthorized` / `missing_authorization_header` / `invalid_token`
  - `404 user_not_found` — пользователь soft‑deleted.

Пример:

```bash
curl -i http://localhost:8080/api/v1/users/me \
  -H "Authorization: Bearer $ACCESS"
```

---

### PUT `/api/v1/users/me`

- **Описание**: частичное обновление профиля. Email нельзя изменить через этот эндпоинт. Для изменения email используйте `/api/v1/users/me/change-email` и `/api/v1/users/me/verify-email-change`.
- **Тело** (все поля опциональны):

```json
{
  "username": "user1_new",
  "first_name": "Иван",
  "last_name": "Иванов",
  "birth_date": "1990-01-01",
  "gender": "male",
  "avatar_url": "https://example.com/avatar.png",
  "training_level": "intermediate"
}
```

- **Успех**: `200 OK` + обновлённый профиль.
- **Ошибки**:
  - `400 invalid_request` — невалидный JSON/формат.
  - `401 unauthorized`
  - `404 user_not_found`
  - `409 username_already_exists` — указанный username уже используется.

Пример:

```bash
curl -i -X PUT http://localhost:8080/api/v1/users/me \
  -H "Authorization: Bearer $ACCESS" \
  -H "Content-Type: application/json" \
  -d '{"username":"user1_new","training_level":"intermediate"}'
```

---

### DELETE `/api/v1/users/me`

- **Описание**: soft‑delete текущего пользователя (заполняет `deleted_at`).
- **Успех**: `204 No Content`
- **Ошибки**:
  - `401 unauthorized`
  - `404 user_not_found`

Пример:

```bash
curl -i -X DELETE http://localhost:8080/api/v1/users/me \
  -H "Authorization: Bearer $ACCESS"
```

---

### POST `/api/v1/users/me/change-email`

- **Описание**: запрос на изменение email пользователя. Отправляет код подтверждения на новый email. Для завершения изменения email необходимо подтвердить код через `/api/v1/users/me/verify-email-change`.
- **Заголовок**: `Authorization: Bearer <access_token>`
- **Тело запроса**:

```json
{
  "new_email": "newemail@example.com"
}
```

- **Успех**: `200 OK`

```json
{
  "message": "Verification code has been sent to your new email"
}
```

- **Ошибки**:
  - `400 invalid_request` — невалидное тело запроса.
  - `400 email_same_as_current` — новый email совпадает с текущим.
  - `401 unauthorized` — требуется аутентификация.
  - `404 user_not_found` — пользователь не найден.
  - `409 email_already_exists` — указанный email уже используется другим пользователем.

Пример:

```bash
curl -i -X POST http://localhost:8080/api/v1/users/me/change-email \
  -H "Authorization: Bearer $ACCESS" \
  -H "Content-Type: application/json" \
  -d '{"new_email":"newemail@example.com"}'
```

---

### POST `/api/v1/users/me/verify-email-change`

- **Описание**: подтверждение изменения email одноразовым кодом, отправленным на новый email. После успешного подтверждения email пользователя обновляется, и `IsEmailVerified` устанавливается в `true`.
- **Заголовок**: `Authorization: Bearer <access_token>`
- **Тело запроса**:

```json
{
  "code": "123456"
}
```

- **Успех**: `200 OK` + обновлённый профиль пользователя.

```json
{
  "id": "3691663d-0fb2-4cc4-a0c3-8ad710d00835",
  "email": "newemail@example.com",
  "username": "user1",
  "first_name": "Иван",
  "last_name": "Иванов",
  "gender": "male",
  "role": "user",
  "training_level": "intermediate",
  "created_at": "...",
  "updated_at": "..."
}
```

- **Ошибки**:
  - `400 invalid_request` — невалидное тело запроса.
  - `400 verification_code_not_found` — код не найден или истёк срок действия. Запросите новый код через `/api/v1/users/me/change-email`.
  - `400 verification_code_invalid` — неверный код подтверждения.
  - `400 verification_attempts_exceeded` — превышен лимит попыток ввода кода. Запросите новый код.
  - `401 unauthorized` — требуется аутентификация.
  - `404 user_not_found` — пользователь не найден.
  - `409 email_already_exists` — указанный email уже используется другим пользователем.

Пример:

```bash
curl -i -X POST http://localhost:8080/api/v1/users/me/verify-email-change \
  -H "Authorization: Bearer $ACCESS" \
  -H "Content-Type: application/json" \
  -d '{"code":"123456"}'
```

---

## Admin (роль admin)

### GET `/api/v1/admin/users`

- **Описание**: получить список всех активных пользователей.
- **Доступ**: только для пользователей с ролью `admin`.
- **Заголовок**: `Authorization: Bearer <access_token>` (токен администратора)
- **Успех**: `200 OK`

```json
[
  {
    "id": "3691663d-0fb2-4cc4-a0c3-8ad710d00835",
    "email": "user1@example.com",
    "username": "user1",
    "role": "user",
    "training_level": "beginner",
    "created_at": "...",
    "updated_at": "..."
  },
  {
    "id": "....",
    "email": "user2@example.com",
    "username": "user2",
    "role": "coach",
    "training_level": "intermediate",
    "created_at": "...",
    "updated_at": "..."
  }
]
```

- **Ошибки**:
  - `401 unauthorized` / `missing_authorization_header` / `invalid_token`
  - `403 forbidden` — роль пользователя не входит в разрешённые (не admin).



