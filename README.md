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

For detailed API documentation, including request bodies and headers, please see [API.md](API.md).

### Quick List
- `GET /api/healthz`
- `GET /api/chirps`
- `GET /api/chirps/{chirpID}`
- `POST /api/users`
- `POST /api/login`
- `POST /api/chirps`
- `DELETE /api/chirps/{chirpID}`
- `PUT /api/users`
- `POST /api/refresh`
- `POST /api/revoke`
- `POST /api/polka/webhooks`
