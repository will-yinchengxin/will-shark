# pkg 能力包整合计划

> 执行方式：superpowers:executing-plans；用户已确认上一轮给出的目录和迁移范围。

**Goal:** 将 utils 整合到 pkg，按能力组织，保持现有行为与集中测试能力。

**Architecture:** 保留 errs/httpx/logx/scheduler，增加 httpclient/validator/syncx/filex/hashx/idgen/randx/timex/netx/stringx/slicex/structx/mapx/cryptox。不存在 pkg 根包或新的 common/utils 杂项包。

**Tech Stack:** Go >= 1.26.10，现有 Gin/GORM 与 Go 原生 overlay 测试入口。

**Spec:** 用户已批准的目录提案；全部测试仍放 test/cases，完成后提交并 push。

## Global Constraints

- 不改变函数签名、默认值、校验或运行时行为，不增加旧路径兼容层。
- 业务和生成代码的 import 同步迁移，保留 willshark:gen 插入点。
- 集中测试镜像生产包目录，保持历史测试断言及六个 fuzz 入口。
- 不修改全局配置，不引入新依赖。

## Review Focus

- 混合文件拆分时是否丢失函数、私有辅助函数、常量和测试断言。
- runtime.Caller、随机数共享源、时间辅助函数的行为是否保持。
- bool 与 pool 合并到 syncx 后是否引入命名冲突或依赖循环。
- overlay、Makefile、生成器副本测试及文档是否仍指向存在的生产包。
- 包导入路径是否全部迁移，生成器真实 CRUD 是否仍通过。

## Tasks

- [x] 1. 记录基线测试入口与导出 API，迁移独立包：file/hash/slice/httpclient/validator/atomic/rand(Snowflake)。
- [x] 2. 拆分 utils 根包及 strings：common→errs/logx/structx/cryptox；encode→stringx/hashx/cryptox；pool/cond/limit→syncx；time/interval/fill→timex；IP→netx；UUID→idgen；随机数和随机字符串→randx；ret→mapx。
- [x] 3. 同步集中测试包名、镜像路径和跨能力调用；保持 Test/Fuzz 入口及所有原断言；移除空 utils 目录。
- [x] 4. 更新调用方、模板、Makefile fuzz 路径、README 和仓库规则，运行 tidy -diff 检查依赖稳定。
- [x] 5. 运行 build/vet/race、真实 MySQL/Redis/生成器 CRUD、覆盖率和迁移 fuzz；独立只读复核。
- [x] 6. 检查最终差异，准备经验证的提交；普通 push 与远端核对结果以最终执行回报为准。

## Execution notes

- 基线：6f55fe1；开工时工作树干净。
- 包路径属于本次有意变更；项目外使用者需按迁移表更新 import，导出函数名称保留。
- AST 对比结果：240 个生产声明完全一致；测试跨包前缀归一化后 235 个声明一致。186 Test / 6 Fuzz 入口逐项一致，测试按能力拆分为 48 个文件。
- 生成器模板本来只引用 pkg/httpx、pkg/errs 和 pkg/scheduler，目标包未移动，因此无需无意义修改模板文本；副本构建和真实 CRUD 验收通过。
- make check、make integration、含集成的完整 race/cover 已通过，覆盖率 92.3%；tidy -diff 无差异。gofmt、staticcheck 全规则、gosec 与凭证扫描通过；govulncheck 0 个可达漏洞，保留既有不可达 OpenPGP 模块公告。
- 最终六项 fuzz 各 45 秒全部通过，共 19,057,345 次输入。
- 独立只读复核结论：可合并，无待修复项。旧导入路径/类型身份变化属于批准的迁移；历史混合测试保留跨能力调用，不减少断言。
