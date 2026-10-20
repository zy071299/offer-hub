import { useParams } from 'react-router-dom'

export function QuestionDetailPage() {
  const { id } = useParams<{ id: string }>()

  return (
    <main className="mx-auto max-w-5xl space-y-4 px-6 py-12">
      <h1 className="text-3xl font-semibold">题目详情</h1>
      <p className="text-muted-foreground">题目 ID：{id}</p>
      <p className="text-muted-foreground">题目详情页面正在建设中。</p>
    </main>
  )
}
