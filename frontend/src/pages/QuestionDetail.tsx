import axios from 'axios'
import { Link, useParams, useSearchParams } from 'react-router-dom'

import { Button } from '@/components/ui/button'
import DetailContent from '@/components/detail/DetailContent'
import Comments from '@/components/comment/Comments'
import { useBankQuestionList, useQuestionDetail } from '@/hooks/useQuestionQueries'
import { QuestionApiError } from '@/services/question'
import { TargetType } from '@/types/comment'

const difficultyLabels = { 1: '简单', 2: '中等', 3: '困难' } as const

export default function QuestionDetail() {
  const { question_id: questionId = '' } = useParams<{ question_id: string }>()
  const [searchParams] = useSearchParams()
  const detail = useQuestionDetail(questionId)
  const banks = detail.data?.bank_list ?? []
  const requestedBank = searchParams.get('bank_id')
  const bankId = requestedBank && banks.includes(requestedBank) ? requestedBank : banks[0] ?? ''
  const bankQuestions = useBankQuestionList(bankId)
  const navigationParams = new URLSearchParams()
  if (bankId) navigationParams.set('bank_id', bankId)
  const activeIndex = searchParams.get('activeIndex')
  if (activeIndex !== null) navigationParams.set('activeIndex', activeIndex)
  const navigationSearch = navigationParams.size ? `?${navigationParams}` : ''
  const notFound = !questionId.trim()
    || (detail.error instanceof QuestionApiError && detail.error.code === 404)
    || (axios.isAxiosError(detail.error) && detail.error.response?.status === 404)

  if (notFound) {
    return (
      <main className="flex min-h-svh flex-col items-center justify-center gap-5 p-6 text-center">
        <p className="text-sm text-muted-foreground">404</p>
        <h1 className="text-3xl font-semibold">题目不存在</h1>
        <p className="text-muted-foreground">该题目可能已被删除或链接有误。</p>
        <Button asChild><Link to="/questions">返回题目列表</Link></Button>
      </main>
    )
  }

  return (
    <main className="mx-auto max-w-7xl space-y-6 px-6 py-10">
      <Link to={`/questions${navigationSearch}`} className="text-sm text-muted-foreground hover:underline">
        返回题目列表
      </Link>
      {detail.isPending ? (
        <div role="status" aria-label="正在加载题目详情" className="grid gap-8 lg:grid-cols-[16rem_minmax(0,1fr)]">
          <span className="sr-only">正在加载题目详情…</span>
          <div aria-hidden="true" className="h-80 animate-pulse rounded-xl bg-muted" />
          <div aria-hidden="true" className="space-y-6 rounded-xl border p-8">
            <div className="h-9 w-3/4 animate-pulse rounded bg-muted" />
            <div className="h-6 w-1/3 animate-pulse rounded bg-muted" />
            {[0, 1, 2, 3, 4].map((index) => <div key={index} className="h-5 animate-pulse rounded bg-muted" />)}
          </div>
        </div>
      ) : detail.isError ? (
        <section role="alert" className="space-y-4 rounded-xl border p-8">
          <h1 className="text-xl font-semibold">题目加载失败</h1>
          <p className="break-words text-destructive">{detail.error.message}</p>
          <Button variant="outline" disabled={detail.isFetching} onClick={() => void detail.refetch()}>
            {detail.isFetching ? '正在重试…' : '重新加载'}
          </Button>
        </section>
      ) : detail.data ? (
        <div className="grid items-start gap-8 lg:grid-cols-[16rem_minmax(0,1fr)]">
          <aside aria-label="题库题目导航" className="min-w-0 space-y-4 rounded-xl border p-5 lg:sticky lg:top-6">
            <h2 className="font-semibold">题库题目</h2>
            {!bankId ? (
              <p className="text-sm text-muted-foreground">该题目暂无所属题库。</p>
            ) : bankQuestions.isPending ? (
              <div role="status" aria-label="正在加载题库题目" className="space-y-3">
                {[0, 1, 2, 3].map((index) => <div key={index} aria-hidden="true" className="h-5 animate-pulse rounded bg-muted" />)}
              </div>
            ) : bankQuestions.isError ? (
              <div role="alert" className="space-y-3">
                <p className="break-words text-sm text-destructive">{bankQuestions.error.message}</p>
                <Button size="sm" variant="outline" disabled={bankQuestions.isFetching} onClick={() => void bankQuestions.refetch()}>
                  重试导航列表
                </Button>
              </div>
            ) : !bankQuestions.data?.list.length ? (
              <p className="text-sm text-muted-foreground">暂无题目。</p>
            ) : (
              <>
                <nav aria-label="题库内题目" className="max-h-96 space-y-1 overflow-y-auto">
                  {bankQuestions.data.list.map((question) => (
                    <Link
                      key={question.question_id}
                      to={`/questions/${encodeURIComponent(question.question_id)}${navigationSearch}`}
                      aria-current={question.question_id === questionId ? 'page' : undefined}
                      className={`block break-words rounded-md px-3 py-2 text-sm hover:bg-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring ${question.question_id === questionId ? 'bg-accent font-medium' : 'text-muted-foreground'}`}
                    >
                      {question.title}
                    </Link>
                  ))}
                </nav>
                {bankQuestions.data.total > bankQuestions.data.list.length && (
                  <Link to={`/questions${navigationSearch}`} className="block text-sm text-muted-foreground hover:underline">
                    查看全部 {bankQuestions.data.total} 道题目
                  </Link>
                )}
              </>
            )}
          </aside>

          <article className="min-w-0 space-y-8 rounded-xl border bg-card p-6 sm:p-8">
            <header className="space-y-4 border-b pb-6">
              <h1 className="break-words text-3xl font-semibold tracking-tight">{detail.data.title}</h1>
              <div className="flex flex-wrap gap-2">
                <span className="rounded-md bg-secondary px-3 py-1 text-sm font-medium">
                  {difficultyLabels[detail.data.difficulty]}
                </span>
                {detail.data.tags.map((tag) => (
                  <span key={tag} className="break-words rounded-md border px-3 py-1 text-sm text-muted-foreground">{tag}</span>
                ))}
              </div>
            </header>
            <DetailContent content={detail.data.content} />
            <Comments targetType={TargetType.QUESTION} targetId={questionId} />
          </article>
        </div>
      ) : null}
    </main>
  )
}
