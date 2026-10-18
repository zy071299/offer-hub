# Offer Hub 后端

Go + Gin 项目骨架，所有 Go 源码位于 `src/`。

调用方向：`router → ctrl → service → data`。`model` 存放请求和响应 DTO，`config` 读取配置。

## 启动

需要 Go 1.23 或更高版本，在 `backend` 目录执行：

```sh
go mod tidy
go run ./src
```

默认读取 `config-test.toml`，通过 `[COMMON].port` 配置监听端口，示例值为 `8180`。当前未注册业务路由，请求返回 404。

## 配置

使用 `github.com/BurntSushi/toml` 读取 TOML。`TomlConfig` 包含 `Common`、`MySQL`、`MongoDB`、`Redis`、`JWT`、`RateLimit` 六节，成功读取后赋值给全局变量 `config.Conf`。

- `config.Init("path/to/config.toml")`：显式路径优先，不受 `APP_ENV` 影响。
- `config.Init("")`：读取 `config-{APP_ENV}.toml`；`APP_ENV` 未设置或为空时读取 `config-test.toml`。
- 相对路径基于进程当前工作目录。读取或解析失败时返回错误，保留原有 `Conf`。应在启动服务前初始化，运行期间只读。

PowerShell 环境选择示例（需先准备 `config-prod.toml`）：

```powershell
$env:APP_ENV = "prod"
go run ./src
```

`config.toml` 保留为通用模板，入口默认加载 `config-test.toml`。连接信息和 JWT 密钥需按环境填写。旧的 `HTTP_ADDR`、`MYSQL_DSN`、`MONGO_URI`、`MONGO_DATABASE` 环境变量不再用于配置。

配置模块仅负责读取数据，不执行完整性校验。数据库连接初始化已接入启动流程；HTTPS、JWT 验证和限流逻辑尚未实现。

## 检查

```sh
go test ./...
go vet ./...
```

## 数据库初始化

启动顺序：`config.Init("")` → `data.NewData(&config.Conf)`（并按需 `data.InitRedis(&config.Conf)`）→ `gin.Default()` → `rt.RegisterRouter(router)` → `router.Run()`。路由包以 `rt` 别名导入，端口读取 `config.Conf.Common.Port`。任何步骤返回错误或发生 panic 时，记录失败步骤并以非零状态退出；退出前清理已初始化的连接。
MySQL 与 MongoDB 必须连接成功；JWT 或限流开启时，Redis 也必须连接成功。示例配置中的空连接信息需先填写。

- MySQL `url` 为 `host:port`，通过 GORM 使用 MySQL 驱动连接，启用 `parseTime` 和 `utf8mb4`。
- MongoDB `url` 为 `mongodb://...` 或 `mongodb+srv://...` URI；使用官方 `go.mongodb.org/mongo-driver/mongo` 驱动，应用 `max_pool`、`min_pool`。`database` 指定业务数据库，通过 `data.GetData().MongoDB.Database(config.Conf.MongoDB.Database)` 访问。认证库使用 URI 的 `authSource`（未指定时遵循驱动默认规则）。
- Redis `url` 为 `host:port`，使用 `go-redis/v9`，应用 `pwd` 和 `db`，通过 `data.GetRedis()` 访问。

连接初始化均执行 Ping 并设置 5 秒连接超时；失败返回错误并清理本次已创建的连接。全部成功后才保存全局实例。重复初始化返回错误，初始化前 getter 返回 `nil`。连接应在服务启动前初始化，业务停止后调用 `Data.Close()` / `data.CloseRedis()`，不要与正在使用连接的请求并发关闭。

此处只建立连接，不创建数据库、迁移表结构，也不实现 JWT 黑名单或限流逻辑。



