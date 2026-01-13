# Versiy — URL Shortener API (Go + PostgreSQL + Redis)

Versiy is a simple URL shortener service built with Go and PostgreSQL.

> Status: **Under active development** — breaking changes may happen.

## Hosted API (demo)

Create a short URL:

```sh
curl -X POST https://api.versiy.cc \
  -H "Content-Type: application/json" \
  -d '{"original_url":"https://example.com"}'
```

## Local build

### Requirements

- Docker
- Docker Compose
- .env file (check .env.example)

```sh
docker compose -f docker-compose.dev.yml up --build
```

In a new terminal run:

```sh
docker compose exec app make migrate-up
```

Check health:

```sh
curl http://localhost:3000/health
```

## RoadMap

- A user sends a POST request to <https://api.versiy.cc/> with payload:

```json
{ "original_url": "https://example.com" }
```

- The server will return the shorten URL.

- When a user sends a GET request to:

```css
https://api.versiy.cc/{code}
```

- the server redirects the user to the original URL.

- On GET requests, the server:
  - checks the cache for the shortened URL
  - returns the cached value if it exists
  - otherwise fetches the URL from the database and caches it for future requests

## Contribution

Feel free to fork the project and experiment with it.
