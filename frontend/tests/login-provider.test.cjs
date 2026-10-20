const assert = require('node:assert/strict')
const { readFileSync, existsSync } = require('node:fs')
const Module = require('node:module')
const path = require('node:path')
const { test } = require('node:test')
const ts = require('typescript')
const React = require('react')
const { act, create } = require('react-test-renderer')

const dto = {
  userId: 'u1', username: 'alice', nickName: 'Alice', avatar: '/avatar.png',
  sex: 1, vip: false, phone: '', email: '', userStatus: 1, userType: 1,
}

test('login provider restores, maps, logs out and ignores obsolete user requests', async () => {
  const originalStorage = globalThis.localStorage
  const storage = new Map()
  globalThis.localStorage = {
    getItem: (key) => storage.get(key) ?? null,
    setItem: (key, value) => storage.set(key, value),
    removeItem: (key) => storage.delete(key),
  }
  const modules = new Map()
  const pending = []
  let getCalls = 0
  function loadTs(relativePath) {
    let filename = path.resolve(__dirname, '../src', relativePath)
    if (!path.extname(filename)) filename += existsSync(`${filename}.ts`) ? '.ts' : '.tsx'
    if (modules.has(filename)) return modules.get(filename).exports
    const loaded = new Module(filename, module)
    loaded.paths = Module._nodeModulePaths(path.dirname(filename))
    loaded.require = (id) => {
      if (id === '@/services/auth') {
        return {
          ...loadTs('services/auth'),
          getUserInfo: () => {
            getCalls++
            return new Promise((resolve, reject) => pending.push({ resolve, reject }))
          },
        }
      }
      return id.startsWith('@/') ? loadTs(id.slice(2)) : module.require(id)
    }
    modules.set(filename, loaded)
    const compiled = ts.transpileModule(readFileSync(filename, 'utf8'), {
      compilerOptions: {
        module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022,
        jsx: ts.JsxEmit.ReactJSX,
      },
    }).outputText
    loaded._compile(compiled, `${filename}.cjs`)
    return loaded.exports
  }
  let tree
  try {
    const { LoginProvider } = loadTs('components/provider/LoginProvider')
    const { useLogin } = loadTs('hooks/useLogin')
    const { queryClient } = loadTs('lib/query-client')
    queryClient.setQueryData(['questions', 'detail', 'q1'], { content: 'preview' })
    queryClient.setQueryData(['questions', 'list'], { total: 1 })
    const { mapPasswordAuthUserToUserInfo } = loadTs('services/auth')
    const user = mapPasswordAuthUserToUserInfo(dto)
    let context
    function Consumer() { context = useLogin(); return null }
    const render = () => create(React.createElement(LoginProvider, null, React.createElement(Consumer)))
    await act(async () => { tree = render() })
    assert.equal(getCalls, 0)
    assert.equal(context.isLoggedIn, false)
    assert.equal(context.userInfo, null)
    act(() => context.setShowLoginDialog(true))
    assert.equal(context.showLoginDialog, true)
    act(() => context.login(dto, 'token-a'))
    assert.deepEqual(context.userInfo, user)
    assert.equal(context.isLoggedIn, true)
    assert.equal(storage.get('token'), 'token-a')
    assert.equal(context.showLoginDialog, false)
    assert.equal(queryClient.getQueryState(['questions', 'detail', 'q1']).isInvalidated, true)
    assert.equal(queryClient.getQueryState(['questions', 'list']).isInvalidated, false)

    let refresh
    act(() => { refresh = context.refreshUserInfo() })
    await act(async () => { pending.shift().resolve({ ...user, nick_name: 'Updated' }); await refresh })
    assert.equal(context.userInfo.nick_name, 'Updated')

    act(() => { refresh = context.refreshUserInfo() })
    act(() => context.logout())
    await act(async () => { pending.shift().resolve(user); await refresh })
    assert.equal(context.isLoggedIn, false)
    assert.equal(storage.has('token'), false)

    act(() => context.login(dto, 'token-old'))
    act(() => { refresh = context.refreshUserInfo() })
    act(() => context.login({ ...dto, userId: 'u2' }, 'token-new'))
    await act(async () => { pending.shift().resolve(user); await refresh })
    assert.equal(context.userInfo.user_id, 'u2')
    act(() => tree.unmount())

    await act(async () => { tree = render() })
    assert.equal(context.isLoggedIn, false)
    await act(async () => { pending.shift().resolve(user) })
    assert.equal(context.isLoggedIn, true)
    act(() => { refresh = context.refreshUserInfo() })
    const failure = new Error('Invalid token')
    const rejected = assert.rejects(refresh, (error) => error === failure)
    await act(async () => { pending.shift().reject(failure); await rejected })
    assert.equal(context.userInfo, null)
    assert.equal(storage.has('token'), false)
  } finally {
    if (tree) act(() => tree.unmount())
    if (originalStorage === undefined) delete globalThis.localStorage
    else globalThis.localStorage = originalStorage
  }
})
