# Versiy — URL Shortener API

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

Feel free to fork the project and experiment with it.