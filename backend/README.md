# Offer Hub 后端

Go + Gin 项目骨架，所有 Go 源码位于 `src/`。

调用方向：`router → ctrl → service → data`。`model` 存放请求和响应 DTO，`config` 读取配置。

## 启动

需要 Go 1.23 或更高版本，在 `backend` 目录执行：

```sh
go mod tidy
go run ./src
```

默认读取 `config-test.toml`，通过 `[COMMON].port` 配置监听端口，示例值为 `8180`。已注册 `GET /api/v1/question/all/list` 和 `GET /api/v1/question/list`，其他未注册路径返回 404。

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

配置模块仅负责读取数据，不执行完整性校验。数据库连接初始化已接入启动流程；登出接口已实现 JWT 验证，HTTPS 和限流逻辑尚未实现。

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

此处只建立连接，不创建数据库、迁移表结构。JWT 黑名单由鉴权业务层使用 Redis 管理，限流逻辑尚未实现。





## 题库全量列表

`GET /api/v1/question/all/list?job_name=后端开发`，`job_name` 可省略，游客可访问。

- data 层查询 `question_bank_series`、`question_bank`，精确匹配职位方向，按 `order` 升序（相同权重以对应 ID 排序）。
- 使用一次 Mongo 聚合，根据 `question.bank_list` 批量统计返回题库的题目数，只包含 `status = 1` 的题目。同一道题在同一题库只计数一次。
- service 层按职位方向、系列、题库组装；保留空系列，空列表返回 `[]`，没有正常题目的题库 `count` 为 0。
- ctrl 层绑定 query 并返回 HTTP 200，成功 `code: 0`；参数错误 `code: 400`；数据库错误 `code: 500, data: null`，错误详情只写服务日志。

数据库文档模型与 HTTP DTO 分别定义在 `src/data/question.go`、`src/model/question.go`，字段名依据数据字典。测试使用 Mongo 官方驱动的 mock deployment 验证查询命令，未连接真实 MongoDB。

## 题目列表

`GET /api/v1/question/list`：支持 `bank_id`、`keyword`、`difficulty`、重复的 `tags` 参数（如 `tags=Go&tags=数据库`）、`job_name`、分页和排序。

- 只返回正常题目（`status = 1`），请求不开放 `status` 过滤。
- `keyword` 对 `title`、`content` 进行不区分大小写的字面匹配，正则特殊字符会被转义。
- 多标签固定使用 `$all`：所有标签均匹配。分页默认 `page=1`、`page_size=20`。
- 接口文档未规定默认排序，本次约定为 `create_time desc`。允许按 `create_time`、`view_count`、`thumbs_up_count`、`dislike_count` 升序或降序，增加 `question_id` 升序作为同值排序依据。
- `total` 为分页前总数，统计和查询复用同一过滤条件；空结果返回 `list: []`。
- 本次实现游客行为，尚未接入软 JWT 鉴权：忽略 `user_tag` 过滤，`user_tag=0`、`user_liked=false`；service 层将 `content` 截断至前 150 个 Unicode 字符。登录用户的完整内容及个性化状态将在鉴权接入后实现。
- 非法分页、难度或排序参数返回 HTTP 200、业务码 400；数据库错误返回业务码 500，错误详情不向客户端暴露。

分层：ctrl 绑定并校验 query → service 调用查询并转换展示字段 → data 构建 BSON、统计与分页查询。BSON 文档结构使用数据字典中的字段和类型，数据库层不截断内容。


## 题目元数据列表

`GET /api/v1/question/meta/list`：参数、分页、排序、游客过滤规则与题目列表一致。
两个接口共享 ctrl 参数绑定与校验函数，以及 service 过滤入口和 data 的同一个 `FilterQuestions` 调用链，不维护第二套 BSON 条件。

响应为 `ListQuestionMetaResp`：成功 `code: 0, msg: "success"`，`data.total` 为符合条件的总数，`data.list` 每项仅包含 `question_id`、`title`；空列表返回 `[]`。
元数据服务不做正文截断或完整题目 DTO 转换；数据层仍复用现有文档查询，HTTP 响应不包含正文和其他题目字段。

## 题目详情

`GET /api/v1/question/detail?question_id=q001`，`question_id` 必填，空白 ID 不接受。

data 层只按 `question_id` 查询 `question`，本次不附加状态过滤，不自增浏览量；没有记录时返回可通过 `errors.Is` 判断的 `ErrQuestionNotFound`。
service 复用题目字段转换，按当前阶段要求返回完整 `content`，不执行游客 150 字截断，`user_tag=0`、`user_liked=false`。详情内容截断与个性化状态在鉴权模块接入时处理；列表接口原有截断规则保持不变。

成功返回 `GetQuestionDetailResp`（`code: 0, msg: "success", data: OneQuestion`）；参数错误返回业务码 400，题目不存在返回 404，数据库错误返回 500，错误时 `data: null`，HTTP 状态均为 200。详情查询、完整正文、错误分类和路由已由测试覆盖，Mongo 查询使用 mock deployment 验证。

