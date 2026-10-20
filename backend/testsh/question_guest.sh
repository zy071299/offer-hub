#!/usr/bin/env bash
# 游客题库/题目回归。依赖：Bash 4+、curl、python3、mktemp；不依赖 jq。
# 用法：bash backend/testsh/question_guest.sh
#       BASE_URL=http://127.0.0.1:8180 bash backend/testsh/question_guest.sh
#       PYTHON_BIN=/path/to/python3 bash backend/testsh/question_guest.sh
# 先启动后端并准备非空种子数据；BASE_URL 不含 /api/v1/question 前缀。
# 只测 router.go 目前存在的五个 GET 路径；不发 token，不写数据库。
# 当前游客列表截断 150 字、user_tag=0/user_liked=false；详情暂返回完整正文。
# 登录扩展：复制为 question_authenticated.sh，在其 curl 请求中显式加入
# Authorization: Bearer "$TOKEN"，并改为断言完整内容、实际点赞/掌握状态。
# 标记/点赞等 POST 必须等 router.go 真正注册后再加；本脚本不猜测路径。
# 固定测试数据运行期间请勿并发修改题目，以保证 total/分页/排序比较稳定。
set -Eeuo pipefail
export PYTHONUTF8=1

BASE_URL="${BASE_URL:-http://127.0.0.1:8180}"
BASE_URL="${BASE_URL%/}"
PYTHON_BIN="${PYTHON_BIN:-python3}"
passed=0
failed=0
work=''

