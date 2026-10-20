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
  loaded.require = (id) => id.startsWith('@/')
    ? loadTs(`${id.slice(2)}.ts`)
    : module.require(id)
  modules.set(filename, loaded)
  const compiled = ts.transpileModule(readFileSync(filename, 'utf8'), {
    compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 },
  }).outputText
  loaded._compile(compiled, `${filename}.cjs`)
  return loaded.exports
}

test('question services use documented endpoints, queries and business error handling', async () => {
  const services = loadTs('services/question.ts')
  const { apiClient } = loadTs('lib/axios.ts')
  const originalStorage = globalThis.localStorage
  const originalAdapter = apiClient.defaults.adapter
  globalThis.localStorage = { getItem: () => null }
  try {
    const filter = {
      bank_id: 'b001', keyword: 'Redis 持久化', difficulty: 2,
      tags: ['Go', '数据库'], job_name: '后端开发', user_tag: 1,
      sort_by: 'view_count', sort_order: 'desc', page: 2, page_size: 10,
    }
    const cases = [
      ['getAllQuestionList', '/api/v1/question/all/list', { job_name: '后端开发' }, []],
      ['getQuestionList', '/api/v1/question/list', filter, { total: 0, list: [] }],
      ['getQuestionMetaList', '/api/v1/question/meta/list', filter, { total: 0, list: [] }],
      ['getQuestionDetail', '/api/v1/question/detail', { question_id: 'q001' }, { question_id: 'q001' }],
      ['getHotQuestionList', '/api/v1/question/hot/list', { limit: 5, job_name: '后端开发' }, { list: [] }],
    ]
    for (const [name, endpoint, params, payload] of cases) {
      apiClient.defaults.adapter = async (config) => {
        assert.equal(config.method, 'get')
        assert.equal(config.url, endpoint)
        const query = new URL(apiClient.getUri(config), 'http://localhost').searchParams
        for (const [key, value] of Object.entries(params)) {
          assert.deepEqual(query.getAll(key), Array.isArray(value) ? value : [String(value)])
        }
        assert.equal(query.has('tags[]'), false)
        return { data: { code: 0, msg: 'success', data: payload }, status: 200, headers: {}, config }
      }
      assert.deepEqual(await services[name](params), payload)

      for (const code of [400, 401, 500, '0']) {
        apiClient.defaults.adapter = async (config) => ({
          data: { code, msg: '查询失败', data: null }, status: 200, headers: {}, config,
        })
        await assert.rejects(services[name](params), { message: '查询失败' })
      }
      apiClient.defaults.adapter = async (config) => ({
        data: { code: 404, msg: '题目不存在', data: null }, status: 200, headers: {}, config,
      })
      await assert.rejects(services[name](params), (error) => error.code === 404 && error.message === '题目不存在')
      const networkError = new Error('Network Error')
      apiClient.defaults.adapter = async () => { throw networkError }
      await assert.rejects(services[name](params), (error) => error === networkError)
    }

    for (const name of ['getAllQuestionList', 'getQuestionList', 'getQuestionMetaList', 'getHotQuestionList']) {
      apiClient.defaults.adapter = async (config) => {
        assert.equal(new URL(apiClient.getUri(config), 'http://localhost').search, '')
        return { data: { code: 0, msg: 'success', data: { list: [] } }, status: 200, headers: {}, config }
      }
      await services[name]()
    }
  } finally {
    apiClient.defaults.adapter = originalAdapter
    if (originalStorage === undefined) delete globalThis.localStorage
    else globalThis.localStorage = originalStorage
  }
})
