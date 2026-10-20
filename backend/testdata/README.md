# 本地 MongoDB 种子数据

两个入口脚本使用 MongoDB 官方 Node.js 驱动，字段对齐 `src/data/question.go` 与数据字典。题库、系列、题目均存 MongoDB，无需向 MySQL 写入；它们之间是应用层关联，并非 SQL 外键。

## 安装与运行

需要 Node.js 18 或更新版本。先填写 `backend/config-test.toml` 的 `[MongoDB]` 节，再在项目根目录执行：

```powershell
cd backend/testdata
npm.cmd ci --ignore-scripts
npm.cmd run seed
```

`seed` 顺序执行以下两个脚本，第一个失败时不会执行第二个：

```powershell
node insert_bank_and_series.js
node insert_question.js
```

默认配置路径通过脚本目录定位到 `backend/config-test.toml`，不依赖启动目录。项目目前不存在 `config/config-test/toml`。如需指定其他配置，可给两个脚本传入相同路径：

```powershell
node insert_bank_and_series.js ../config-test.toml
node insert_question.js ../config-test.toml
```

脚本读取 `MongoDB.url`、`user`、`pwd`、`database`、`max_pool`、`min_pool`。URI 交由官方驱动解析，保留 `authSource` 等选项；显式凭证优先。连接及服务选择超时均为 5 秒，与 Go 初始化配置保持一致。脚本始终读取测试配置或显式路径，不使用 `APP_ENV` 切换环境。

## 数据与可重复执行规则

- 4 个系列、7 个题库、30 道题，覆盖后端开发、前端开发、AI开发；27 道正常题、2 道隐藏题、1 道删除题。
- 第一个脚本先 upsert `question_bank_series`，再 upsert `question_bank`。
- 第二个脚本批量检查全部题库、对应系列及职位关联；发现缺失或冲突时，在写入任何题目前退出，提示先执行第一个脚本。
- 使用 `seed_` 前缀的稳定业务 ID，通过 `updateOne + upsert` 写入，按顺序重复运行不会新增相同 ID 的记录。不执行清库、删除其他记录或修改索引。不要并发运行多个种子任务。
- 再次执行会把同 ID 的种子字段（包括正文、状态、计数和更新时间）恢复为固定测试值；`create_time` 使用 `$setOnInsert`，已有记录的创建时间与 `_id` 保留。原数据库需保证业务 ID 唯一。
- 新文档的 `_id` 由 MongoDB 生成为 ObjectID；`order` 显式使用 BSON Int64，时间使用 BSON Date，其余字段按数据字典。`user_tag`、`user_liked` 属于动态响应字段，不写入题目集合。
- 写入按集合分批完成，不使用跨集合事务；中途失败可修复后重新执行。日志打印新增、匹配、修改数量，不输出 URI 或密码。异常时退出码为 1，连接在 finally 中关闭。

## 可验证场景

- `/api/v1/question/all/list`：系列/题库顺序、职位过滤、空题库 `seed_bank_empty` 的 `count=0`。
- `/api/v1/question/list?page=1&page_size=20` 和 `page=2`：正常题目分页，隐藏/删除题目不返回。
- `bank_id=seed_bank_mysql`、`difficulty=2`、`job_name=后端开发`、`tags=Redis&tags=数据库`：过滤条件与多标签 `$all`。
- `keyword=redis`：忽略大小写；`keyword=Go.*`：正文中的字面关键词；长正文 `seed_q_001`：验证列表前 150 字与详情完整内容。
- 跨题库题目：`seed_q_006` 属于 Redis 和 MySQL，`seed_q_025` 属于 Go 和 Redis。
- `/api/v1/question/meta/list`：只返回 ID 和标题。
- `/api/v1/question/detail?question_id=seed_q_001`：正常详情；不存在的 ID：404 业务码。
- `/api/v1/question/hot/list`：正常题中热度最高为 `seed_q_001`，浏览量最高为 `seed_q_002`。隐藏/删除题热度更高但不能出现在热门列表。

仅统计本批种子数据时，正常题数：MySQL 6、Redis 5、Go 6、JavaScript 5、Vue 3、机器学习 4、空题库 0。跨题库题目会分别计入所属题库；数据库已有其他数据时接口数量会相应增加。

## 本地检查

```powershell
node --check insert_bank_and_series.js
node --check insert_question.js
npm.cmd test
```

测试不连接数据库，验证字段、BSON Int64 / Date、固定 ID、upsert 更新内容、配置读取和关联缺失时停止写入。真实连接与重复写入需在填写测试配置后运行验证。
