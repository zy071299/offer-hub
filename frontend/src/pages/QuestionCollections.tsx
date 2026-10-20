import { Link, useSearchParams } from 'react-router-dom'

import { Button } from '@/components/ui/button'
import { useQuestionBanks } from '@/hooks/useQuestionQueries'

export default function QuestionCollections() {
  const [searchParams, setSearchParams] = useSearchParams()
  const { data: groups = [], isPending, isError, error, refetch, isFetching } = useQuestionBanks()
  const rawIndex = searchParams.get('activeIndex') ?? '0'
  const requestedIndex = /^\d+$/.test(rawIndex) ? Number(rawIndex) : 0
  const activeIndex = Number.isSafeInteger(requestedIndex) && requestedIndex < groups.length
    ? requestedIndex
    : 0
  const activeGroup = groups[activeIndex]

  function selectGroup(index: number) {
    setSearchParams((previous) => {
      const next = new URLSearchParams(previous)
      next.set('activeIndex', String(index))
      return next
    })
  }

  return (
    <main className="mx-auto max-w-6xl space-y-8 px-6 py-12">
      <header className="space-y-3">
        <h1 className="text-3xl font-semibold tracking-tight">题库分类</h1>
        <p className="text-muted-foreground">选择领域，找到你想练习的题库。</p>
      </header>

      {isPending ? (
        <section role="status" aria-label="正在加载题库" aria-busy="true" className="space-y-6">
          <span className="sr-only">正在加载题库…</span>
          <div aria-hidden="true" className="flex gap-3">
            {[0, 1, 2].map((index) => (
              <div key={index} className="h-9 w-24 animate-pulse rounded-md bg-muted" />
            ))}
          </div>
          <div aria-hidden="true" className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {[0, 1, 2, 3, 4, 5].map((index) => (
              <div key={index} className="space-y-4 rounded-xl border p-6">
                <div className="h-6 w-2/3 animate-pulse rounded bg-muted" />
                <div className="h-4 w-full animate-pulse rounded bg-muted" />
                <div className="h-4 w-4/5 animate-pulse rounded bg-muted" />
              </div>
            ))}
          </div>
        </section>
      ) : isError ? (
        <section role="alert" className="space-y-4 rounded-xl border border-destructive/30 p-6">
          <h2 className="font-semibold">题库加载失败</h2>
          <p className="break-words text-sm text-muted-foreground">{error.message || '请稍后重试。'}</p>
          <Button variant="outline" disabled={isFetching} onClick={() => void refetch()}>
            {isFetching ? '正在重试…' : '重新加载'}
          </Button>
        </section>
      ) : groups.length === 0 ? (
        <p className="rounded-xl border p-8 text-center text-muted-foreground">暂无题库，敬请期待。</p>
      ) : (
        <>
          <nav aria-label="题库领域" className="flex flex-wrap gap-3">
            {groups.map((group, index) => (
              <Button
                key={group.job_name}
                variant={index === activeIndex ? 'default' : 'outline'}
                aria-pressed={index === activeIndex}
                onClick={() => selectGroup(index)}
              >
                {group.job_name}
              </Button>
            ))}
          </nav>

          <div className="space-y-8">
            {activeGroup?.series_list.length === 0 && (
              <p className="rounded-xl border p-8 text-center text-muted-foreground">该领域暂无题库。</p>
            )}
            {activeGroup?.series_list.map((series) => (
              <section key={series.series_id} aria-label={series.series_name} className="space-y-4">
                <h2 className="text-xl font-semibold">{series.series_name}</h2>
                {series.bank_list.length === 0 ? (
                  <p className="text-sm text-muted-foreground">该系列暂无题库。</p>
                ) : (
                  <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                    {series.bank_list.map((bank) => (
                      <Link
                        key={bank.bank_id}
                        to={`/questions?${new URLSearchParams({
                          bank_id: bank.bank_id,
                          activeIndex: String(activeIndex),
                        })}`}
                        className="block min-w-0 space-y-3 rounded-xl border bg-card p-6 text-card-foreground shadow-sm transition-colors hover:border-primary/40 hover:bg-accent/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                      >
                        <h3 className="break-words text-lg font-semibold">{bank.bank_name}</h3>
                        <p className="break-words text-sm leading-6 text-muted-foreground">
                          {bank.desc || '暂无描述'}
                        </p>
                      </Link>
                    ))}
                  </div>
                )}
              </section>
            ))}
          </div>
        </>
      )}
    </main>
  )
}
