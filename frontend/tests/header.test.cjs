const assert = require('node:assert/strict')
const { existsSync, readFileSync } = require('node:fs')
const Module = require('node:module')
const path = require('node:path')
const { test } = require('node:test')
const React = require('react')
const { act, create } = require('react-test-renderer')
const { MemoryRouter } = require('react-router-dom')
const ts = require('typescript')

test('header opens login and logs out locally only after the authenticated server request', async () => {
  const modules = new Map()
  const calls = []
  const openings = []
  let token = 'saved-token'
  let finishRequest
  let failRequest = false
  const user = { user_id: 'u1', username: 'alice', nick_name: 'Alice', avatar_url: '/avatar.png' }
  const context = {
    isLoggedIn: false, userInfo: null,
    setShowLoginDialog: (show) => openings.push(show),
    logout: () => {
      calls.push('local logout')
      token = null
      context.isLoggedIn = false
      context.userInfo = null
    },
  }
  const passthrough = ({ children }) => React.createElement('div', null, children)
  function load(relativePath) {
    let filename = path.resolve(__dirname, '../src', relativePath)
    if (!path.extname(filename)) filename += existsSync(`${filename}.ts`) ? '.ts' : '.tsx'
    if (modules.has(filename)) return modules.get(filename).exports
    const loaded = new Module(filename, module)
    loaded.paths = Module._nodeModulePaths(path.dirname(filename))
    loaded.require = (id) => {
      if (id === '@/hooks/useLogin') return { useLogin: () => context }
      if (id === '@/services/auth') return {
        postAuthLogout: () => {
          assert.equal(token, 'saved-token')
          calls.push('server logout')
          return new Promise((resolve, reject) => {
            finishRequest = () => failRequest ? reject(new Error('Network Error')) : resolve({ message: '登出成功' })
          })
        },
      }
      // Keep menu actions directly accessible in this renderer; real menu UI is browser-checked.
      if (id === '@radix-ui/react-dropdown-menu') return {
        Root: passthrough, Trigger: ({ children }) => children, Portal: passthrough,
        Content: passthrough,
        Item: ({ children, onSelect, disabled }) => React.createElement('button', { onClick: onSelect, disabled }, children),
      }
      return id.startsWith('@/') ? load(id.slice(2)) : module.require(id)
    }
    modules.set(filename, loaded)
    loaded._compile(ts.transpileModule(readFileSync(filename, 'utf8'), {
      compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022, jsx: ts.JsxEmit.ReactJSX },
    }).outputText, `${filename}.cjs`)
    return loaded.exports
  }
  const { default: Header } = load('components/Header')
  const render = () => React.createElement(MemoryRouter, {
    future: { v7_startTransition: true, v7_relativeSplatPath: true },
  }, React.createElement(Header))
  let tree
  try {
    await act(async () => { tree = create(render()) })
    act(() => tree.root.findAllByType('button').find((button) => button.props.children === '登录').props.onClick())
    assert.deepEqual(openings, [true])
    act(() => tree.unmount())

    for (const shouldFail of [false, true]) {
      calls.length = 0
      failRequest = shouldFail
      token = 'saved-token'
      context.isLoggedIn = true
      context.userInfo = user
      await act(async () => { tree = create(render()) })
      assert.match(JSON.stringify(tree.toJSON()), /Alice/)
      assert.equal(tree.root.findByType('img').props.src, '/avatar.png')
      const item = tree.root.findAllByType('button').find((button) => button.props.children === '退出登录')
      let result
      act(() => { result = item.props.onClick() })
      assert.deepEqual(calls, ['server logout'])
      assert.equal(token, 'saved-token')
      await act(async () => { finishRequest(); await result })
      assert.deepEqual(calls, ['server logout', 'local logout'])
      assert.equal(token, null)
      assert.match(JSON.stringify(tree.toJSON()), /登录/)
      if (shouldFail) assert.match(JSON.stringify(tree.toJSON()), /服务器暂未确认退出/)
      act(() => tree.unmount())
    }
  } finally {
    if (tree) act(() => tree.unmount())
  }
})
