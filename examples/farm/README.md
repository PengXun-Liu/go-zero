# Farm Game — go-zero Example

This directory contains a **complete, runnable example** showing how to use the
[go-zero](https://github.com/zeromicro/go-zero) framework to build the backend
of a simple farm game.  
It demonstrates the most important framework features you will use in any
real-world go-zero service:

| Feature | Where it is used |
|---------|-----------------|
| REST server (`rest.Server`) | `farm.go` |
| Configuration loading (`conf.MustLoad`) | `farm.go` |
| JWT authentication (`rest.WithJwt`) | `internal/handler/routes.go` |
| Request parsing (`httpx.Parse`) | every handler |
| JSON responses (`httpx.OkJsonCtx`, `httpx.ErrorCtx`) | every handler |
| Redis client (`stores/redis`) | `internal/logic/` |
| Structured logging (`logx`) | every logic struct |
| Service context (dependency injection) | `internal/svc/servicecontext.go` |

---

## Architecture Overview

```
adhoc/farm/
├── farm.go                        ← main entry point
├── etc/farm.yaml                  ← configuration (host, port, redis, jwt)
└── internal/
    ├── config/config.go           ← Config struct
    ├── svc/servicecontext.go      ← shared dependencies (Redis, Config)
    ├── types/types.go             ← request / response structs
    ├── handler/
    │   ├── routes.go              ← route registration
    │   ├── user/userhandler.go    ← register & login handlers
    │   └── farm/farmhandler.go    ← plant / water / harvest / status handlers
    └── logic/
        ├── user/userlogic.go      ← business logic: registration & login
        └── farm/farmlogic.go      ← business logic: farm actions
```

The layered architecture follows go-zero conventions:

```
HTTP request → Handler → Logic → Storage (Redis)
```

- **Handler** – parses the request, calls logic, writes the response.
- **Logic** – contains all business rules; receives a `ServiceContext`.
- **ServiceContext** – holds shared dependencies (Redis connection, config).

---

## Prerequisites

- Go 1.23+
- A running Redis instance on `localhost:6379`

---

## Running the Example

```bash
# From the repository root
cd adhoc/farm

# Download dependencies (first time only)
go mod tidy

# Start the server (default: listens on :8888)
go run farm.go -f etc/farm.yaml
```

You should see:

```
Starting farm game server at 0.0.0.0:8888...
```

---

## API Reference

### Register a new user

```bash
curl -X POST http://localhost:8888/api/v1/user/register \
  -H "Content-Type: application/json" \
  -d '{"username":"farmer1","password":"secret"}'
```

Response:
```json
{"userId":"user:1700000000000000000","username":"farmer1"}
```

---

### Login and get a JWT token

```bash
curl -X POST http://localhost:8888/api/v1/user/login \
  -H "Content-Type: application/json" \
  -d '{"username":"farmer1","password":"secret"}'
```

Response:
```json
{
  "userId": "user:1700000000000000000",
  "username": "farmer1",
  "accessToken": "<JWT>",
  "expireAt": 1700086400
}
```

Copy the `accessToken` and use it as a Bearer token for all farm endpoints.

---

### View farm status  *(requires JWT)*

```bash
TOKEN="<your JWT from login>"

curl http://localhost:8888/api/v1/farm/status \
  -H "Authorization: Bearer $TOKEN"
```

Response:
```json
{
  "userId": "user:...",
  "plots": [
    {"plotId":0,"cropType":"","plantedAt":0,"harvestAt":0,"watered":false,"ready":false},
    ...
  ]
}
```

---

### Plant a crop  *(requires JWT)*

Valid crop types (you can add more in `farmlogic.go`): `wheat`, `corn`, `carrot`, etc.

```bash
curl -X POST http://localhost:8888/api/v1/farm/plant \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"plotId":0,"cropType":"wheat"}'
```

Response:
```json
{"plotId":0,"cropType":"wheat","plantedAt":1700000000,"harvestAt":1700000060}
```

The crop is ready to harvest after **60 seconds** (`cropGrowSeconds` in `farmlogic.go`).

---

### Water a crop  *(requires JWT)*

Watering a crop gives a **+50 % yield bonus** at harvest time.

```bash
curl -X POST http://localhost:8888/api/v1/farm/water \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"plotId":0}'
```

---

### Harvest a crop  *(requires JWT)*

You can only harvest once the crop is ready (`harvestAt <= now`).

```bash
curl -X POST http://localhost:8888/api/v1/farm/harvest \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"plotId":0}'
```

Response:
```json
{"plotId":0,"cropType":"wheat","yield":15}
```

(`yield` is 15 if watered, 10 if not.)

---

## Key go-zero Patterns Explained

### 1. Server setup

```go
// farm.go
var c config.Config
conf.MustLoad(*configFile, &c)        // load YAML config

server := rest.MustNewServer(c.RestConf)
defer server.Stop()                   // graceful shutdown
```

### 2. Route groups with JWT middleware

```go
// handler/routes.go — public routes (no auth)
server.AddRoutes([]rest.Route{
    {Method: http.MethodPost, Path: "/api/v1/user/login", Handler: ...},
})

// protected routes — JWT is enforced automatically
server.AddRoutes([]rest.Route{
    {Method: http.MethodGet, Path: "/api/v1/farm/status", Handler: ...},
}, rest.WithJwt(svcCtx.Config.Auth.AccessSecret))
```

`rest.WithJwt` validates the `Authorization: Bearer <token>` header and
injects all **custom JWT claims** (e.g. `userId`) into the request context.

### 3. Reading JWT claims from context

```go
// handler/farm/farmhandler.go
userId := r.Context().Value("userId").(string)
```

### 4. Service context (dependency injection)

```go
// svc/servicecontext.go
type ServiceContext struct {
    Config config.Config
    Redis  *redis.Redis
}
```

The `ServiceContext` is created once in `main` and passed to every handler and
logic struct, keeping dependencies explicit and testable.

### 5. Logic layer

```go
// logic/farm/farmlogic.go
type PlantLogic struct {
    ctx    context.Context
    svcCtx *svc.ServiceContext
    logx.Logger          // structured logging included automatically
}
```

---

## Extending the Example

Here are ideas for growing this into a fuller farm game:

| Feature | How to add it |
|---------|--------------|
| Persistent user storage | Replace Redis with MySQL via `stores/sqlx` |
| Multiple crop types with different grow times | Add a `cropConfigs` map in `farmlogic.go` |
| Inventory / coins system | Add a new Redis hash `inventory:<userId>` |
| Real-time notifications | Use SSE (`rest.WithSSE`) or add a WebSocket handler |
| Multiple game servers | Add service discovery with etcd + zRPC |
| Metrics & monitoring | Enable Prometheus in `etc/farm.yaml` (`Middlewares.Prometheus: true`) |
| Rate limiting | Use `core/limit.PeriodLimit` inside a handler |
| Circuit breaker | Enabled automatically by go-zero's built-in middleware |
