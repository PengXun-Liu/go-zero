# Farm Game Example — go-zero Framework Tutorial

This directory contains a fully-working farm game **backend** that shows step-by-step
how to build a service using [go-zero](https://github.com/zeromicro/go-zero).

> 本示例代码演示了如何使用 go-zero 框架搭建一个农场游戏的服务端。  
> (This example demonstrates how to use the go-zero framework to build a farm game server.)

---

## What the example covers

| go-zero concept | Where to look |
|---|---|
| `rest.MustNewServer` + `RestConf` | `main.go` |
| Route registration (`AddRoutes`) | `main.go` |
| JSON request parsing (`httpx.Parse`) | `handler/handler.go` |
| JSON response writing (`httpx.OkJsonCtx`) | `handler/handler.go` |
| Error responses (`httpx.ErrorCtx`) | `handler/handler.go` |
| Redis store (`stores/redis`) | `logic/farmlogic.go` |
| Layered architecture: config / handler / logic | whole example |
| Configuration from YAML (`conf.MustLoad`) | `main.go` |

---

## Game API

```
GET  /crops                           List available crop types
GET  /farm/:playerID                  Get a player's farm state
POST /farm/:playerID/plant            Plant a crop on an empty plot
POST /farm/:playerID/water/:plotID    Water a growing plot (−20 % grow time)
POST /farm/:playerID/harvest/:plotID  Harvest a ready crop and earn coins
```

### Crop catalog

| Crop | Grow time | Coins earned |
|------|-----------|--------------|
| wheat | 30 s | 10 |
| carrot | 45 s | 15 |
| corn | 60 s | 20 |

---

## Prerequisites

* Go 1.23+
* Redis (local or Docker)

```bash
# Start a local Redis with Docker
docker run -d -p 6379:6379 redis:7
```

---

## Run the server

```bash
# from the repository root
go run ./example/farmgame -f example/farmgame/farmgame.yaml
```

The server listens on `:8888` by default.

---

## Quick-start walkthrough

```bash
# 1. List available crops
curl http://localhost:8888/crops

# 2. Create / view your farm (auto-created on first access)
curl http://localhost:8888/farm/alice

# 3. Plant wheat on plot 0
curl -X POST http://localhost:8888/farm/alice/plant \
     -H 'Content-Type: application/json' \
     -d '{"plot_id": 0, "crop": "wheat"}'

# 4. Water plot 0 to speed up growth
curl -X POST http://localhost:8888/farm/alice/water/0

# 5. Wait 30 seconds (or less after watering), then harvest
curl -X POST http://localhost:8888/farm/alice/harvest/0

# 6. Check your updated coin balance
curl http://localhost:8888/farm/alice
```

---

## Code walkthrough

### 1 — Configuration (`main.go`)

```go
type Config struct {
    rest.RestConf          // HTTP server settings embedded from go-zero
    Redis  redis.RedisConf // Redis connection settings
}

var c Config
conf.MustLoad(*configFile, &c)  // load from YAML
```

`rest.RestConf` carries `Host`, `Port`, `Timeout`, `MaxConns`, and all built-in
middleware toggles.  Embedding it gives your service automatic logging, metrics,
circuit-breaking, and load shedding — for free.

### 2 — Server bootstrap (`main.go`)

```go
server := rest.MustNewServer(c.RestConf)
defer server.Stop()      // graceful shutdown on SIGTERM / SIGINT

server.AddRoutes([]rest.Route{
    {Method: "GET",  Path: "/farm/:playerID", Handler: handler.GetFarmHandler(fl)},
    // ...
})

server.Start()           // blocks until shutdown signal
```

Routes follow Go's `net/http` path-value syntax (`:param`).

### 3 — Handlers (`handler/handler.go`)

Each handler is a factory function that closes over the logic layer:

```go
func GetFarmHandler(fl *logic.FarmLogic) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        playerID := r.PathValue("playerID")
        farm, err := fl.GetFarm(r.Context(), playerID)
        if err != nil {
            httpx.ErrorCtx(r.Context(), w, err)
            return
        }
        httpx.OkJsonCtx(r.Context(), w, farm)
    }
}
```

**Key httpx helpers:**

| Function | Description |
|---|---|
| `httpx.OkJsonCtx(ctx, w, v)` | 200 JSON response |
| `httpx.ErrorCtx(ctx, w, err)` | error response (default 500) |
| `httpx.Parse(r, &req)` | decode JSON body + path params |

### 4 — Business logic (`logic/farmlogic.go`)

The logic layer holds all game rules and persists state to Redis:

```go
type FarmLogic struct {
    rds   *redis.Redis
    nowFn func() time.Time   // injectable clock (for tests)
}

// Each player's farm is serialised as JSON and stored under "farm:<playerID>".
func (l *FarmLogic) Plant(ctx context.Context, playerID string,
    plotID int, crop types.CropKind) (types.Plot, error) { ... }
```

go-zero's Redis client (`core/stores/redis`) wraps `go-redis` and provides:
- context-aware methods (`GetCtx`, `SetCtx`, …)
- built-in metrics and tracing
- cluster & sentinel support via config

### 5 — Configuration file (`farmgame.yaml`)

```yaml
Name: farmgame
Host: 0.0.0.0
Port: 8888
Timeout: 5000   # ms

Redis:
  Host: 127.0.0.1:6379
  Type: node
```

---

## Extending this example

| Feature | How to add it |
|---|---|
| **User authentication** | Add `rest.WithJwt("secret")` to `AddRoutes` options |
| **Leaderboard** | Use `redis.Zadd` / `redis.Zrange` (sorted sets) in a new logic method |
| **MySQL persistence** | Swap Redis for `sqlx.SqlConn` or `sqlc.CachedConn` |
| **gRPC service** | Create a `zrpc` server alongside the REST server |
| **Rate limiting** | Add `rest.WithPeriodLimit` or use `core/limit` in logic |
| **More middleware** | Enable in YAML: `Middlewares: {Breaker: true, Shedding: true}` |

---

## Project structure

```
example/farmgame/
├── farmgame.yaml       # sample YAML configuration
├── main.go             # server bootstrap & route registration
├── handler/
│   └── handler.go      # HTTP handlers (thin layer, delegates to logic)
├── logic/
│   ├── farmlogic.go    # game rules + Redis persistence
│   └── farmlogic_test.go
└── types/
    └── types.go        # domain model & request/response DTOs
```

The layered architecture (`handler → logic → store`) is the standard go-zero
pattern recommended for all services.
