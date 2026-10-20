# Offer Hub 前端

React 18 + TypeScript + Vite 7。样式使用 Tailwind CSS 4，组件采用 shadcn/ui（new-york），路由使用 React Router v6。

## 启动

需要 Node.js 20.19+ 或 22.12+（建议使用受支持的 Node LTS）以及 npm。本项目已在 Node 22.15.0 环境配置。

```sh
cd frontend
npm install
npm run dev
```

按终端提示打开开发地址，默认为 http://localhost:3001。

```sh
npm run typecheck
npm run lint
npm run build
npm run preview
```

构建结果位于 `dist/`。已有锁文件时可使用 `npm ci` 重现安装。`preview` 仅供本地预览。

## 目录

```text
src/
├── components/
│   ├── provider/  # 应用 Context Provider
│   └── ui/        # shadcn/ui 组件源码
├── pages/         # 首页与 404 页面
├── services/      # 业务 API 封装预留
├── hooks/         # 自定义 Hook 预留
├── types/         # TypeScript 类型预留
├── lib/           # axios、QueryClient 与 cn 工具
└── utils/         # 其他工具函数预留
```

`@/` 指向 `src/`，TypeScript 与 Vite 已同时配置。

## 请求与表单

从 `@/lib/axios` 导入 `apiClient`，在 services 中编写具体请求。`baseURL` 固定为空字符串，请使用 `/api/...` 等相对路径；开发时 Vite 将 `/api`、`/auth`、`/home` 开头的请求转发至 `http://localhost:8180`，设置 `changeOrigin: true` 并保留原始路径。生产时由服务器反向代理。

实例超时为 10 秒，每次请求读取 `localStorage` 的 `token`，存在时添加 `Authorization: Bearer <token>`。HTTP 401 会删除该 token；成功响应保持 AxiosResponse，错误继续抛给调用方。`VITE_API_BASE_URL` 不再用于此实例。页面或 Hook 使用已挂载 Provider 的 `@tanstack/react-query` 处理请求状态。

运行 axios 行为检查：`node --test tests/axios.test.cjs`。

已安装 `react-hook-form`、`zod`、`@hookform/resolvers`，后续表单可使用 `zodResolver` 集成校验。首页不会自动请求业务 API。

## 添加组件

在 frontend 目录运行：

```sh
npx shadcn@latest add input
```

组件写入 `src/components/ui/`，主题位于 `src/index.css`。Button 已包含常用变体及 `asChild`，支持通过 React 18 的 forwardRef 转发引用。

## 部署

将 `dist/` 部署为静态站点。由于使用 BrowserRouter，服务器必须将非静态资源路径回退到 `index.html`，否则直接访问或刷新深层路由会返回服务器 404。Nginx 可使用 `try_files $uri $uri/ /index.html;`，API 路径应单独配置反向代理。

## 依赖审计记录

2026-10-20 的 `npm audit` 报告 Router v6 依赖链中有两项中危记录，涉及外部输入驱动的导航及 SSR hydration；建议修复版本为 v7.18.4。本项目按要求保留 v6，当前仅使用固定站内路径且未启用 SSR。后续增加动态跳转时应校验目标，升级主版本需单独评估。
