# 测试

全部测试源码放在 `test/cases/`，按生产包的相对路径组织，例如：

```
test/cases/pkg/httpx/httpx_test.go
test/cases/internal/gen/generator_test.go
test/cases/bootstrap/integration_test.go
```

这些是包含私有函数测试的同包测试。Go 原生测试要求它们在所属包中编译，因此 `test/run.go` 使用标准 `go -overlay` 在内存中恢复所属路径；磁盘上不生成旁路测试文件、不修改生产源码。`test/go.mod` 隔离测试存储目录，防止 Go 把它当成另一套业务包。

统一使用以下入口，**直接执行 `go test ./...` 不会加载集中测试，不能作为验收结果**。IDE 请配置同样的运行命令。

```bash
make test                       # 完整回归（含生成器在仓库副本上的 build / vet / 测试）
make race                       # 完整 race 回归
make vet                        # 包括同包测试源码
make tidy                       # 保留集中测试依赖；替代直接 go mod tidy
make check                      # build + vet + race
make cover                      # 全生产包覆盖率，输出 coverage.out
docker compose up -d --wait
make integration                # 真实 MySQL/Redis；未启用 WS_INTEGRATION 的普通回归会 skip
WS_INTEGRATION=1 make cover      # 包含集成路径的覆盖率
make fuzz                       # 六个入口各 45 秒
make fuzz FUZZTIME=10s

# 参数转交给 go test；包名仍是生产包路径
go run ./test/run.go ./pkg/httpx -run TestJSON -count=1 -v
go run ./test/run.go -list 'Test|Fuzz' ./...
go run ./test/run.go ./internal/gen -fuzz '^FuzzParseSQL$' -fuzztime=45s -run '^$'
```

runner 会在临时目录生成绝对路径 overlay，转交测试退出码，结束后删除临时文件。添加测试时沿用原包的 `package` 名称和相对路径；测试夹具放在 `test/fixtures/`。如 fuzz 发现失败样本，Go 会按原包路径保存 `testdata/fuzz`；应把样本收录为集中测试的种子/夹具，再删除原包中的临时样本目录。

整理依赖使用 `make tidy`。直接 `go mod tidy` 同样看不到集中测试，会移除仅供测试使用的依赖；生成器副本验收会先执行带 overlay 的 tidy，再构建和测试，覆盖这一条路径。

`WS_INTEGRATION=1` 会连接配置中指定的真实开发依赖，并创建/删除测试表；只对本地或独立的测试数据库运行。
