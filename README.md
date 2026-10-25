# 🚀 Will-Shark

> 基于 gin + gorm 的轻量级 Go Web 框架，面向生产环境，内置代码生成器

[![Go Version](https://img.shields.io/badge/Go-1.26+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## ✨ 特性

- **代码生成器**：按需组合生成 router / controller / service / dto / 定时任务，或从建表 SQL 生成 CRUD，并自动接入
- **MySQL**：单库 / 主从读写分离 / 主主（多主）；**Redis**：单机 / 集群 / 哨兵
- **定时任务**：间隔或 cron 表达式、分布式锁（多副本只执行一次）、超时、手动触发、依赖注入
- 强类型配置 + 环境变量覆盖，启动时校验
- `log/slog` 结构化日志（JSON），每条日志自动带 `trace_id` / `request_id`
- OpenTelemetry OTLP 链路追踪（Jaeger / SkyWalking / 阿里云 ARMS / 腾讯云 APM）
- 泛型 HTTP handler：参数绑定 + 中文校验提示 + 统一响应 / 错误映射
- 优雅退出：摘流 → 等待在途请求 → 停止定时任务 → 逆序关闭资源
- 探针：`/healthz`（存活）、`/readyz`（就绪：仅反映本进程是否启动完成、是否在退出中）；`/healthz/deps` 检查 MySQL / Redis 各节点，供监控告警使用，**不要**配置为 K8s 探针——共用依赖抖动时所有 Pod 会同时被摘流量

## 本地开发
> Go >= 1.26.10（go.mod 与 Docker 构建镜像均固定最低补丁版本）。

```bash
docker compose up -d        # MySQL 8 + Redis 7，自动执行 sql/init
go run . dev                # 加载 envconfig/dev_config.yaml
make race
make integration            # 连接 docker compose 的集成测试
```

## 代码生成器

所有测试集中在 `test/`，运行方式和 fuzz/覆盖率命令见 [测试说明](test/README.md)。统一使用 `make test` / `make race`，或 `go run ./test/run.go`；直接 `go test ./...` 不会加载集中测试。

源码在 `cmd/willshark-gen`、`internal/gen`，已通过 go.mod 的 `tool` 指令注册，无需单独安装：

```bash
go tool willshark-gen module <name> [flags]        # 生成业务模块，并自动接入 bootstrap 与路由
go tool willshark-gen model -sql <file> [flags]    # 只生成 entity + dao
go tool willshark-gen module -h                    # 查看全部参数
```

### 组件可任意组合（`-with`）

| 组件 | 生成文件 | 说明 |
|---|---|---|
| `router` | `app/router/<name>.go` | 挂载到 `/v1/<name>`，自动注册到 `router/base.go` |
| `controller` | `app/controller/<name>.go` | HTTP 输入输出：参数绑定、校验、错误映射、响应与 Swagger 注释 |
| `service` | `app/service/<name>.go` | 核心业务规则、流程编排与事务，使用普通 `context.Context` 和 DTO |
| `dto` | `app/do/request/<name>.go`、`app/do/response/<name>.go` | 请求 / 响应结构体，带 `binding` 校验 |
| `job` | `app/daemon_jobs/cron/<name>.go` | 定时任务，自动注入同名 service |
| `all` | 以上全部 | |

- 默认 `-with router,controller,service,dto`，默认接口 `-actions list,get,create,update,delete`
- **固定 HTTP 调用链**：Router → Controller → Service → DAO；请求/响应 DTO 保留在 `app/do/request`、`app/do/response`
- **依赖自动补齐**：router → controller → service → dto；**已存在且已接入的组件直接复用**（例如先生成模块，之后再单独加 `-with job`）。仍可单独生成 service/dto/job，用于非 HTTP 入口
- **安全**：任一目标文件已存在时整体中止、不写任何文件（`-force` 覆盖）；`-dry-run` 只预览；重复执行不会重复接入
- `-no-wire` 不修改 `bootstrap/app.go` 与 `app/router/base.go`（接入点是其中的 `// willshark:gen:*` 注释，请保留）
- 旧版本若在 bootstrap 中用 `Router{Svc: ...}` 直接绑定 Service，自动接入会拒绝旧装配，避免写入重复路由字段。可先用 `-force -no-wire` 生成四层文件，再手动增加 Controller 并将原路由字段改为 `Ctl: xxxCtl`，最后运行 `make check`

### 常用示例

```bash
# 1. 标准模块：router + controller + service + dto（CRUD 骨架）
go tool willshark-gen module order

# 2. 给已有模块追加定时任务（复用已有 service）
go tool willshark-gen module order -with job -every 10m

# 3. 全部组件 + 自定义接口 + cron 表达式
go tool willshark-gen module report -with all -actions list,export,daily-summary -cron "0 3 * * *"

# 4. 从建表语句生成完整可用的 CRUD（entity + dao + dto + service + controller + router）
go tool willshark-gen module task -sql sql/dynamic.sql
go tool willshark-gen module order -sql sql/tables.sql -table t_order -with all   # 模块名与表名不同

# 5. 只生成 entity + dao
go tool willshark-gen model -sql sql/dynamic.sql -tables ann_order,ann_log_statistics
```

### 效果

```text
$ go tool willshark-gen module task -sql sql/dynamic.sql -with all -every 2s
+ create  app/modules/mysql/entity/task.go
+ create  app/modules/mysql/dao/task.go
+ create  app/do/request/task.go
+ create  app/do/response/task.go
+ create  app/service/task.go
+ create  app/controller/task.go
+ create  app/router/task.go
+ create  app/daemon_jobs/cron/task.go
~ update  bootstrap/app.go (+6)
~ update  app/router/base.go (+2)
```

自动接入后的 `bootstrap/app.go`（`wire` 函数）：

```go
taskDao := &dao.Task{MysqlPool: pool}
taskSvc := &service.Task{Dao: taskDao}
taskCtl := &controller.Task{Svc: taskSvc}
routers := &router.Routers{
    Task: &router.TaskRouter{Ctl: taskCtl},
}
(&cron.TaskJob{Svc: taskSvc}).Register(sched)
```

生成后无需修改即可运行，接口如下（从 `sql/dynamic.sql` 的 `task` 表生成，已在真实 MySQL 上验证）：

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/v1/task?page=1&size=20` | 分页列表（按主键倒序），`size` ≤ 100，`page` ≤ 10000（更深的翻页请改用游标） |
| GET | `/v1/task/:id` | 详情，不存在返回 404 / code 9004 |
| POST | `/v1/task` | 创建；`NOT NULL` 且无默认值的列为必填（数值列生成指针类型，`0` 是合法值），`varchar(n)` 校验长度 |
| PUT | `/v1/task/:id` | 部分更新：只修改传入的字段；存在性检查与回读走主库 |
| DELETE | `/v1/task/:id` | 删除，不存在返回 404 |

```text
$ curl -XPOST localhost:8899/v1/task -d '{"cmd":"x..."}' -H 'Content-Type: application/json'
{"code":5003,"message":"args为必填字段; cmd长度不能超过255个字符","data":{}}

$ curl -XPUT localhost:8899/v1/task/1022 -d '{"cmd":"updated","pid":42}' -H 'Content-Type: application/json'
{"code":200,"message":"请求成功","data":{"id":1022,"cmd":"updated","pid":42,"load_method":"http",...}}
```

SQL 到 Go 的映射规则：
- 类型：`tinyint→int8`、`int→int32`、`bigint unsigned→uint64`、`decimal→string`（避免精度丢失）、`datetime→time.Time`、`blob→[]byte`、`bit(1)→bool`
- 可空列生成指针类型；`create_time/created_at` 等列自动加 `autoCreateTime`，`update_time/updated_at` 加 `autoUpdateTime`
- 非自增主键进入创建参数；整数主键允许 `0`，并显式禁止 GORM 将其当作自增列
- 带 SQL `DEFAULT` 的列在创建参数中区分省略与显式零值：省略时由数据库应用默认值，并从主库回读实际结果；显式 `0`、`false`、空字符串按传入值写入
- 字段名遵循 Go 惯例（`job_id→JobID`、`tenantId→TenantID`）；JSON 名保持列名；列注释作为字段注释和校验提示中的字段名
- 表必须有单列主键才能生成 CRUD；否则用 `model` 命令只生成 entity
- SQL 文件中的 `-- `、`#`、`/* */` 注释会被忽略

生成器提供开发骨架和通用 CRUD。认证、权限、租户隔离、业务事务与幂等需要按业务实现；当前生成接口未内置这些策略。更新 DTO 中的 `null` 表示不修改，若业务需要将列清空为 SQL NULL，需要单独设计该参数语义。

## 新增接口（手写）

1. 在 `app/do/request`、`app/do/response` 定义 DTO（`binding` tag 校验，`label` 作为中文字段名）
2. 在 `app/service` 写强类型方法：
   ```go
   func (s *Task) Create(ctx context.Context, req *request.CreateTask) (*response.Task, error) {
       if exists { return nil, errs.Conflict.WithMsg("任务已存在") }
       if err != nil { return nil, errs.Internal.Wrap(err) }   // 原因只进日志，不返回给前端
       return &response.Task{ID: id}, nil
   }
   ```
3. 在 `app/controller` 管理 HTTP 输入输出，并在方法上写 Swagger 注释：
   ```go
   func (c *Task) Create(ctx *gin.Context) {
       var req request.CreateTask
       if err := httpx.Bind(ctx, &req); err != nil {
           httpx.Fail(ctx, err)
           return
       }
       resp, err := c.Svc.Create(ctx.Request.Context(), &req)
       if err != nil {
           httpx.Fail(ctx, err)
           return
       }
       if resp == nil {
           httpx.OK(ctx, nil)
           return
       }
       httpx.OK(ctx, resp)
   }
   ```
4. Router 只注册路径、方法和中间件：`grp.POST("", r.Ctl.Create)`
5. 在 `bootstrap/app.go` 的 `wire` 中组装 `DAO → Service → Controller → Router`；定时任务直接复用 Service

Controller 调用 `httpx.Bind` 依次绑定查询参数（`form`）、请求体（JSON / 表单）、路径参数（`uri`，优先级最高），统一校验一次；
查询参数 / 表单 / 路径参数只绑定显式声明了 `form` / `uri` tag 的字段（gin 默认会按字段名绑定未打 tag 的字段，例如 `json:"-"` 的内部字段），
输出 `{"code":200,"message":"请求成功","data":...}`。`pkg/errs` 的错误映射为 HTTP 状态码 + 业务码
（底层原因是客户端断开导致的 ctx 取消时返回 499 / code 4990，不记 error 日志；超时返回 504 / code 9005）；
前端只认 HTTP 200 时设置 `http.alwaysOK: true`。

Swagger 注释写在 Controller 方法上，更新文档（`app.swagger: true` 时访问 `/swagger/index.html`）：
```bash
go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g main.go \
  -d ./ -o ./docs --parseInternal
```

## 定时任务

任务是一个**持有依赖的普通结构体**，在 `bootstrap.wire` 中注入 Service，因此任务里可以直接调用业务方法读写数据库，不经过 HTTP Controller，也能像普通对象一样单测：

```go
// app/daemon_jobs/cron/order.go
type OrderJob struct {
    Svc *service.Order // 由 bootstrap 注入
}

func (j *OrderJob) Register(s *scheduler.Scheduler) {
    s.Every("order_cleanup", 5*time.Minute, j.Run,
        scheduler.WithLock(time.Minute),        // 多副本只执行一次（Redis 分布式锁）
        scheduler.WithTimeout(30*time.Second))  // 单次执行超时，超时后 ctx 取消
    // 或 s.Cron("order_report", "0 3 * * *", j.Report)  // 支持 6 段（含秒）与 @daily、@every 1h
}

func (j *OrderJob) Run(ctx context.Context) error {
    return j.Svc.CleanExpired(ctx)
}

// bootstrap/app.go → wire()
(&cron.OrderJob{Svc: orderSvc}).Register(sched)
```

调度器（`pkg/scheduler`）的行为：
- `Every` 按固定间隔对齐到时间刻度执行（如 `5m` → 每小时的 :00、:05…，支持亚秒间隔），各副本的刻度一致；同一任务在进程内串行执行，不会重叠，上一轮超过间隔未结束时跳过错过的刻度
- `WithLock` 多副本部署时每个调度刻度只有一个实例执行：互斥锁 `cron:<app.name>:<任务名>` 防止并发执行，另按刻度占位 `cron:<app.name>:<任务名>:<刻度纳秒>`（不主动释放，ttl 到期删除），避免任务很快结束后时钟稍慢的副本重复执行；ttl 应大于最长执行时间与副本间时钟偏差；获取不到锁时本轮跳过
- `WithRunOnStart` 启动后立即执行一次
- `sched.Trigger(name)` 手动触发（例如做一个管理接口）：通过 channel 通知任务循环，不阻塞调用方，执行期间的重复触发会合并
- panic 被隔离并记录堆栈；每次执行一个 trace span，并记录耗时与错误日志
- 退出时不再开始新一轮，并等待正在执行的任务结束（最长 `http.shutdownTimeout`）

## MySQL 读写分离 / 主主

```yaml
mysql:
  willshark:
    host: mysql-primary          # 主库
    port: 3306
    user: willshark
    password: ""                 # 通过 WS_MYSQL_WILLSHARK_PASSWORD 注入
    database: willshark
    policy: round_robin          # random（默认）| round_robin
    replicas:                    # 从库：非事务 SELECT 走从库
      - host: mysql-replica-1
      - host: mysql-replica-2
        user: readonly           # user/password/database/port 不填则继承主库
        password: ""
    sources:                     # 额外主库（主主/多主）：写请求在主库与 sources 间分配
      - host: mysql-primary-2
```

- 写操作、事务（含事务中的读）走主库；非事务 `SELECT` 走从库
- 写后立即读等强一致场景用 `pool.Master()`（生成的 dao 提供 `GetFromMaster`）：
  ```go
  d.Master().WithContext(ctx).First(&m, id)
  ```
- 每个节点独立连接池，启动时逐一 ping；`/healthz/deps` 检查所有节点，任一节点不可用时返回 503
- 多主同时写入需配置 `auto_increment_increment/offset` 避免主键冲突，通常建议单写
- 基于 GORM 官方插件 `gorm.io/plugin/dbresolver`

## Redis 集群 / 哨兵

```yaml
redis:
  willshark:
    mode: cluster                       # single（默认）| cluster | sentinel
    addrs: [n1:6379, n2:6379, n3:6379]  # 集群种子节点，部分节点即可
    readFromReplica: true               # 只读命令路由到从节点
    password: ""
  # 哨兵：mode: sentinel，addrs 为哨兵地址，masterName: mymaster
```

- 三种模式统一返回 `redis.UniversalClient`，业务代码和分布式锁无需区分
- 集群模式只支持 `db: 0`；`/healthz/deps` 会逐个检查集群主节点
- 列表类配置也可用环境变量覆盖：`WS_REDIS_WILLSHARK_ADDRS=n1:6379,n2:6379,n3:6379`

## RocketMQ

`rocketmq.enable: true` 时 `bootstrap.Infra.MQ` 提供已启动的 `Producer`；消费者先 `MQ.Consumer.Subscribe(...)`，再调用 `MQ.StartConsumer()`（不要直接调用 `Consumer.Start()`，否则退出时不会关闭消费者）。

## 配置

- 文件：`envconfig/<env>_config.yaml`；env 取值优先级：`$APP_ENV` > 第一个命令行参数 > `dev`
- 配置目录可用 `WS_CONFIG_DIR` 修改
- **环境变量覆盖配置文件**：`WS_` + 大写的配置路径，`.` 换成 `_`；文件中没写的键也可以覆盖（如 pro 模板中没有的 `WS_REDIS_WILLSHARK_MODE`），但 mysql / redis 的连接名必须在文件中出现（map 的键会被转为小写），`replicas` / `sources` 列表不支持环境变量

```bash
WS_MYSQL_WILLSHARK_PASSWORD=xxx WS_HTTP_PORT=9000 ./willshark pro
```

- 时长必须带单位：`500ms`、`30s`、`8h`（裸数字会被拒绝）
- `uat` / `pro` 是模板：主机和密码为空，必须通过环境变量注入（如 K8s Secret），否则启动时报错退出
- 全部配置项见 `envconfig/dev_config.yaml`：`app`、`http`、`log`、`mysql`、`redis`、`rocketmq`、`tracing`、`cors`

> 不要把真实凭证提交到仓库。dev 配置里只有本地 docker compose 用的凭证。

## Docker

```bash
docker build -t willshark --build-arg VERSION=$(git rev-parse --short HEAD) .
# 基础镜像可指向私有仓库：
#   --build-arg GO_IMAGE=harbor.example.com/library/golang:1.26-alpine
docker run -e WS_MYSQL_WILLSHARK_HOST=... -e WS_MYSQL_WILLSHARK_PASSWORD=... \
           -e WS_REDIS_WILLSHARK_ADDR=... willshark pro
```

镜像默认以 `pro` 环境启动：未注入数据库等配置时校验失败并退出，避免把 dev 配置（swagger、debug 日志、记录请求体）带上线。
本地 `docker compose` 的 MySQL / Redis 只监听 `127.0.0.1`。

## 目录结构

```
main.go             # 配置 → 日志 → bootstrap → 运行（信号处理）
bootstrap/          # 依赖组装（wire）、gin 引擎、优雅退出
envconfig/          # <env>_config.yaml + 强类型配置
core/               # 基础设施：mysql（读写分离）、redis（单机/集群/哨兵）、tracing、rocketmq
cmd/willshark-gen/  # 代码生成器入口
internal/gen/       # 代码生成器实现与模板
test/               # 集中测试、原生 overlay 运行入口（make test / make race）
pkg/
├── errs/           # 业务错误（业务码 + HTTP 状态码 + 提示 + 内部原因）
├── httpx/          # Handle / Bind / OK / Fail
├── httpclient/     # 出站 HTTP 请求、超时与链路透传
├── validator/      # 参数校验、中文提示
├── logx/           # slog 初始化，从 ctx 提取 trace_id / request_id
├── scheduler/      # 定时任务调度
├── syncx/          # goroutine 池、限流信号量、条件通知、原子值
├── filex/          # 文件与目录
├── hashx/          # 一致性哈希、摘要与校验和
├── idgen/          # UUID、Snowflake
├── randx/          # 随机数、随机字符串
├── timex/          # 时间转换、区间、时间点补齐
├── netx/           # IP、网卡
├── stringx/        # 字符串转换、编码
├── slicex/         # 切片集合操作
├── structx/        # 结构体字段复制
├── mapx/           # map 构造辅助
└── cryptox/        # 密码哈希、HMAC 签名
app/
├── middlewares/    # request id、链路追踪、访问日志、recovery、body 限制、cors
├── router/         # 路由
├── controller/     # HTTP 输入输出、校验、错误映射与 Swagger
├── service/        # 业务逻辑
├── daemon_jobs/    # 定时任务
├── modules/        # mysql（entity、dao）、redis（封装、分布式锁）
└── do/             # 请求 / 响应 DTO
```

## 🤝 参与贡献

通用能力统一放在 `pkg/<能力>/`，包职责及旧路径迁移表见 [pkg/README.md](pkg/README.md)。

欢迎提交 Pull Request。

## 📝 License

MIT License，详见 [LICENSE](LICENSE)。
