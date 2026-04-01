# Stockyard Spur

**API mock server.** Define endpoints and response shapes, serve fake data instantly. Every frontend dev needs this during development. Single binary, no external dependencies.

Part of the [Stockyard](https://stockyard.dev) suite of self-hosted developer tools.

## Quick Start

```bash
curl -sfL https://stockyard.dev/install/spur | sh
spur
```

## Usage

```bash
# Create a project
curl -X POST http://localhost:8950/api/projects \
  -H "Content-Type: application/json" \
  -d '{"name":"My API"}'

# Define mock endpoints
curl -X POST http://localhost:8950/api/projects/{id}/endpoints \
  -H "Content-Type: application/json" \
  -d '{"method":"GET","path":"/api/users","status_code":200,
       "response_body":"[{\"id\":1,\"name\":\"Alice\"}]"}'

# Hit the mock — returns your defined response
curl http://localhost:8950/mock/api/users
# → [{"id":1,"name":"Alice"}]

# All requests are logged for debugging
curl http://localhost:8950/api/endpoints/{ep_id}/log
```

Point your frontend at `http://localhost:8950/mock` as your API base URL during development.

## Free vs Pro

| Feature | Free | Pro ($2.99/mo) |
|---------|------|----------------|
| Projects | 2 | Unlimited |
| Endpoints | 10 | Unlimited |
| Request logging | ✓ | ✓ |
| Simulated latency | — | ✓ |
| Log retention | 7 days | 90 days |

## License

Apache 2.0 — see [LICENSE](LICENSE).
