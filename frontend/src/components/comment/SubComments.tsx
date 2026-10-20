import CommentItem from '@/components/comment/CommentItem'
import { useCommentThread } from '@/components/comment/comment-context'
import { Button } from '@/components/ui/button'
import { useInfiniteComments } from '@/hooks/useCommentQueries'
import { useLogin } from '@/hooks/useLogin'

interface SubCommentsProps {
  parentId: string
}

/** 仅在展开后挂载，回复分页以 parent_id 为条件，每页 5 条。 */
export default function SubComments({ parentId }: SubCommentsProps) {
  const { targetType, targetId } = useCommentThread()
  const { userInfo } = useLogin()
  const query = useInfiniteComments({ target_type: targetType, target_id: targetId, parent_id: parentId, page_size: 5, sort_by: 'create_time', sort_order: 'asc' }, true, userInfo?.user_id)
  const comments = [...new Map(query.data?.pages.flatMap((page) => page.list).map((comment) => [comment.comment_id, comment])).values()]

  return (
    <div className="mt-4 space-y-4 border-l-2 pl-3 sm:pl-5">
      {query.isPending && <p role="status" className="text-sm text-muted-foreground">正在加载回复…</p>}
      {comments.map((comment) => <CommentItem key={comment.comment_id} comment={comment} isSub />)}
      {!query.isPending && !query.isError && !comments.length && <p className="text-sm text-muted-foreground">暂无回复。</p>}
      {query.isError && (
        <div role="alert" className="space-y-2 text-sm">
          <p className="text-destructive">回复加载失败：{query.error.message}</p>
          <Button size="sm" variant="outline" disabled={query.isFetching} onClick={() => void (query.isFetchNextPageError ? query.fetchNextPage() : query.refetch())}>重试回复</Button>
        </div>
      )}
      {query.hasNextPage && !query.isFetchNextPageError && (
        <Button size="sm" variant="ghost" disabled={query.isFetching} onClick={() => void query.fetchNextPage()}>
          {query.isFetchingNextPage ? '正在加载…' : '查看更多回复'}
        </Button>
      )}
    </div>
  )
}
