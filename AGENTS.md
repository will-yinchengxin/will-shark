# Repository guidance

- 使用中文沟通；先阅读实际调用链、配置和测试再修改。保留用户已有改动，不提交真实凭证。
- Go 最低版本为 `go.mod` 中的补丁版本；不要为运行本项目修改全局 shell 配置。
- 全部 `*_test.go` 放在 `test/cases/`，按生产包相对路径组织，保留所属包名；夹具放 `test/fixtures/`。
- 执行 `make test` / `make race`，使用 `make vet` 检查含测试的包。直接 `go test ./...` 不会加载集中测试。
- 整理依赖用 `make tidy`，直接 `go mod tidy` 会遗漏集中测试依赖。
- 完成代码修改后运行 `make check`；涉及数据库或生成模板时，先 `docker compose up -d --wait`，再 `make integration`。
- `make cover` 统计全生产包覆盖率；`WS_INTEGRATION=1 make cover` 包含真实依赖路径。skip、未运行和环境阻塞不能报告成通过。
- `make fuzz` 覆盖六个外部输入入口，默认每个 45 秒；失败样本应收录到集中测试种子或夹具。
- 修改生成器必须保留 `willshark:gen:*` 插入点，并验证生成后 build、vet、测试及实际 CRUD 行为。
- 请求参数只能写入显式声明绑定 tag 的字段；路径参数优先。JSON 必须读至 EOF，并保留请求体大小限制。
- 依赖通过 `bootstrap.New` 创建并按逆序关闭，业务依赖在 `bootstrap.wire` 注入；不增加生产代码中的测试专用接口。
- SQL 查询必须参数化；写后读取走主库；日志不记录真实凭证或原始敏感请求体。
- 分布式锁只提供 TTL 有效期内的互斥；耗时任务必须响应 context，并为副作用设计幂等。
- push 需要用户授权；已明确要求 push 时完成验证后普通推送，不强制推送。
- 详细测试命令见 [test/README.md](test/README.md)。
