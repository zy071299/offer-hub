# Frontend Initialization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 `frontend/` 建立满足已确认技术栈和目录要求、可启动并通过检查的基础项目。

**Architecture:** Vite 提供 React 单页应用构建；应用 Provider 统一挂载 React Query，React Router 管理首页及 404。axios 实例作为未来 service 的请求入口，Tailwind 与 shadcn/ui 提供基础样式和 Button。

**Tech Stack:** React 18、TypeScript、Vite、Tailwind CSS、shadcn/ui、React Router v6、@tanstack/react-query、axios、React Hook Form、Zod、@hookform/resolvers、npm。

**Spec:** `docs/superpowers/specs/2026-10-20-frontend-init-design.md`

## Global Constraints

- React 必须为 18，React Router 必须为 v6；安装后通过 npm ls 验证，使用 lockfile 固定实际版本。
- 所有应用文件置于 frontend；不修改 backend 或用户已有修改。
- 保留 components/provider、components/ui、pages、services、hooks、types、lib、utils 目录。
- Vite 与 TypeScript 都支持 @/ 别名。
- VITE_API_BASE_URL 未配置时使用同源地址；axios 超时设为 10000ms，保留原始错误传播。
- 不实现业务 API、鉴权或演示表单；空目录用 .gitkeep 保留。
- 本次基础脚手架采用类型、lint、构建与页面冒烟检查，不增加镜像实现的单元测试。

## Review Focus

- Node 22.15.0 环境：选择兼容该 Node 版本的 Vite 和插件，启动与构建不得有引擎不兼容错误。
- 未配置接口地址：检查 axios 默认 baseURL 为同源地址，不触发初始化网络请求。
- 未知路由：访问不存在路径显示 404，点击返回首页可恢复。
- 样式产物：生产 CSS 包含 Tailwind 工具样式与主题变量，首页 Button 正常呈现。
- SPA 深链接：README 明确生产服务器须回退 index.html，开发环境访问未知路径仍可加载应用。

## Task 1: 建立可构建的工具链和 UI 基础

**Files:** Create `frontend/package.json`, `frontend/package-lock.json`, `frontend/index.html`, `frontend/tsconfig.json`, `frontend/tsconfig.app.json`, `frontend/tsconfig.node.json`, `frontend/vite.config.ts`, `frontend/eslint.config.js`, `frontend/components.json`, `frontend/src/vite-env.d.ts`, `frontend/src/index.css`, `frontend/src/lib/utils.ts`, `frontend/src/components/ui/button.tsx`。

**Interfaces:** 导出 `cn(...inputs: ClassValue[]): string`；导出遵循所选 shadcn 官方实现的 `Button` 和 `buttonVariants`。配置 npm scripts：dev、build、preview、lint、typecheck。

- [ ] 检查官方 Vite、Tailwind、shadcn 安装说明与包 engine/peerDependencies，选择相互兼容且支持 Node 22.15.0 的版本；保留 React 18 与 Router v6 的主版本约束。
- [ ] 创建 package.json 和 TypeScript/Vite/ESLint 配置；build 执行类型检查及 vite build，typecheck 同时检查应用与 Vite 配置。
- [ ] 配置 Tailwind 和 shadcn 主题、components.json、cn 及 Button；别名与 CSS 路径一致。
- [ ] 执行 npm install；执行 npm ls react react-dom react-router-dom，预期 React/React DOM 为 18.x，Router 为 6.x，无无效 peer dependency。

## Task 2: 连接入口、Provider、路由与 HTTP 实例

**Files:** Create `frontend/src/main.tsx`, `frontend/src/App.tsx`, `frontend/src/components/provider/app-provider.tsx`, `frontend/src/lib/query-client.ts`, `frontend/src/lib/axios.ts`, `frontend/src/pages/home-page.tsx`, `frontend/src/pages/not-found-page.tsx`, `frontend/.env.example`；在 services、hooks、types、utils 中创建 `.gitkeep`。

**Interfaces:** `AppProvider({ children }: PropsWithChildren)`；命名导出 `queryClient: QueryClient`、`apiClient: AxiosInstance`；页面导出 `HomePage`、`NotFoundPage`；App 与 main 负责路由及挂载。

- [ ] 创建稳定 QueryClient 和 AppProvider，main 使用 StrictMode、AppProvider、BrowserRouter 挂载 App。
- [ ] App 中 `/` 渲染 HomePage，`*` 渲染 NotFoundPage；首页展示 Offer Hub 和初始化状态，404 使用 Button asChild 包装返回首页 Link。
- [ ] 创建 apiClient，baseURL 使用 `import.meta.env.VITE_API_BASE_URL || '/'`，timeout 为 10000，不增加拦截器或自动请求；声明可选环境变量类型并补充 .env.example。
- [ ] 执行 npm run typecheck、npm run lint、npm run build；均须退出码 0；检查 dist 中生成 JS/CSS。
- [ ] 启动 Vite，检查首页、未知路径和返回首页行为；检查控制台、样式和窄屏布局，并确认首页没有业务 API 请求。

## Task 3: 文档与交付检查

**Files:** Create `frontend/README.md`；仅在验证发现问题时修改上面新建的文件。

**Interfaces:** README 给出可复制的安装、启动、检查、构建命令及环境变量配置方式。

- [ ] 写明 Node 版本要求、npm install、npm run dev、npm run typecheck、npm run lint、npm run build、npm run preview。
- [ ] 说明目录用途、@/ 别名、添加 shadcn 组件的方法、VITE_API_BASE_URL 默认同源行为及 SPA fallback 部署要求。
- [ ] 检查 lockfile、环境文件忽略规则与目录跟踪；执行 git diff --check，并检查 git status，确保原有 backend 修改未被覆盖。
- [ ] 最终报告实际完成内容、验证结果和启动方式；若有安装或页面验证限制，明确列出。
