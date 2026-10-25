# HTTP 四层职责修正

用户批准：Router 只注册路由；Controller 管理 HTTP 输入输出；Service 管理业务；DTO 保留 app/do/request 与 app/do/response 命名。

- [x] 先增加回归：默认/单独 router 均生成 Controller；Controller 可单独生成及复用；实际 Apps 路由经过 Controller；已有响应与校验行为保持。
- [x] 增加 Apps Controller，路由注入 Controller，Service 不包含 HTTP/Swagger 职责；Apps 返回 response DTO，保持接口 JSON 字段一致。
- [x] 生成器默认 router/controller/service/dto；Router 永远绑定 Controller，依赖自动补齐；独立 Service/DTO/Job 保留非 HTTP 使用方式。
- [x] Swagger 注释迁到 Controller，更新生成模板、CLI 帮助、README、AGENTS，验证 Swagger 实际可生成且无重复接口。
- [x] 完整 build/vet/race、真实 MySQL/Redis 与生成 CRUD、覆盖率、静态扫描和独立只读复核。
- [x] 核对最终差异并准备提交；普通 push 结果以最终执行回报为准。

约束：保留原请求路径、HTTP 状态码、响应 JSON、校验、默认值和现有测试的业务断言。已有 bootstrap/app.go 的用户修改（删除旧文件头注释）保留。HTTP 公共工具仍使用 pkg/httpx.Bind/OK/Fail；httpx.Handle 保留作公共适配工具，但应用 Router 和生成路由不得绕过 Controller。

审查重点：router/controller/service/dto 依赖闭环、已有组件复用、controller 单独生成的未使用局部变量、Swagger 类型解析、旧路由直接注入 Service 时不得写入重复装配字段。

## 验证记录

- 新增职责回归先失败，再通过；默认模块和仅请求 router 均生成完整链路，独立 controller 后追加 router 可编译。
- make check 通过；含真实 MySQL/Redis、默认生成 CRUD、Controller 实际路由归属、Swagger schema 的完整 race/cover 通过，语句覆盖率 92.2%。
- Controller 与旧 httpx.Handle 保持 Request.Context、校验、错误与 nil 响应处理；User 响应 DTO 的字段类型与 JSON tag 保持一致，app/do 未改名。
- Swagger 严格解析校验生成模块的 list/create/get/update/delete 与请求/响应 schema；仅 Service 不产 HTTP 文档。根 Swagger 已实际重新生成，无警告和重复接口。
- 独立复核发现的旧多行 Router{Svc:...} 绕过问题已按 RED→GREEN 修复；现在按 AST 匹配导入路径及别名，自动接入拒绝旧装配且不写文件，-no-wire 可用于手动迁移。
- gofmt、vet、staticcheck 全规则、gosec、凭证扫描通过；govulncheck 为 0 个调用可达漏洞，保留此前不可达 OpenPGP 依赖公告。
- 复核结论：可合并，无剩余重要缺陷；既有组件动作集不一致等原有限制维持，不在本轮扩展。
