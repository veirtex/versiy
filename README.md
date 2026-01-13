# Versiy — URL Shortener API (Go + PostgreSQL + Redis)

Versiy is a URL shortening API built with Go, PostgreSQL, and Redis, designed with caching, rate limiting, and containerized deployment in mind.

> Status: **Under active development** — breaking changes may occur.

---

## Tech Stack

- **Go** — REST API and service logic
- **PostgreSQL** — persistent storage for shortened URLs
- **Redis** — caching and distributed rate limiting
- **Docker & Docker Compose** — local development and deployment

---

## Architecture Overview

1. A client sends a request to shorten a URL
2. The API applies rate limiting using Redis
3. The shortened URL is persisted in PostgreSQL
4. On redirect requests:
   - Redis cache is checked first
   - PostgreSQL is queried on cache miss
   - The result is cached for subsequent requests

This design minimizes database load while allowing the service to scale horizontally.

---

## Rate Limiting (Fixed Window)

Versiy implements a **fixed-size window rate limiting algorithm** backed by Redis.

- Limit: **10 requests per 15 seconds**
- Counters stored in Redis with TTL
- Atomic increments using Redis `INCR`
- Shared across multiple application instances

**Trade-offs:**:

- Allows short bursts at window boundaries
- Chosen for simplicity, predictability, and performance

---

## API

### Create Short URL

```sh
POST https://api.versiy.cc/
```

### Request body

```json
{ "original_url": "https://example.com" }
```

### Response

```json
{ "short_url": "https://api.versiy.cc/abc123" }
```

### Redirect

```sh
GET https://api.versiy.cc/{code}
```

- Redirects to the original URL.
- Uses Redis cache before falling back to PostgreSQL.

---

## Hosted API (Demo)

```sh
curl -X POST https://api.versiy.cc \
  -H "Content-Type: application/json" \
  -d '{"original_url":"https://example.com"}'
```

---

## Local Development

### Requirements

- Docker
- Docker Compose
- .env file (see .env.example)

Start services:

```sh
docker compose -f docker-compose.dev.yml up --build
```

Run migrations:

```sh
docker compose exec app make migrate-up
```

Health check:

```sh
curl http://localhost:3000/health
```

---

## Roadmap

- Custom aliases
- URL custom expiration
- Click analytics

---

## Contributing

Contributions are welcome.
Feel free to fork the repository and submit pull requests.