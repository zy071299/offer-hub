const assert = require('node:assert/strict')
const { existsSync, readFileSync } = require('node:fs')
const Module = require('node:module')
const path = require('node:path')
const { test } = require('node:test')
const ts = require('typescript')
const React = require('react')
const { act, create } = require('react-test-renderer')
const { QueryClient, QueryClientProvider } = require('@tanstack/react-query')

test('comment UI paginates, gates guests, replies in two levels and handles owner actions', async () => {
  const makeComment = (id, parent = '') => ({ comment_id: id, user_id: id === 'c0' ? 'me' : 'other', user_name: id, user_avatar: '', content: `内容 ${id}`, parent_id: parent, reply_to: '', reply_to_name: '', status: 2, thumbs_up: 2, sub_comment_total: parent ? 0 : id === 'c0' ? 6 : 0, user_liked: false, sub_comments: [], create_time: '2026-10-21 10:00:00', update_time: '' })
  let comments = Array.from({ length: 21 }, (_, i) => makeComment(`c${i}`))
  const replies = Array.from({ length: 6 }, (_, i) => makeComment(`r${i}`, 'c0'))
  const calls = []
  let finishLike
  let rejectAdd = false
  const auth = { isLoggedIn: false, userInfo: null, setShowLoginDialog: (show) => calls.push(['login', show]) }
  const services = {
    getCommentList: async (params) => {
      calls.push(['list', params])
      const source = params.parent_id ? replies : comments
      const size = params.page_size ?? 20
      return { total: source.length, list: source.slice((params.page - 1) * size, params.page * size) }
    },
    postCommentAdd: async (params) => { calls.push(['add', params]); if (rejectAdd) throw new Error('发表失败'); return { comment_id: 'new', comment: makeComment('new') } },
    postCommentUpdate: async (params) => { calls.push(['update', params]); comments = comments.map((c) => c.comment_id === params.comment_id ? { ...c, content: params.content } : c); return { comment_id: params.comment_id } },
    postCommentDelete: async (params) => { calls.push(['delete', params]); comments = comments.filter((c) => c.comment_id !== params.comment_id) },
    thumbsUpComment: (params) => { calls.push(['like', params]); return new Promise((resolve) => { finishLike = () => resolve({ liked: true, count: 3 }) }) },
  }
  const modules = new Map()
  function load(relativePath) {
    let filename = path.resolve(__dirname, '../src', relativePath)
    if (!path.extname(filename)) filename += existsSync(`${filename}.ts`) ? '.ts' : '.tsx'
    if (modules.has(filename)) return modules.get(filename).exports
    const loaded = new Module(filename, module)
    loaded.paths = Module._nodeModulePaths(path.dirname(filename))
    loaded.require = (id) => id === '@/services/comment' ? services : id === '@/hooks/useLogin' ? { useLogin: () => auth } : id.startsWith('@/') ? load(id.slice(2)) : module.require(id)
    modules.set(filename, loaded)
    loaded._compile(ts.transpileModule(readFileSync(filename, 'utf8'), { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022, jsx: ts.JsxEmit.ReactJSX } }).outputText, `${filename}.cjs`)
    return loaded.exports
  }
  const { default: Comments } = load('components/comment/Comments')
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  const render = () => React.createElement(QueryClientProvider, { client }, React.createElement(Comments, { targetType: 1, targetId: 'q1' }))
  let tree
  const flush = async (fn = () => {}) => act(async () => { await fn(); await new Promise((resolve) => setTimeout(resolve, 25)) })
  const text = (node) => node.children.map((child) => typeof child === 'string' ? child : text(child)).join('')
  const button = (label) => tree.root.findAllByType('button').find((node) => text(node) === label)
  const item = (id) => tree.root.findByProps({ 'data-comment-id': id })
  const itemButton = (id, label) => item(id).findAllByType('button').find((node) => text(node) === label)
  try {
    await flush(() => { tree = create(render()) })
    assert.equal(calls.filter(([kind]) => kind === 'list').length, 1)
    assert.equal(tree.root.findAllByType('button').some((node) => text(node) === '删除'), false)
    act(() => tree.root.findByProps({ 'aria-label': '发表评论' }).props.onFocus())
    assert.deepEqual(calls.at(-1), ['login', true])
    await flush(() => button('加载更多').props.onClick())
    assert.ok(item('c20'))
    auth.isLoggedIn = true
    auth.userInfo = { user_id: 'me' }
    await flush(() => tree.update(render()))
    assert.ok(itemButton('c0', '删除'))
    await flush(() => itemButton('c0', '查看 6 条回复').props.onClick())
    assert.equal(calls.findLast(([kind]) => kind === 'list')[1].page_size, 5)
    await flush(() => button('查看更多回复').props.onClick())
    assert.ok(item('r5'))
    act(() => itemButton('r0', '回复').props.onClick())
    const replyForm = () => tree.root.findByProps({ 'aria-label': '回复评论' })
    act(() => replyForm().findByType('textarea').props.onChange({ target: { value: '回复内容' } }))
    await flush(() => replyForm().props.onSubmit({ preventDefault() {} }))
    assert.deepEqual(calls.findLast(([kind]) => kind === 'add')[1], { target_type: 1, target_id: 'q1', parent_id: 'c0', reply_to: 'other', content: '回复内容' })
    act(() => itemButton('c0', '编辑').props.onClick())
    const editForm = () => tree.root.findByProps({ 'aria-label': '编辑评论' })
    act(() => editForm().findByType('textarea').props.onChange({ target: { value: '已修改' } }))
    await flush(() => editForm().props.onSubmit({ preventDefault() {} }))
    assert.equal(calls.findLast(([kind]) => kind === 'update')[1].content, '已修改')
    await flush(() => item('c0').findAllByType('button').find((node) => node.props['aria-label'] === '点赞评论').props.onClick())
    assert.equal(item('c0').findAllByType('button').find((node) => node.props['aria-label'] === '点赞评论').props['aria-pressed'], false)
    await flush(() => finishLike())
    assert.equal(item('c0').findAllByType('button').find((node) => node.props['aria-label'] === '点赞评论').props['aria-pressed'], true)
    rejectAdd = true
    const input = () => tree.root.findByProps({ 'aria-label': '发表新评论' })
    act(() => input().findByType('textarea').props.onChange({ target: { value: '保留草稿' } }))
    await flush(() => input().props.onSubmit({ preventDefault() {} }))
    assert.equal(input().findByType('textarea').props.value, '保留草稿')
    assert.ok(tree.root.findAllByProps({ role: 'alert' }).some((node) => text(node).includes('发表失败')))
    await flush(() => itemButton('c0', '删除').props.onClick())
    assert.equal(tree.root.findAllByProps({ 'data-comment-id': 'c0' }).length, 0)
  } finally {
    if (tree) act(() => tree.unmount())
    client.clear()
  }
})
