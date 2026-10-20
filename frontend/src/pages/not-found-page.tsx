import { Link } from 'react-router-dom'

import { Button } from '@/components/ui/button'

export function NotFoundPage() {
  return (
    <main className="flex min-h-svh flex-col items-center justify-center gap-5 p-6 text-center">
      <p className="text-sm font-medium text-muted-foreground">404</p>
      <h1 className="text-3xl font-semibold">页面未找到</h1>
      <p className="text-muted-foreground">你访问的页面不存在。</p>
      <Button asChild><Link to="/">返回首页</Link></Button>
    </main>
  )
}
