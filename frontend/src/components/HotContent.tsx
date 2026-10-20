import { Link } from 'react-router-dom'

import { Button } from '@/components/ui/button'
import { useHotQuestions } from '@/hooks/useQuestionQueries'

export default function HotContent() {
  const { data, isPending, isError, error, refetch, isFetching } = useHotQuestions()

  return (
    <aside aria-labelledby="hot-questions-title" className="h-fit space-y-5 rounded-xl border bg-card p-6">
      <h2 id="hot-questions-title" className="text-xl font-semibold">热门题目</h2>
      {isPending ? (
        <div role="status" aria-label="正在加载热门题目" className="space-y-4">
          <span className="sr-only">正在加载热门题目…</span>
          {[0, 1, 2, 3, 4].map((index) => (
            <div key={index} aria-hidden="true" className="h-5 animate-pulse rounded bg-muted" />
          ))}
        </div>
      ) : isError ? (
        <div role="alert" className="space-y-3">
          <p className="break-words text-sm text-destructive">{error.message || '热门题目加载失败'}</p>
          <Button variant="outline" size="sm" disabled={isFetching} onClick={() => void refetch()}>
            {isFetching ? '正在重试…' : '重新加载'}
          </Button>
        </div>
      ) : !data?.list.length ? (
        <p className="text-sm text-muted-foreground">暂无热门题目。</p>
      ) : (
        <ol className="space-y-4">
          {data.list.map((question, index) => (
            <li key={question.question_id}>
              <Link
                to={`/questions/${encodeURIComponent(question.question_id)}`}
                className="flex gap-3 rounded-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              >
                <span aria-hidden="true" className="text-sm font-semibold text-muted-foreground">{index + 1}.</span>
                <div className="min-w-0 space-y-1">
                  <h3 className="break-words text-sm font-medium hover:underline">{question.title}</h3>
                  <p className="text-xs text-muted-foreground">{question.view_count} 次浏览</p>
                </div>
              </Link>
            </li>
          ))}
        </ol>
      )}
    </aside>
  )
}
