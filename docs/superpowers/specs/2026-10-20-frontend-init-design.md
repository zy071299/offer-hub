# frontend 初始化设计

## 目标与范围

在仓库根目录新建 `frontend/`，提供可安装、可启动、可构建的前端基础项目，供后续 Offer Hub 业务开发使用。采用用户指定的 React 18、TypeScript、Vite、Tailwind CSS、shadcn/ui、React Router v6、@tanstack/react-query、axios、React Hook Form 与 Zod。使用 npm 并提交 package-lock.json，具体依赖版本在实施时检查兼容性后锁定。

本次只建立基础项目、首页与 404 页面，不接入业务 API，不增加登录、业务表单或模拟业务数据。现有 backend 文件及用户未提交修改不在本次范围内。

## 项目组织

```text
frontend/
├── src/
│   ├── components/
│   │   ├── provider/       # 应用 Provider，集中挂载 QueryClientProvider
│   │   └── ui/             # shadcn/ui Button 及后续组件
│   ├── pages/              # HomePage、NotFoundPage
│   ├── services/           # 业务 API 封装预留目录
│   ├── hooks/              # 自定义 Hook 预留目录
│   ├── types/              # 通用类型预留目录
│   ├── lib/                # axios 实例、QueryClient、shadcn cn 工具
│   ├── utils/              # 其他工具函数预留目录
│   ├── App.tsx             # 路由定义
│   ├── main.tsx            # React 入口
│   └── index.css           # Tailwind 和 shadcn 主题样式
├── components.json         # shadcn CLI 配置
├── .env.example
├── package.json
├── package-lock.json
├── README.md
└── 构建、TypeScript 与代码检查配置
```

空的预留目录通过 `.gitkeep` 保留。Vite 与 TypeScript 共同配置 `@/` 指向 `src/`。

## 运行与数据流

React 根节点使用 StrictMode。应用 Provider 创建并复用 QueryClient，路由采用 BrowserRouter，`/` 显示简洁初始化首页，其他路径显示带返回首页链接的 404 页面。首页使用 Button 验证 shadcn/ui 与 Tailwind 的集成。

后续数据请求路径为页面或 Hook → service → axios 实例。axios 的 baseURL 读取 `VITE_API_BASE_URL`，未配置时使用同源地址，设置合理超时。HTTP 错误保持 Promise 拒绝，交由调用方处理；初始化阶段不增加鉴权、自动跳转或业务响应解包。

React Hook Form、Zod 及 `@hookform/resolvers` 安装为依赖，供后续表单集成；初始化不额外展示演示表单。

## 样式与组件

配置 Tailwind CSS 和 shadcn/ui 所需主题变量、工具函数及依赖，提供 Button 组件和可继续通过 shadcn CLI 添加组件的配置。首页仅展示项目名称和初始化状态，使用响应式基础布局。

## 验收

- `npm install` 可完成并生成锁文件。
- `npm run dev` 能启动 Vite，首页及未知路径可访问。
- `npm run typecheck`、`npm run lint`、`npm run build` 均通过。
- 生产构建包含页面样式，别名和 React 18、Router v6 版本符合约束。
- README 说明依赖安装、开发、检查、构建、环境变量及目录用途，并说明生产部署需要 SPA fallback。
- 无业务请求或业务功能测试需求；通过类型检查、代码检查、构建与页面冒烟检查验证初始化。
