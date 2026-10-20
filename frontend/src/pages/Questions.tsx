import type { FormEvent } from 'react'
import { Link, useSearchParams } from 'react-router-dom'

import HotContent from '@/components/HotContent'
import { Button } from '@/components/ui/button'
import { useQuestionList } from '@/hooks/useQuestionQueries'
import type { QuestionFilters } from '@/types/question'

const PAGE_SIZE = 20
const difficultyLabels = { 1: '简单', 2: '中等', 3: '困难' } as const

export default function Questions() {
  const [searchParams, setSearchParams] = useSearchParams()
  const bankId = searchParams.get('bank_id') || undefined
  const keyword = searchParams.get('keyword') || ''
  const rawDifficulty = searchParams.get('difficulty')
  const difficulty = rawDifficulty === '1' || rawDifficulty === '2' || rawDifficulty === '3'
    ? Number(rawDifficulty) as 1 | 2 | 3
    : undefined
  const rawPage = searchParams.get('page') ?? '1'
  const parsedPage = /^\d+$/.test(rawPage) ? Number(rawPage) : 1
  const page = Number.isSafeInteger(parsedPage) && parsedPage > 0 ? parsedPage : 1
  const filters: QuestionFilters = {
    bank_id: bankId,
    difficulty,
    keyword: keyword || undefined,
    page,
    page_size: PAGE_SIZE,
  }
  const { data, isPending, isError, error, refetch, isFetching } = useQuestionList(filters)
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const firstPage = Math.max(1, Math.min(page - 2, totalPages - 4))
  const visiblePages = Array.from(
    { length: Math.min(5, totalPages) }, (_, index) => firstPage + index,
  )

  function updateFilter(name: 'difficulty' | 'keyword', value: string) {
    setSearchParams((previous) => {
      const next = new URLSearchParams(previous)
      if (value) next.set(name, value)
      else next.delete(name)
      next.set('page', '1')
      return next
    })
  }

  function search(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const value = new FormData(event.currentTarget).get('keyword')
    updateFilter('keyword', typeof value === 'string' ? value.trim() : '')
  }

  function changePage(nextPage: number) {
    setSearchParams((previous) => {
      const next = new URLSearchParams(previous)
      next.set('page', String(Math.max(1, Math.min(nextPage, totalPages))))
      return next
    })
  }

  const collectionsParams = new URLSearchParams()
  const activeIndex = searchParams.get('activeIndex')
  if (activeIndex !== null) collectionsParams.set('activeIndex', activeIndex)

  return (
    <main className="mx-auto max-w-6xl space-y-8 px-6 py-12">
      <header className="space-y-3">
        <Link to={`/collections?${collectionsParams}`} className="text-sm text-muted-foreground hover:underline">
          返回题库分类
        </Link>
        <h1 className="text-3xl font-semibold tracking-tight">题目列表</h1>
        {bankId && <p className="break-words text-sm text-muted-foreground">当前题库：{bankId}</p>}
      </header>

      <div className="grid items-start gap-8 lg:grid-cols-[minmax(0,1fr)_20rem]">
        <div className="min-w-0 space-y-6">
          <section aria-label="题目筛选" className="flex flex-wrap items-end gap-4 rounded-xl border p-5">
            <label className="space-y-2 text-sm font-medium">
              <span className="block">难度</span>
              <select
                value={difficulty ?? ''}
                onChange={(event) => updateFilter('difficulty', event.target.value)}
                className="h-9 rounded-md border bg-background px-3 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              >
                <option value="">全部难度</option>
                <option value="1">简单</option>
                <option value="2">中等</option>
                <option value="3">困难</option>
              </select>
            </label>
            <form onSubmit={search} className="flex min-w-0 flex-1 flex-wrap items-end gap-2">
              <label className="min-w-0 flex-1 space-y-2 text-sm font-medium">
                <span className="block">关键词</span>
                <input
                  key={keyword}
                  name="keyword"
                  type="search"
                  defaultValue={keyword}
                  placeholder="搜索题目标题或内容"
                  className="h-9 w-full rounded-md border bg-background px-3 font-normal focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                />
              </label>
              <Button type="submit">搜索</Button>
            </form>
          </section>

          {isPending ? (
            <div role="status" aria-label="正在加载题目" className="space-y-3">
              <span className="sr-only">正在加载题目…</span>
              {[0, 1, 2, 3, 4].map((index) => (
                <div key={index} aria-hidden="true" className="h-24 animate-pulse rounded-xl bg-muted" />
              ))}
            </div>
          ) : isError ? (
            <section role="alert" className="space-y-4 rounded-xl border p-6">
              <h2 className="font-semibold">题目加载失败</h2>
              <p className="break-words text-sm text-destructive">{error.message}</p>
              <Button variant="outline" disabled={isFetching} onClick={() => void refetch()}>
                {isFetching ? '正在重试…' : '重新加载'}
              </Button>
            </section>
          ) : (
            <>
              <p role="status" className="text-sm text-muted-foreground">共 {total} 道题目，每页 {PAGE_SIZE} 条</p>
              {!data?.list.length ? (
                <div className="space-y-4 rounded-xl border p-8 text-center">
                  <p className="text-muted-foreground">暂无符合条件的题目。</p>
                  {page > totalPages && <Button variant="outline" onClick={() => changePage(1)}>返回第一页</Button>}
                </div>
              ) : (
                <ul className="divide-y rounded-xl border">
                  {data.list.map((question) => (
                    <li key={question.question_id}>
                      <Link
                        to={`/questions/${encodeURIComponent(question.question_id)}`}
                        className="block space-y-3 rounded-md p-5 transition-colors hover:bg-accent/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                      >
                        <h2 className="break-words font-medium">{question.title}</h2>
                        <div className="flex flex-wrap gap-3 text-xs text-muted-foreground">
                          <span className="rounded bg-muted px-2 py-1">{difficultyLabels[question.difficulty]}</span>
                          <span className="py-1">{question.view_count} 次浏览</span>
                        </div>
                      </Link>
                    </li>
                  ))}
                </ul>
              )}
              <nav aria-label="题目分页" className="flex flex-wrap items-center justify-center gap-2">
                <Button variant="outline" size="sm" disabled={page <= 1 || isFetching} onClick={() => changePage(page - 1)}>
                  上一页
                </Button>
                {visiblePages.map((number) => (
                  <Button
                    key={number}
                    variant={number === page ? 'default' : 'outline'}
                    size="sm"
                    aria-label={`第 ${number} 页`}
                    aria-current={number === page ? 'page' : undefined}
                    disabled={isFetching}
                    onClick={() => changePage(number)}
                  >
                    {number}
                  </Button>
                ))}
                <Button variant="outline" size="sm" disabled={page >= totalPages || isFetching} onClick={() => changePage(page + 1)}>
                  下一页
                </Button>
                <span className="text-sm text-muted-foreground">第 {page} / {totalPages} 页</span>
              </nav>
            </>
          )}
        </div>
        <HotContent />
      </div>
    </main>
  )
}
