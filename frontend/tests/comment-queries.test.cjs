const assert = require('node:assert/strict')
const { readFileSync } = require('node:fs')
const Module = require('node:module')
const path = require('node:path')
const { test } = require('node:test')
const ts = require('typescript')
const React = require('react')
const { act, create } = require('react-test-renderer')
const { QueryClient, QueryClientProvider } = require('@tanstack/react-query')

test('comment hooks disable invalid queries and refresh lists only after successful writes', async () => {
  const calls = []
  let failure
  const result = { total: 1, list: [] }
  const services = Object.fromEntries(['getCommentList', 'postCommentAdd', 'postCommentUpdate', 'postCommentDelete'].map((name) => [name, async (params) => {
    calls.push({ name, params })
    if (failure) throw failure
    return name === 'getCommentList' ? result : name === 'postCommentDelete' ? undefined : { comment_id: 'c1' }
  }]))
  const filename = path.resolve(__dirname, '../src/hooks/useCommentQueries.ts')
  const loaded = new Module(filename, module)
  loaded.paths = Module._nodeModulePaths(path.dirname(filename))
  loaded.require = (id) => id === '@/services/comment' ? services : module.require(id)
  loaded._compile(ts.transpileModule(readFileSync(filename, 'utf8'), {
    compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 },
  }).outputText, `${filename}.cjs`)
  const hooks = loaded.exports
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity }, mutations: { retry: false } } })
  let params = { target_type: 1, target_id: ' ' }
  let enabled = true
  let state
  function Consumer() {
    state = { list: hooks.useComments(params, enabled), add: hooks.useAddComment(), update: hooks.useUpdateComment(), delete: hooks.useDeleteComment() }
    return null
  }
  const render = () => React.createElement(QueryClientProvider, { client }, React.createElement(Consumer))
  let tree
  try {
    await act(async () => { tree = create(render()) })
    assert.equal(calls.length, 0)
    params = { target_type: 1, target_id: 'q1', page: 1, parent_id: 'c0' }
    enabled = false
    await act(async () => tree.update(render()))
    assert.equal(calls.length, 0)
    enabled = true
    await act(async () => { tree.update(render()); await new Promise((resolve) => setTimeout(resolve, 20)) })
    assert.deepEqual(state.list.data, result)
    assert.deepEqual(calls[0].params, params)
    assert.deepEqual(hooks.queryKeys.list(params), ['comments', 'list', params])
    client.setQueryData(['questions', 'list'], { total: 1 })
    for (const [hook, name, variables] of [
      ['add', 'postCommentAdd', { target_type: 1, target_id: 'q1', content: 'hello' }],
      ['update', 'postCommentUpdate', { comment_id: 'c1', content: 'changed' }],
      ['delete', 'postCommentDelete', { comment_id: 'c1' }],
    ]) {
      const before = calls.filter((call) => call.name === 'getCommentList').length
      await act(async () => { await state[hook].mutateAsync(variables) })
      assert.deepEqual(calls.findLast((call) => call.name === name).params, variables)
      assert.equal(calls.filter((call) => call.name === 'getCommentList').length, before + 1)
      failure = new Error('denied')
      await act(async () => { await assert.rejects(state[hook].mutateAsync(variables), (error) => error === failure) })
      assert.equal(calls.filter((call) => call.name === 'getCommentList').length, before + 1)
      failure = undefined
    }
    assert.equal(client.getQueryState(['questions', 'list']).isInvalidated, false)
  } finally {
    if (tree) act(() => tree.unmount())
    client.clear()
  }
})
