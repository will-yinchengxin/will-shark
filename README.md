# 🚀 Will-Shark

> A lightweight Go web framework (gin + gorm) for production services

[![Go Version](https://img.shields.io/badge/Go-1.26+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## ✨ Features

- MySQL (gorm) / Redis (go-redis) / RocketMQ (optional) connection management
- Typed config with environment variable overrides, validated at startup
- Structured logging with `log/slog` (JSON), `trace_id` / `request_id` in every log line
- Distributed tracing via OpenTelemetry OTLP (Jaeger / SkyWalking / ARMS / Tencent APM)
- Generic HTTP handler: binding + validation (Chinese messages) + unified response / error mapping
- Graceful shutdown: readiness drain → wait in-flight requests → stop jobs → close resources in reverse order
- `/healthz` and `/readyz` probes for Kubernetes
- Explicit dependency wiring in `bootstrap` (no code generation step)

## Local Development
> Go >= 1.26.10 is recommended (earlier 1.26.x patch releases have known stdlib vulnerabilities).

```bash
docker compose up -d        # MySQL 8 + Redis 7, seeded from sql/init
go run . dev                # loads envconfig/dev_config.yaml
go test -race ./...
WS_INTEGRATION=1 go test ./bootstrap -run Integration   # against docker compose
```

## Configuration
- File: `envconfig/<env>_config.yaml`; env = `$APP_ENV` > first CLI argument > `dev`
- Directory can be changed with `WS_CONFIG_DIR`
- **Environment variables override the file**: `WS_` + upper-cased key path, `.` → `_`
  (only keys present in the file or with defaults can be overridden; map keys are lower-cased)

```bash
WS_MYSQL_WILLSHARK_PASSWORD=xxx WS_HTTP_PORT=9000 ./willshark pro
```

- Durations need units: `500ms`, `30s`, `8h` (bare numbers are rejected)
- `uat` / `pro` configs are templates: hosts and passwords are empty and must be injected via env
  (e.g. K8s Secret), otherwise startup fails with a clear error
- See `envconfig/dev_config.yaml` for all keys: `app`, `http`, `log`, `mysql`, `redis`, `rocketmq`, `tracing`, `cors`

> Never commit real credentials. The dev config only contains local docker-compose credentials.

## Adding an Endpoint

1. Request / response DTOs in `app/do/request`, `app/do/response` (validation via `binding` tags, `label` for Chinese field names)
2. Business method with a typed signature in `app/service`:
   ```go
   func (s *Task) Create(ctx context.Context, req *request.CreateTask) (*response.Task, error) {
       if exists { return nil, errs.Conflict.WithMsg("任务已存在") }
       if err != nil { return nil, errs.Internal.Wrap(err) }   // cause is logged, not returned to client
       return &response.Task{ID: id}, nil
   }
   ```
3. One line in `app/router`: `g.POST("/task", httpx.Handle(api.Task.Create))`
4. Wire the dependency in `bootstrap/app.go`

`httpx.Handle` binds path (`uri` tag), query (`form` tag) and body (JSON / form), validates once,
and writes `{"code":200,"message":"请求成功","data":...}`. Errors from `pkg/errs` map to HTTP status + business code;
set `http.alwaysOK: true` if the frontend expects HTTP 200 for business errors.

Swagger annotations live on service methods; regenerate docs (served at `/swagger/index.html` when `app.swagger: true`):
```bash
go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g main.go \
  -d ./,./app/service,./pkg/httpx,./app/do,./app/modules/mysql/entity -o ./docs --parseInternal
```

## Docker
```bash
docker build -t willshark --build-arg VERSION=$(git rev-parse --short HEAD) .
# base images can be redirected to a private registry:
#   --build-arg GO_IMAGE=harbor.example.com/library/golang:1.26-alpine
docker run -e WS_MYSQL_WILLSHARK_HOST=... -e WS_MYSQL_WILLSHARK_PASSWORD=... \
           -e WS_REDIS_WILLSHARK_ADDR=... willshark pro
```

## Project Structure
```
main.go             # config → logger → bootstrap → run (signal handling)
bootstrap/          # dependency wiring, gin engine, graceful shutdown
envconfig/          # <env>_config.yaml + typed config package
core/               # infra constructors: mysql, redis, tracing, rocketmq
pkg/
├── errs/           # business errors (code + HTTP status + message + cause)
├── httpx/          # Handle / Bind / OK / Fail
└── logx/           # slog setup, trace_id / request_id from ctx
app/
├── middlewares/    # request id, tracing, access log, recovery, body limit, cors
├── router/         # route definitions
├── service/        # business logic
├── daemon_jobs/    # in-process cron jobs
├── modules/        # mysql (entity, dao), redis (wrapper, lock)
└── do/             # request / response DTOs
```

## Code Generator
> The bundled `gen` binary predates the current framework and its output does not compile against it.
> It will be replaced by a generator whose source lives in this repository.

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
