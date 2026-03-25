# Product Review System

REST API for managing products and their reviews. Built in Go with PostgreSQL, Redis for caching, and Kafka for event notifications.

## How it works

Two services run together:

1. The API handles all HTTP requests: product and review CRUD, caching, and publishing events to Kafka when reviews change.
2. The Notifier is a small separate service that consumes those Kafka events and logs them. It shows how downstream services would pick up review changes in a real setup.

```
Client -> API (:8080) -> PostgreSQL
                      -> Redis (cache)
                      -> Kafka -> Notifier (logs events)
```

## Running it

You need Docker and Docker Compose. That's it.

```bash
docker compose up --build
```

Starts five containers: the API, notifier, Postgres, Redis, and Kafka. Takes about 30-40 seconds because Kafka is slow to boot. Once it's up:

```bash
curl http://localhost:8080/health
# {"status":"ok"}
```

To tear everything down including the database volume:

```bash
docker compose down -v
```

## API endpoints

### Products

```
POST   /api/v1/products         Create a product
GET    /api/v1/products         List all products
GET    /api/v1/products/:id     Get a product by ID
PUT    /api/v1/products/:id     Update a product
DELETE /api/v1/products/:id     Delete a product
```

Product responses include `average_rating` but not reviews. Reviews have their own endpoint.

### Reviews

```
POST   /api/v1/products/:id/reviews    Add a review to a product
GET    /api/v1/products/:id/reviews    List reviews for a product
PUT    /api/v1/reviews/:id             Update a review
DELETE /api/v1/reviews/:id             Delete a review
```

### Quick examples

Create a product:
```bash
curl -X POST http://localhost:8080/api/v1/products \
  -H "Content-Type: application/json" \
  -d '{"name": "Wireless Headphones", "description": "Noise-cancelling over-ear", "price": 149.99}'
```

Add a review:
```bash
curl -X POST http://localhost:8080/api/v1/products/{product_id}/reviews \
  -H "Content-Type: application/json" \
  -d '{"first_name": "John", "last_name": "Doe", "review_text": "Great sound quality!", "rating": 5}'
```

## Project structure

```
cmd/
  api/            main entry point for the API server
  notifier/       main entry point for the Kafka consumer
internal/
  cache/          Redis caching layer
  config/         env-based config
  events/         Kafka producer + consumer
  handler/        HTTP handlers
  model/          structs and shared error types
  repository/     database queries
  service/        business logic sits here
migrations/       SQL schema, runs on Postgres startup
```

Standard Go layout. Handlers call services, services call repositories. Nothing fancy.

## Design decisions

### Average rating calculation

This was the interesting part. I didn't want to compute `AVG(rating)` on every product read since that gets slow with lots of reviews. Instead there's a `product_ratings` table that stores the precomputed average. It gets updated inside a transaction whenever a review changes.

For concurrency safety I went with two layers:
- A `SELECT FOR UPDATE` lock on the rating row at the DB level, so concurrent transactions don't step on each other
- A per-product `sync.Mutex` in Go, which cuts down on unnecessary DB contention when multiple requests hit the same product

One mutex per product (stored in a `sync.Map`), not a global lock. The DB lock is the real safety net, the Go mutex is just to avoid hammering the database.

Worth noting: this only works within a single API instance. If you scale to multiple replicas, you'd want a distributed lock or just rely on the DB-level locking alone (which still guarantees correctness, just with more contention).

### Caching

Reviews list is cached in Redis with a 15-minute TTL. When a review is created, updated, or deleted, the cache gets invalidated and repopulated on the next read.

The average rating works differently. After recalculation, the new value is written directly to Redis so it's always fresh right after a review change. Two strategies for two types of data: write-invalidate where the full dataset would be expensive to recompute for cache, write-through where the value is already computed.

### Event notifications

Every review mutation publishes an event to Kafka (topic: `review-events`). Events are keyed by product ID, so all events for the same product go to the same partition and stay ordered.

The notifier service just logs these events. In production you'd have other consumers doing things like updating search indexes or sending notifications, all decoupled from the API.

### Error handling

- 400 for bad input (missing fields, invalid rating range, malformed JSON)
- 404 when a product or review doesn't exist
- 500 for unexpected errors

Foreign key violations (like trying to review a product that doesn't exist) come back as 404, not a raw database error.

## Tech choices

- Go, because the assignment suggested it and it fits well for this kind of service
- chi for routing, lightweight and works with the standard library
- PostgreSQL for relational data and ACID transactions during rating calculation
- Redis for caching reviews and ratings
- Kafka in KRaft mode for event streaming, no Zookeeper needed
- log/slog for structured logging, part of the standard library so no extra dependency

## What I'd add with more time

- Pagination on list endpoints
- Authentication
- OpenAPI/Swagger spec
- Integration tests
- Request rate limiting
