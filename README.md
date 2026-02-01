# Chirpy

A social media backend clone written in Go!

## Features

- **User Authentication**: Secure signup and login using JWTs and refresh tokens.
- **Chirps**: Create, read, and delete short text posts ("chirps").
- **Sorting**: Fetch chirps in ascending or descending order by creation time.
- **Author Filtering**: Retrieve all chirps from a specific user.
- **Chirpy Red**: A premium membership tier managed via webhooks.
- **Admin Metrics**: Track server hits and manage database resets (dev mode only).

## Tech Stack

- **Language**: Go (Golang)
- **Database**: PostgreSQL
- **SQL Generator**: sqlc
- **Migrations**: goose
- **Router**: Standard library `net/http`

## Installation

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/ifeanyiBatman/chirpy.git
    cd chirpy
    ```

2.  **Set up the environment:**
    Create a `.env` file in the root directory with the following variables:
    ```env
    DB_URL="postgres://postgres:postgres@localhost:5432/chirpy?sslmode=disable"
    PLATFORM="dev"
    JWT_SECRET="your-jwt-secret-key"
    POLKA_KEY="your-polka-api-key"
    ```

3.  **Run migrations:**
    ```bash
    goose postgres "postgres://postgres:postgres@localhost:5432/chirpy" up
    ```

4.  **Run the server:**
    ```bash
    go run .
    ```
    The server works on `http://localhost:8080`.

## API Endpoints

### Public
- `GET /api/healthz`: Check server health.
- `GET /api/chirps`: Get all chirps. Supports query parameters `?sort=desc` and `?author_id=<uuid>`.
- `GET /api/chirps/{chirpID}`: Get a single chirp by ID.
- `POST /api/users`: Create a new user account.
- `POST /api/login`: Login to get access and refresh tokens.

### Authenticated
- `POST /api/chirps`: Create a new chirp.
- `DELETE /api/chirps/{chirpID}`: Delete your own chirp.
- `PUT /api/users`: Update your email and password.
- `POST /api/refresh`: Refresh your access token.
- `POST /api/revoke`: Revoke your refresh token.

### Webhooks
- `POST /api/polka/webhooks`: Upgrade users to Chirpy Red status.


