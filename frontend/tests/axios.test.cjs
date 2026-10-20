const assert = require('node:assert/strict')
const { readFileSync } = require('node:fs')
const Module = require('node:module')
const path = require('node:path')
const { test } = require('node:test')
const ts = require('typescript')
const { AxiosError } = require('axios')

// Load the TypeScript module with the existing compiler; no extra test runtime needed.
const filename = path.resolve(__dirname, '../src/lib/axios.ts')
const compiled = ts.transpileModule(readFileSync(filename, 'utf8'), {
  compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 },
}).outputText
const loaded = new Module(filename, module)
loaded.filename = filename
loaded.paths = Module._nodeModulePaths(path.dirname(filename))
loaded._compile(compiled, `${filename}.cjs`)
const { apiClient } = loaded.exports

test('uses same-origin proxy requests and a 10-second timeout', () => {
  assert.equal(apiClient.defaults.baseURL, '')
  assert.equal(apiClient.defaults.timeout, 10_000)
})

test('reads the current token for each request and omits auth when absent', async () => {
  let token = 'first-token'
  const original = globalThis.localStorage
  globalThis.localStorage = { getItem: (key) => key === 'token' ? token : null }
  try {
    const adapter = async (config) => ({
      data: config.headers.get('Authorization'), status: 200,
      statusText: 'OK', headers: {}, config,
    })
    assert.equal((await apiClient.get('/api/example', { adapter })).data, 'Bearer first-token')
    token = 'second-token'
    assert.equal((await apiClient.get('/api/example', { adapter })).data, 'Bearer second-token')
    token = null
    assert.equal((await apiClient.get('/api/example', { adapter })).data, undefined)
  } finally {
    if (original === undefined) delete globalThis.localStorage
    else globalThis.localStorage = original
  }
})

test('clears only token on 401 and preserves other failures and error identity', async () => {
  const original = globalThis.localStorage
  const storage = new Map([['token', 'saved-token'], ['preference', 'dark']])
  globalThis.localStorage = {
    getItem: (key) => storage.get(key) ?? null,
    removeItem: (key) => storage.delete(key),
  }
  try {
    for (const status of [500, undefined, 401]) {
      let failure
      const adapter = async (config) => {
        const response = status === undefined ? undefined : {
          data: {}, status, statusText: 'Error', headers: {}, config,
        }
        failure = new AxiosError('Request failed', undefined, config, undefined, response)
        throw failure
      }
      await assert.rejects(apiClient.get('/api/example', { adapter }), (error) => error === failure)
      assert.equal(storage.has('token'), status !== 401)
      assert.equal(storage.get('preference'), 'dark')
    }
  } finally {
    if (original === undefined) delete globalThis.localStorage
    else globalThis.localStorage = original
  }
})
