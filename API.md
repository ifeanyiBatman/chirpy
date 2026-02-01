# API Documentation

## API Endpoints

### Public

#### `GET /api/healthz`
Check server health.
- **Response:** `200 OK` (Text: "OK")

#### `GET /api/chirps`
Get all chirps.
- **Query Parameters:**
  - `sort`: `asc` or `desc` (optional, defaults to `asc`)
  - `author_id`: UUID of a specific user (optional)
- **Response:** `200 OK` (JSON list of chirps)

#### `GET /api/chirps/{chirpID}`
Get a single chirp by ID.
- **Response:** `200 OK` (JSON chirp object) or `404 Not Found`

#### `POST /api/users`
Create a new user account.
- **Body:**
  ```json
  {
    "email": "user@example.com",
    "password": "securepassword"
  }
  ```
- **Response:** `201 Created` (JSON user object with `id`, `email`, `is_chirpy_red`)

#### `POST /api/login`
Login to get access and refresh tokens.
- **Body:**
  ```json
  {
    "email": "user@example.com",
    "password": "securepassword"
  }
  ```
- **Response:** `200 OK` (JSON user object including `token` and `refresh_token`)

### Authenticated

**Note:** Authenticated endpoints require the header `Authorization: Bearer <access_token>` unless specified otherwise.

#### `POST /api/chirps`
Create a new chirp.
- **Body:**
  ```json
  {
    "body": "This is my chirp!",
    "user_id": "uuid-here" // Must match the authenticated user
  }
  ```
- **Response:** `201 Created` (JSON chirp object)

#### `DELETE /api/chirps/{chirpID}`
Delete your own chirp.
- **Response:** `204 No Content`

#### `PUT /api/users`
Update your email and password.
- **Body:**
  ```json
  {
    "email": "new@example.com",
    "password": "newpassword"
  }
  ```
- **Response:** `200 OK` (Updated JSON user object)

#### `POST /api/refresh`
Refresh your access token.
- **Header:** `Authorization: Bearer <refresh_token>`
- **Response:** `200 OK` (JSON object with new `token`)

#### `POST /api/revoke`
Revoke your refresh token.
- **Header:** `Authorization: Bearer <refresh_token>`
- **Response:** `204 No Content`

### Webhooks

#### `POST /api/polka/webhooks`
Upgrade users to Chirpy Red status.
- **Header:** `X-API-Key: <POLKA_KEY>`
- **Body:**
  ```json
  {
    "event": "user.upgraded",
    "data": {
      "user_id": "uuid-of-user-to-upgrade"
    }
  }
  ```
- **Response:** `204 No Content`
