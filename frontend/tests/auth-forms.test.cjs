const assert = require('node:assert/strict')
const { existsSync, readFileSync } = require('node:fs')
const Module = require('node:module')
const path = require('node:path')
const { test } = require('node:test')
const React = require('react')
const { act, create } = require('react-test-renderer')
const ts = require('typescript')

function makeLoader(overrides = {}) {
  const cache = new Map()
  function load(relativePath) {
    let filename = path.resolve(__dirname, '../src', relativePath)
    if (!path.extname(filename)) filename += existsSync(`${filename}.ts`) ? '.ts' : '.tsx'
    if (cache.has(filename)) return cache.get(filename).exports
    const loaded = new Module(filename, module)
    loaded.paths = Module._nodeModulePaths(path.dirname(filename))
    loaded.require = (id) => overrides[id] ?? (id.startsWith('@/') ? load(id.slice(2)) : module.require(id))
    cache.set(filename, loaded)
    loaded._compile(ts.transpileModule(readFileSync(filename, 'utf8'), {
      compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022, jsx: ts.JsxEmit.ReactJSX },
    }).outputText, `${filename}.cjs`)
    return loaded.exports
  }
  return load
}

test('auth validation checks username, password length and registration confirmation', () => {
  const { createPasswordAuthSchema } = makeLoader()('lib/auth-validation')
  const login = createPasswordAuthSchema('login')
  const register = createPasswordAuthSchema('register')
  assert.equal(login.safeParse({ username: '  ', password: '123456', confirmPassword: '' }).success, false)
  assert.equal(login.safeParse({ username: 'alice', password: '12345', confirmPassword: '' }).success, false)
  assert.equal(register.safeParse({ username: 'alice', password: '123456', confirmPassword: '654321' }).success, false)
  assert.equal(register.safeParse({ username: 'alice', password: '123456', confirmPassword: '123456' }).success, true)
  assert.deepEqual(login.parse({ username: ' alice ', password: ' 12345 ', confirmPassword: '' }), {
    username: 'alice', password: ' 12345 ', confirmPassword: '',
  })
})

test('form validates, switches registration to login, shows failures and completes login', async () => {
  const requests = []
  const logins = []
  let rejectLogin = false
  let finishLogin
  const loginData = { token: 'test-token', userInfo: { userId: 'u1' } }
  const load = makeLoader({
    '@/hooks/useLogin': { useLogin: () => ({ login: (...args) => logins.push(args) }) },
    '@/services/auth': {
      postAuthRegister: async (...args) => requests.push(['register', ...args]),
      postAuthLogin: async (...args) => {
        requests.push(['login', ...args])
        if (rejectLogin) throw new Error('用户名或密码错误')
        return new Promise((resolve) => { finishLogin = () => resolve(loginData) })
      },
    },
  })
  const { default: Forms } = load('components/auth/UsernamePasswordAuthForms')
  let tree
  try {
    await act(async () => { tree = create(React.createElement(Forms)) })
    const input = (name) => tree.root.findAllByType('input').find((item) => item.props.name === name)
    const fill = async (name, value) => act(async () => {
      await input(name).props.onChange({ target: { name, value }, type: 'change' })
    })
    const submit = () => tree.root.findByType('form').props.onSubmit({ preventDefault() {}, persist() {} })
    await act(async () => { await submit() })
    assert.equal(requests.length, 0)
    assert.match(JSON.stringify(tree.toJSON()), /请输入用户名/)
    await act(async () => tree.root.findAllByType('button').find((b) => b.props.type === 'button').props.onClick())
    await fill('username', 'alice')
    await fill('password', '123456')
    await fill('confirmPassword', '654321')
    await act(async () => { await submit() })
    assert.equal(requests.length, 0)
    assert.match(JSON.stringify(tree.toJSON()), /两次密码不一致/)
    await fill('confirmPassword', '123456')
    await act(async () => { await submit() })
    assert.deepEqual(requests[0], ['register', 'alice', '123456'])
    assert.equal(input('confirmPassword'), undefined)
    assert.match(JSON.stringify(tree.toJSON()), /注册成功/)
    rejectLogin = true
    await fill('password', '123456')
    await act(async () => { await submit() })
    assert.match(JSON.stringify(tree.toJSON()), /用户名或密码错误/)
    rejectLogin = false
    let submission
    await act(async () => { submission = submit(); await Promise.resolve() })
    assert.equal(tree.root.findAllByType('button').find((b) => b.props.type === 'submit').props.disabled, true)
    await act(async () => { finishLogin(); await submission })
    assert.deepEqual(logins, [[loginData.userInfo, loginData.token]])
  } finally {
    if (tree) act(() => tree.unmount())
  }
})
