# 游客题库/题目 HTTP 回归

依赖 Bash 4+、curl、Python 3.8+（默认命令 `python3`）及常见 Unix 工具。Git Bash、WSL、Linux、macOS 可运行，不需要 jq。

先启动后端并准备非空测试数据，再从仓库根目录执行：

```bash
bash backend/testsh/question_guest.sh
BASE_URL=http://127.0.0.1:8180 bash backend/testsh/question_guest.sh
```

如 Python 命令或位置不同：

```bash
PYTHON_BIN=/path/to/python3 bash backend/testsh/question_guest.sh
```

Linux/macOS 也可 `chmod +x backend/testsh/question_guest.sh` 后直接执行。脚本默认地址为 `http://127.0.0.1:8180`。不发送 token，不修改数据，只检查 `src/router/router.go` 中当前注册的五个 GET 接口。

覆盖题库结构和排序、列表分页与部分过滤、list/meta 一致性、由列表动态取得 ID 后查询详情、游客字段和列表截断、热门四字段及通过详情验证热度排序，以及参数错误与不存在题目。当前详情按已实现行为返回完整正文。运行期间请勿修改题目，避免比较分页及总数时出现并发变化。

每个检查打印 `[PASS]` 或 `[FAIL]`；独立检查会在其他步骤失败后继续执行，依赖步骤缺失会明确报错。结尾汇总通过/失败数量，任一失败、缺少依赖或异常退出均返回非零状态。临时响应文件退出时自动清理。

登录后的回归应独立为 `question_authenticated.sh`：加入 token，并验证完整正文和实际用户状态。只有对应 POST 真正注册后才添加写操作测试；避免把登录或未实现路径混入当前游客回归。
