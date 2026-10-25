# 通用能力包

通用能力统一放在 `pkg/<能力>/`，应用在 `bootstrap` 组装，在 `app` 使用。`pkg` 根目录仅作组织，不提供根包；不再设置 `utils`、`common` 等混合包。

## 按能力查找

| 包 | 职责 | 典型入口 |
| --- | --- | --- |
| httpx | 服务端参数绑定、校验、统一响应 | Handle、Bind、OK、Fail |
| httpclient | 出站 HTTP 请求、超时与链路透传 | NewHTTPClient、HandleResponse |
| validator | 自定义校验与中文提示 | Setup、Translate |
| errs | 业务错误、原因链、调用位置包装 | New、From、WithMessage |
| logx | slog 初始化、请求上下文及日志辅助 | Setup、New、ErrorLog、GetSelfFuncName |
| scheduler | 定时任务调度 | New、Every、Cron |
| syncx | goroutine 池、信号量、条件通知、原子值 | NewPool、NewLimit、NewCond、Boolean |
| filex | 文件与目录操作 | HasDir、CreateDir、ReadFileToBytes |
| hashx | 一致性哈希、摘要与校验和 | NewConsistentHash、Hash、Md5Hex |
| idgen | UUID、Snowflake ID | GetProUUID、New、NewWith |
| randx | 随机数与随机字符串 | NewRand、Randn、RandID、Seed |
| timex | 时间格式转换、区间检查、时间点补齐 | AgoUnixMilli、GetTimeSeries、FillMissingTimePoints |
| netx | IP 检查与本机网卡信息 | ValidateIP、GetIps、InternalIP |
| stringx | 字符转换与编码 | LowerFirst、Base64 |
| slicex | 切片包含、合并、去重、差集与求和 | FindDiff、SliceMerge、SliceUniqueInt |
| structx | 结构体字段复制 | CopyStructFields |
| mapx | map 构造辅助 | SetRetVal |
| cryptox | 密码哈希、兼容第三方接口的 HMAC 签名 | SetPassword、HmacSha1Encrypt |

`hashx` 的 MD5 仅用于校验和或兼容用途，不用于密码存储。普通随机数与 `randx.Randn` 不用于密钥或令牌；`RandID` 仍沿用原来的加密随机数实现。`cryptox.SetPassword` 保留原来的 bcrypt cost 配置。

## 旧路径迁移

| 原路径 / 函数 | 新位置 |
| --- | --- |
| utils/httpclient、utils/validator | pkg/httpclient、pkg/validator |
| utils/file、utils/hash、utils/slice | pkg/filex、pkg/hashx、pkg/slicex |
| utils/atomic、utils 的 Pool / Cond / Limit | pkg/syncx |
| utils/rand（Snowflake）、utils.GetProUUID | pkg/idgen |
| utils 的随机数、utils/strings 的 Rand / RandID / Randn / Seed | pkg/randx |
| utils 的时间、区间和时间点补齐函数 | pkg/timex |
| utils.GetIps / InternalIP / ValidateIP | pkg/netx |
| utils/strings.LowerFirst、utils.Base64 | pkg/stringx |
| utils.CopyStructFields | pkg/structx |
| utils.SetRetVal | pkg/mapx |
| utils.SetPassword / HmacSha1Encrypt | pkg/cryptox |
| utils.Md5Encrypt | pkg/hashx |
| utils.ErrorLog / GetSelfFuncName | pkg/logx |
| utils.WithMessage | pkg/errs |

本次移除旧 `utils` 导入路径，函数名、签名和运行时行为保留；项目外使用者需要更新 import 和对应包名前缀。

## 维护边界

- 新函数按实际能力归位，不重新创建杂项入口。
- pkg 不依赖 app、bootstrap 或生成器。生产包之间按需要依赖，例如 httpx 使用 errs 和 validator；避免循环依赖。
- UUID 与 Snowflake 属于 ID 生成；随机数与随机字符串属于随机能力，分别维护。
- 文件内按具体职责命名，如 timex/interval.go、timex/series.go、randx/number.go、randx/string.go。
- 所有测试放在 `test/cases/pkg/<包>/`，通过根目录 `make test`、`make race` 运行；修改包路径时同步检查 Makefile、模板及测试运行入口。