pass() { passed=$((passed + 1)); printf '[PASS] %s\n' "$1"; }
fail() { failed=$((failed + 1)); printf '[FAIL] %s\n' "$1" >&2; }
finish() {
  local status=$?
  trap - EXIT
  if (( status != 0 && failed == 0 )); then fail "脚本异常终止（exit=$status）"; fi
  printf '\n通过: %d  失败: %d\n' "$passed" "$failed"
  if [[ -n "$work" ]]; then
    rm -f -- "$work"/*.json "$work/check.py"
    rmdir -- "$work"
  fi
  if (( failed > 0 || status != 0 )); then exit 1; fi
}
trap finish EXIT

for dependency in curl mktemp date rm rmdir "$PYTHON_BIN"; do
  if ! command -v "$dependency" >/dev/null 2>&1; then
    fail "缺少依赖 $dependency（Python 可用 PYTHON_BIN 覆盖）"; exit 1
  fi
done
if ! "$PYTHON_BIN" -c 'import sys; assert sys.version_info >= (3, 8)' >/dev/null 2>&1; then
  fail '需要可运行的 Python 3.8+'; exit 1
fi
case "$BASE_URL" in http://*|https://*) ;; *) fail 'BASE_URL 必须以 http:// 或 https:// 开头'; exit 1 ;; esac
work=$(mktemp -d)
cat > "$work/check.py" <<'PY'
import json
import sys
from urllib.parse import quote

# Windows Python 也输出 LF，避免 Bash 提取的 ID 带有 CR。
sys.stdout.reconfigure(newline='\n')

FIELDS = {'question_id', 'bank_list', 'title', 'content', 'difficulty', 'tags', 'status', 'vip',
          'hot_degree', 'view_count', 'thumbs_up_count', 'dislike_count', 'order', 'user_tag',
          'user_liked', 'create_time', 'update_time'}
HOT = {'question_id', 'bank_list', 'title', 'view_count'}

def need(condition, message):
    if not condition:
        raise ValueError(message)

def load(path):
    with open(path, encoding='utf-8-sig') as stream:
        return json.load(stream)

def integer(value):
    return type(value) is int

def strings(value):
    return isinstance(value, list) and all(isinstance(x, str) for x in value)

def question(q, guest_list=False):
    need(isinstance(q, dict) and set(q) == FIELDS, 'OneQuestion 字段不符合约定')
    for key in ('question_id', 'title', 'content', 'create_time', 'update_time'):
        need(isinstance(q[key], str), key + ' 必须为字符串')
    need(bool(q['question_id']), 'question_id 为空')
    need(strings(q['bank_list']) and strings(q['tags']), 'bank_list/tags 必须是字符串数组')
    for key in ('difficulty', 'status', 'hot_degree', 'view_count', 'thumbs_up_count', 'dislike_count', 'order', 'user_tag'):
        need(integer(q[key]), key + ' 必须为整数')
    need(type(q['vip']) is bool and type(q['user_liked']) is bool, '布尔字段类型错误')
    need(q['user_tag'] == 0 and q['user_liked'] is False, '游客用户状态应为零值')
    if guest_list:
        need(q['status'] == 1, '列表包含非正常题目')
        need(len(q['content']) <= 150, '游客列表内容超过 150 个字符')

def listing(body):
    data = body['data']
    need(isinstance(data, dict) and integer(data.get('total')) and data['total'] >= 0, 'total 类型或值错误')
    need(isinstance(data.get('list'), list), 'list 必须为数组，不能为 null')
    need(data['total'] >= len(data['list']), 'total 小于当前页条数')
    for item in data['list']:
        question(item, True)
    ids = [item['question_id'] for item in data['list']]
    need(len(ids) == len(set(ids)), '当前页有重复题目')
    return data

def main():
    action, path, *args = sys.argv[1:]
    if action == 'encode':
        # 从管道读取原始 UTF-8 字节，不把中文放进 Windows curl 的 argv。
        raw = sys.stdin.buffer.read().decode('utf-8')
        name, value = raw.split('=', 1)
        print(quote(name, safe='') + '=' + quote(value, safe=''))
        return
    body = load(path)
    if action == 'extract':
        key = args[0]
        if key in ('qid', 'keyword', 'difficulty', 'tags'):
            item = body['data']['list'][0]
            values = {'qid': item['question_id'], 'keyword': item['title'],
                      'difficulty': str(item['difficulty']), 'tags': item['tags'][0] if item['tags'] else ''}
            value = values[key]
            need(value != '', '前置列表缺少 ' + key)
            print(value)
        elif key in ('bank', 'job'):
            pairs = [(g['job_name'], b) for g in body['data'] for s in g['series_list'] for b in s['bank_list'] if b['count'] > 0]
            need(bool(pairs), '没有含正常题目的题库')
            print(pairs[0][1]['bank_id'] if key == 'bank' else pairs[0][0])
        elif key == 'hot_ids':
            for item in body['data']['list']:
                print(item['question_id'])
        return
    expected, mode, *args = args
    expected = int(expected)
    need(isinstance(body, dict), '响应必须为 JSON 对象')
    need(integer(body.get('code')) and body['code'] == expected, '业务 code 不符：' + str(body.get('code')))
    need(isinstance(body.get('msg'), str) and bool(body['msg']), 'msg 必须为非空字符串')
    need('data' in body, '缺少 data 字段')
    if expected != 0:
        need(body['data'] is None, '错误响应 data 应为 null')
        if expected == 404:
            need(body['msg'] == '题目不存在', '404 msg 不符')
        elif expected == 400:
            need(body['msg'] in ('参数错误', 'question_id 必填', 'limit 必须为正整数'), '400 msg 不符')
        return
    need(body['msg'] == 'success', '成功 msg 应为 success')
    if mode == 'catalog':
        need(isinstance(body['data'], list) and bool(body['data']), '题库数据为空；请先准备种子数据')
        jobs = set()
        for group in body['data']:
            need(isinstance(group.get('job_name'), str) and group['job_name'] not in jobs, '职位分组异常')
            jobs.add(group['job_name'])
            series = group['series_list']
            need(isinstance(series, list), 'series_list 必须为数组')
            need(all(integer(s['order']) for s in series), '系列 order 类型错误')
            need([s['order'] for s in series] == sorted(s['order'] for s in series), '系列 order 未升序')
            for item in series:
                need(set(item) == {'series_id', 'series_name', 'order', 'bank_list'}, '系列字段错误')
                banks = item['bank_list']
                need(isinstance(banks, list), 'bank_list 必须为数组')
                need(all(integer(b['order']) for b in banks), '题库 order 类型错误')
                need([b['order'] for b in banks] == sorted(b['order'] for b in banks), '题库 order 未升序')
                for bank in banks:
                    need(set(bank) == {'bank_id', 'bank_name', 'bank_logo', 'desc', 'count', 'order'}, '题库字段错误')
                    need(integer(bank['count']) and bank['count'] >= 0, '题库数量非法')
    elif mode == 'catalog_job':
        need(isinstance(body['data'], list) and bool(body['data']), '职位过滤后没有题库')
        need(all(g['job_name'] == args[0] for g in body['data']), 'job_name 过滤未生效')
    elif mode in ('list', 'bank', 'difficulty', 'tags', 'keyword', 'same', 'page2', 'sort'):
        data = listing(body)
        if mode == 'list':
            need(data['total'] > 0 and bool(data['list']), '需要非空正常题目数据')
            need(len(data['list']) <= int(args[0]), '分页大小超出限制')
            times = [q['create_time'] for q in data['list']]
            need(times == sorted(times, reverse=True), '默认排序应为 create_time desc')
        elif mode == 'bank':
            need(bool(data['list']) and all(args[0] in q['bank_list'] for q in data['list']), 'bank_id 过滤未生效')
        elif mode == 'difficulty':
            need(bool(data['list']) and all(q['difficulty'] == int(args[0]) for q in data['list']), 'difficulty 过滤未生效')
        elif mode == 'tags':
            need(bool(data['list']) and all(args[0] in q['tags'] for q in data['list']), 'tags 过滤未生效')
        elif mode == 'keyword':
            need(any(q['question_id'] == args[0] for q in data['list']), '关键词搜索未找回原题目')
        elif mode == 'same':
            old = load(args[0])['data']
            need(data == old, '游客 user_tag 过滤未被忽略')
        elif mode == 'page2':
            first = load(args[0])['data']
            need(data['total'] == first['total'], '分页 total 不一致')
            need(len(data['list']) == min(2, max(0, data['total'] - 2)), '第二页条数不符')
            need(not ({q['question_id'] for q in first['list']} & {q['question_id'] for q in data['list']}), '两页有重复题目')
        elif mode == 'sort':
            values = [q[args[0]] for q in data['list']]
            need(values == sorted(values, reverse=args[1] == 'desc'), '排序未生效')
    elif mode in ('meta', 'meta_shape'):
        data = body['data']
        need(isinstance(data, dict) and integer(data.get('total')) and data['total'] >= 0 and isinstance(data.get('list'), list), 'meta total 或数组类型不符')
        need(all(set(q) == {'question_id', 'title'} and isinstance(q['question_id'], str) and isinstance(q['title'], str) for q in data['list']), 'meta 返回了多余字段或类型不符')
        if mode == 'meta':
            original = load(args[0])['data']
            need(data['total'] == original['total'], 'list/meta total 不符')
            need(data['list'] == [{'question_id': q['question_id'], 'title': q['title']} for q in original['list']], 'list/meta 内容或顺序不符')
    elif mode == 'detail':
        question(body['data'])
        need(body['data']['question_id'] == args[0], '详情 question_id 不符')
        if len(args) > 1:
            old = next(q for q in load(args[1])['data']['list'] if q['question_id'] == args[0])
            full = body['data']
            need(old['content'] == full['content'][:150], '列表截断与详情完整内容不一致')
            need(old['title'] == full['title'], '列表和详情标题不一致')
    elif mode == 'hot':
        items = body['data']['list']
        need(isinstance(items, list) and 0 < len(items) <= int(args[0]), '热门条数异常或无数据')
        need(len({q['question_id'] for q in items}) == len(items), '热门题目重复')
        for item in items:
            need(set(item) == HOT, '热门项必须且只能有四个字段')
            need(isinstance(item['question_id'], str) and isinstance(item['title'], str) and strings(item['bank_list']) and integer(item['view_count']), '热门字段类型错误')
    elif mode == 'hot_heat':
        items = body['data']['list']
        details = [load(p)['data'] for p in args]
        need(len(items) == len(details), '热门详情检查缺失')
        heat = []
        for item, detail in zip(items, details):
            need(item['question_id'] == detail['question_id'] and detail['status'] == 1, '热门题目状态/ID 异常')
            need(item == {k: detail[k] for k in HOT}, '热门字段与详情不一致')
            heat.append(detail['hot_degree'])
        need(heat == sorted(heat, reverse=True), '热门未按 hot_degree 降序（不能按 view_count 排序）')
    else:
        raise ValueError('未知校验模式 ' + mode)

try:
    main()
except (ValueError, KeyError, TypeError, IndexError, OSError) as error:
    print('断言失败：' + str(error), file=sys.stderr)
    sys.exit(1)
PY

# 每个检查只统计一次；HTTP/连接/JSON/业务断言失败均记录，独立检查继续。
check() {
  local label=$1 key=$2 route=$3 code=$4 mode=$5
  shift 5
  local -a validation=()
  while (( $# > 0 )) && [[ "$1" != '--' ]]; do validation+=("$1"); shift; done
  if (( $# > 0 )); then shift; fi
  # Git Bash 调用某些 Windows curl 时，中文 argv 会被转成系统代码页。
  # 先用 Python 编成纯 ASCII query，再用 --data（非 --data-urlencode）发送，避免二次编码。
  local -a curl_query=()
  local encoded
  while (( $# > 0 )); do
    if [[ "$1" != '--data-urlencode' ]] || (( $# < 2 )); then
      fail "$label：内部 query 参数格式错误"; return 1
    fi
    if ! encoded=$(printf '%s' "$2" | "$PYTHON_BIN" "$work/check.py" encode -); then
      fail "$label：UTF-8 query 编码失败"; return 1
    fi
    curl_query+=(--data "$encoded")
    shift 2
  done
  local http
  if ! http=$(curl -q --silent --show-error --connect-timeout 3 --max-time 15 \
      --get --header 'Accept: application/json' --header 'Authorization:' \
      --output "$work/$key.json" --write-out '%{http_code}' \
      "$BASE_URL/api/v1/question$route" "${curl_query[@]}"); then
    fail "$label：curl 请求失败"; return 1
  fi
  if [[ "$http" != 200 ]]; then fail "$label：HTTP=$http，预期 200"; return 1; fi
  if ! "$PYTHON_BIN" "$work/check.py" validate "$work/$key.json" "$code" "$mode" "${validation[@]}"; then
    fail "$label：响应断言失败"; return 1
  fi
  pass "$label"
}
extract() { "$PYTHON_BIN" "$work/check.py" extract "$work/$1.json" "$2"; }

printf '游客回归：%s（不发送 token，只读 GET）\n\n' "$BASE_URL"
catalog_ready=0
list_ready=0
page_ready=0
hot_ready=0
if check '题库全量列表、字段及 order 升序' catalog /all/list 0 catalog; then catalog_ready=1; fi
if check '题目列表默认分页、正常状态与游客字段' list /list 0 list 20; then list_ready=1; fi
if check '题目列表第一页 page_size=2' page1 /list 0 list 2 -- --data-urlencode 'page_size=2'; then page_ready=1; fi
if (( page_ready )); then
  check '题目列表第二页无重复且 total 一致' page2 /list 0 page2 "$work/page1.json" -- --data-urlencode 'page=2' --data-urlencode 'page_size=2' || :
else fail '第二页检查：前置第一页失败'; fi
if (( list_ready )); then
  check '元数据与列表过滤、排序及 total 一致' meta /meta/list 0 meta "$work/list.json" || :
  check '游客忽略 user_tag 过滤' usertag /list 0 same "$work/list.json" -- --data-urlencode 'user_tag=3' || :
  if qid=$(extract list qid); then
    check '由列表 ID 查询详情及内容截断一致性' detail /detail 0 detail "$qid" "$work/list.json" -- --data-urlencode "question_id=$qid" || :
    if keyword=$(extract list keyword); then
      check '关键词搜索找回原题目' keyword /list 0 keyword "$qid" -- --data-urlencode "keyword=$keyword" || :
    else fail '关键词检查：列表标题不可用'; fi
  else fail '详情检查：列表没有可用 question_id'; fi
  if difficulty=$(extract list difficulty); then
    check 'difficulty 过滤' difficulty /list 0 difficulty "$difficulty" -- --data-urlencode "difficulty=$difficulty" || :
  else fail 'difficulty 检查：前置字段不可用'; fi
  if tag=$(extract list tags); then
    check 'tags 过滤' tags /list 0 tags "$tag" -- --data-urlencode "tags=$tag" || :
  else fail 'tags 检查：前置列表没有标签'; fi
else
  fail '元数据对比、详情及派生过滤检查：前置列表失败'
  check '元数据独立请求' meta_alone /meta/list 0 meta_shape || :
fi
if (( catalog_ready )); then
  if bank=$(extract catalog bank); then
    check 'bank_id 过滤' bank /list 0 bank "$bank" -- --data-urlencode "bank_id=$bank" || :
  else fail 'bank_id 检查：无非空题库'; fi
  if job=$(extract catalog job); then
    check '题库 job_name 过滤' job /all/list 0 catalog_job "$job" -- --data-urlencode "job_name=$job" || :
  else fail 'job_name 检查：无可用职位'; fi
fi
check 'view_count 降序' sort_desc /list 0 sort view_count desc -- --data-urlencode 'sort_by=view_count' --data-urlencode 'sort_order=desc' || :
check 'view_count 升序' sort_asc /list 0 sort view_count asc -- --data-urlencode 'sort_by=view_count' --data-urlencode 'sort_order=asc' || :
check '热门默认 limit=10、仅四字段' hot_default /hot/list 0 hot 10 || :
if check '热门显式 limit=3' hot /hot/list 0 hot 3 -- --data-urlencode 'limit=3'; then hot_ready=1; fi
if (( hot_ready )); then
  hot_ids=$(extract hot hot_ids)
  hot_index=0
  heat_ready=1
  hot_files=()
  while IFS= read -r hot_id; do
    hot_index=$((hot_index + 1))
    key="hot_detail_$hot_index"
    if check "热门第 $hot_index 项详情" "$key" /detail 0 detail "$hot_id" -- --data-urlencode "question_id=$hot_id"; then
      hot_files+=("$work/$key.json")
    else heat_ready=0; fi
  done <<< "$hot_ids"
  if (( heat_ready )); then
    if "$PYTHON_BIN" "$work/check.py" validate "$work/hot.json" 0 hot_heat "${hot_files[@]}"; then pass '热门按 hot_degree 降序且仅正常题目'; else fail '热门热度与状态断言失败'; fi
  else fail '热门热度检查：详情前置失败'; fi
fi
check '详情 question_id 必填' detail_missing /detail 400 error || :
missing_id="__guest_regression_missing_$(date +%s)_$RANDOM"
check '不存在题目返回业务 404' detail_absent /detail 404 error -- --data-urlencode "question_id=$missing_id" || :
check '列表非法 page 返回业务 400' invalid_page /list 400 error -- --data-urlencode 'page=0' || :
check '列表非法排序返回业务 400' invalid_sort /list 400 error -- --data-urlencode 'sort_by=content' || :
check '元数据非法 page 返回业务 400' invalid_meta /meta/list 400 error -- --data-urlencode 'page=0' || :
check '热门非法 limit 返回业务 400' invalid_limit /hot/list 400 error -- --data-urlencode 'limit=0' || :
if (( failed > 0 )); then exit 1; fi
