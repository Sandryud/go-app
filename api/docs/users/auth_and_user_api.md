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

- **Описание**: регистрация нового пользователя.
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
  "tokens": {
    "access_token": "...",
    "refresh_token": "..."
  }
}
```

- **Ошибки**:
  - `400 invalid_request` — невалидное тело.
  - `409 email_already_exists` — email занят.
  - `409 username_already_exists` — username занят.

Пример:

```bash
curl -i -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user1@example.com","password":"Password123!","username":"user1"}'
```

---

### POST `/api/v1/auth/login`

- **Описание**: вход по email/паролю.
- **Тело**:

```json
{
  "email": "user1@example.com",
  "password": "Password123!"
}
```

- **Успех**: `200 OK` + `LoginResponse` (как при регистрации).
- **Ошибки**:
  - `400 invalid_request`
  - `401 invalid_credentials` — неверный email или пароль.

Пример:

```bash
curl -i -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user1@example.com","password":"Password123!"}'
```

---

### POST `/api/v1/auth/refresh`

- **Описание**: обновление пары access/refresh по refresh‑токену.
- **Тело**:

```json
{
  "refresh_token": "..."
}
```

- **Успех**: `200 OK` + новая пара токенов.
- **Ошибки**:
  - `400 invalid_request`
  - `401 invalid_refresh_token`

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

- **Описание**: частичное обновление профиля.
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
  - `409 email_already_exists` / `username_already_exists`

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



