const assert = require('node:assert/strict')
const { readFileSync } = require('node:fs')
const Module = require('node:module')
const path = require('node:path')
const { test } = require('node:test')
const ts = require('typescript')

const modules = new Map()
function loadTs(relativePath) {
  const filename = path.resolve(__dirname, '../src', relativePath)
  if (modules.has(filename)) return modules.get(filename).exports
  const loaded = new Module(filename, module)
  loaded.paths = Module._nodeModulePaths(path.dirname(filename))
  loaded.require = (id) => id.startsWith('@/') ? loadTs(`${id.slice(2)}.ts`) : module.require(id)
  modules.set(filename, loaded)
  loaded._compile(ts.transpileModule(readFileSync(filename, 'utf8'), {
    compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 },
  }).outputText, `${filename}.cjs`)
  return loaded.exports
}

test('comment services send query/body, unwrap results and preserve errors', async () => {
  const service = loadTs('services/comment.ts')
  const { apiClient } = loadTs('lib/axios.ts')
  const originalAdapter = apiClient.defaults.adapter
  const originalStorage = globalThis.localStorage
  globalThis.localStorage = { getItem: () => 'token' }
  const listParams = { target_type: 1, target_id: 'q1', parent_id: 'c1', sort_by: 'thumbs_up', sort_order: 'asc', page: 2, page_size: 20, sub_comment_page: 2, sub_comment_size: 5 }
  const cases = [
    ['getCommentList', 'get', '/api/v1/open/list_comments', listParams, { total: 1, list: [{ comment_id: 'c1', sub_comments: [] }] }],
    ['postCommentAdd', 'post', '/api/v1/comment/add', { target_type: 1, target_id: 'q1', parent_id: 'c1', reply_to: 'u1', content: '回复' }, { comment_id: 'c2', comment: { content: '回复' } }],
    ['postCommentDelete', 'post', '/api/v1/comment/delete', { comment_id: 'c2' }, undefined],
    ['postCommentUpdate', 'post', '/api/v1/comment/update', { comment_id: 'c1', content: '修改' }, { comment_id: 'c1' }],
    ['thumbsUpComment', 'post', '/api/v1/interaction/like', { comment_id: 'c1' }, { liked: true, count: 3 }],
  ]
  try {
    for (const [name, method, url, params, result] of cases) {
      apiClient.defaults.adapter = async (config) => {
        assert.equal(config.method, method)
        assert.equal(config.url, url)
        assert.equal(config.headers.get('Authorization'), 'Bearer token')
        assert.deepEqual(method === 'get' ? config.params : JSON.parse(config.data), name === 'thumbsUpComment' ? { target_type: 3, target_id: params.comment_id } : params)
        return { data: { code: 0, msg: 'success', ...(result === undefined ? {} : { data: result }) }, status: 200, headers: {}, config }
      }
      assert.deepEqual(await service[name](params), result)
      for (const code of [400, 401, 403, 404, 500, '0']) {
        apiClient.defaults.adapter = async (config) => ({ data: { code, msg: '操作失败', data: null }, status: 200, headers: {}, config })
        await assert.rejects(service[name](params), { message: '操作失败' })
      }
      const failure = new Error('Network Error')
      apiClient.defaults.adapter = async () => { throw failure }
      await assert.rejects(service[name](params), (error) => error === failure)
    }
  } finally {
    apiClient.defaults.adapter = originalAdapter
    if (originalStorage === undefined) delete globalThis.localStorage
    else globalThis.localStorage = originalStorage
  }
})
