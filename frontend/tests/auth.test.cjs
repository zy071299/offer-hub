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
  const compiled = ts.transpileModule(readFileSync(filename, 'utf8'), {
    compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 },
  }).outputText
  loaded._compile(compiled, `${filename}.cjs`)
  return loaded.exports
}

test('maps every password login DTO field to context user info without mutating input', () => {
  const { mapPasswordAuthUserToUserInfo } = loadTs('services/auth.ts')
  const dto = Object.freeze({
    userId: 'u1', username: 'alice', nickName: 'Alice', avatar: '/avatar.png',
    sex: 2, vip: true, phone: '123', email: 'alice@example.com', userStatus: -1, userType: 2,
  })
  assert.deepEqual(mapPasswordAuthUserToUserInfo(dto), {
    user_id: 'u1', username: 'alice', nick_name: 'Alice', avatar: '/avatar.png',
    sex: 2, vip: true, phone: '123', email: 'alice@example.com', user_status: -1, user_type: 2,
    introduction: '', avatar_url: '/avatar.png', create_time: '', update_time: '',
  })
})

test('auth services send documented requests, unwrap data and reject nonzero codes', async () => {
  const auth = loadTs('services/auth.ts')
  const { apiClient } = loadTs('lib/axios.ts')
  const originalStorage = globalThis.localStorage
  const originalAdapter = apiClient.defaults.adapter
  globalThis.localStorage = { getItem: (key) => key === 'token' ? 'saved-token' : null }
  try {
    const cases = [
      ['postAuthRegister', ['alice', 'password'], 'post', '/auth/register', { username: 'alice', password: 'password' }, null],
      ['postAuthLogin', ['alice', 'password'], 'post', '/auth/login', { username: 'alice', password: 'password' }, { token: 'new-token', userInfo: { userId: 'u1' } }],
      ['postAuthLogout', [], 'post', '/auth/logout', undefined, { message: '登出成功' }],
      ['getUserInfo', [], 'get', '/api/v1/user_info/get_user_info', undefined, { user_id: 'u1' }],
    ]
    for (const [name, args, method, url, body, payload] of cases) {
      apiClient.defaults.adapter = async (config) => {
        assert.equal(config.method, method)
        assert.equal(config.url, url)
        assert.equal(config.headers.get('Authorization'), 'Bearer saved-token')
        assert.deepEqual(config.data === undefined ? undefined : JSON.parse(config.data), body)
        return { data: { code: 0, msg: 'success', data: payload }, status: 200, headers: {}, config }
      }
      assert.deepEqual(await auth[name](...args), payload)
      for (const code of [400, 401, 500, '0']) {
        apiClient.defaults.adapter = async (config) => ({
          data: { code, msg: '认证失败', data: null }, status: 200, headers: {}, config,
        })
        await assert.rejects(auth[name](...args), { message: '认证失败' })
      }
      const failure = new Error('Network Error')
      apiClient.defaults.adapter = async () => { throw failure }
      await assert.rejects(auth[name](...args), (error) => error === failure)
    }
  } finally {
    apiClient.defaults.adapter = originalAdapter
    if (originalStorage === undefined) delete globalThis.localStorage
    else globalThis.localStorage = originalStorage
  }
})
