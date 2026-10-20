import { Button } from '@/components/ui/button'

export function HomePage() {
  return (
    <main className="flex min-h-svh items-center justify-center bg-muted/40 p-6">
      <section className="w-full max-w-lg rounded-xl border bg-card p-8 text-card-foreground shadow-sm sm:p-12">
        <p className="text-sm font-medium text-muted-foreground">欢迎使用</p>
        <h1 className="mt-3 text-4xl font-semibold tracking-tight">Offer Hub</h1>
        <p className="mt-4 leading-7 text-muted-foreground">
          前端项目已初始化，可以开始构建你的应用。
        </p>
        <Button className="mt-8" asChild>
          <a href="https://ui.shadcn.com/docs" target="_blank" rel="noreferrer">
            查看组件文档
          </a>
        </Button>
      </section>
    </main>
  )
}