## 热门题目列表

`GET /api/v1/question/hot/list?job_name=后端开发&limit=10`，`job_name` 可选，`limit` 省略时默认 10，显式传入时必须为正整数。

独立的 data 查询固定过滤 `status = 1`，可按 `job_name` 精确过滤，按 `hot_degree` 降序、同热度按 `question_id` 升序，不复用 `FilterQuestions` 的浏览量排序。Mongo 查询投影及接口列表项仅含 `question_id`、`bank_list`、`title`、`view_count`（Mongo `_id` 不返回）。

service 返回 `GetHotQuestionsResp`，复用已有热门列表 DTO，空结果返回 `data.list: []`。成功业务码 0，参数错误 400，数据库失败 500，HTTP 状态均为 200。查询命令、字段投影、默认 limit、过滤和错误响应由测试覆盖；尚未进行真实 MongoDB 联调。

## 用户注册

`POST /auth/register` 无需登录，JSON 请求体包含 `username`、`password`。用户名不可为空或全空白，最长 50 个 Unicode 字符；密码最少 6 个 Unicode 字符，受 bcrypt 限制最多 72 字节。密码使用 bcrypt cost 12 加密，`user_id` 为 UUID v4，昵称默认等于用户名。

在目标 MySQL 数据库中执行 `testdata/init_mysql_schema.sql` 创建 `t_user_info`（仅建表，无业务数据；不会迁移已有的 `user` 表）。启动流程不自动建表。用户名唯一索引同时处理并发注册冲突。

响应 HTTP 状态为 200，`data` 为 `null`：成功 `code: 0, msg: "注册成功"`；参数错误 `code: 400, msg: "用户名或密码不符合要求"`；用户名重复 `code: 500, msg: "用户名已存在"`；存储等内部错误 `code: 500, msg: "服务器内部错误"`。

## 用户登录

`POST /auth/login` 无需登录，JSON 请求体包含必填的 `username`、`password`。从 `t_user_info` 查询用户并用 bcrypt 校验密码，成功返回 `code: 0, msg: "登录成功"`，`data` 包含 `token` 和 `userInfo`。`userInfo` 使用认证文档规定的 camelCase 字段，包含 `userId`、`nickName`、`userStatus`、`userType` 等，不返回密码哈希。

JWT 使用 HS256 签名，密钥读取 `JWT.secret`，有效期读取 `JWT.expire`（小时），payload 包含 `user_id`、`iat`、`exp`。密钥为空、有效期非正数或超出可表示范围时返回内部错误。登录始终签发 token；`JWT.enable` 不控制登录响应。

响应 HTTP 状态为 200；参数错误返回业务码 `400`、`用户名或密码不符合要求`，用户不存在或密码错误统一返回 `401`、`用户名或密码错误`，内部错误返回 `500`、`服务器内部错误`。错误响应的 `data` 均为 `null`。

## 用户登出

`POST /auth/logout` 无请求体，必须携带 `Authorization: Bearer <token>`。校验 HS256 签名、`user_id`、有效期及 Redis 黑名单后，将 `blacklist:<token>` 写入 Redis，TTL 为 `exp` 减当前时间。原子 `SET NX` 避免并发登出覆盖已有记录的过期时间。

成功返回 HTTP 200、`{"code":0,"msg":"success","data":{"message":"登出成功"}}`。缺失、无效、过期或已拉黑的 token 返回业务码 `401`、`msg: "未认证"`；Redis 或 JWT 配置错误返回 `500`。错误响应的 `data` 为 `null`，HTTP 状态保持 200。

需配置 `JWT.secret` 与登录一致，并启用 `JWT.enable = true`、填写 Redis 连接配置，以便启动时初始化 Redis。`LogoutService.AuthenticateToken` 可供后续强制鉴权入口复用；目前登出接口接入黑名单校验，其他接口仍按各自现有鉴权行为执行。

## 获取用户信息

`GET /api/v1/user_info/get_user_info` 当前按阶段约定直接读取请求 Header 中的 `user_id` 作为测试身份；JWT 中间件接入后需切换为中间件提供的可信身份，此阶段不校验该接口的 Bearer token。缺失、空白或重复的 `user_id` Header 返回业务码 `401`。

按 `user_id` 查询 `t_user_info`，成功返回 `code: 0, msg: "success"`，`data` 包含认证文档列出的 14 个 snake_case 字段。`avatar_url` 与 `avatar` 相同，时间格式为 `YYYY-MM-DD HH:mm:ss`，空时间为 `""`；查询不读取密码。用户不存在返回业务码 `404`、`用户不存在`，数据库错误返回 `500`、`服务器内部错误`，错误的 `data` 为 `null`，HTTP 状态均为 200。

## 本地种子数据

参见 `testdata/README.md`。填写 `config-test.toml` 的 MongoDB 配置后，在 `backend/testdata` 执行 `npm.cmd ci --ignore-scripts`、`npm.cmd run seed`；脚本依次创建系列/题库和题目，可顺序重复执行。
