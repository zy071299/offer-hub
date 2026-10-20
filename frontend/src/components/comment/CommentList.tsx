import CommentItem from '@/components/comment/CommentItem'
import { useCommentThread } from '@/components/comment/comment-context'
import { Button } from '@/components/ui/button'
import { useInfiniteComments } from '@/hooks/useCommentQueries'
import { useLogin } from '@/hooks/useLogin'

export default function CommentList() {
  const { targetType, targetId } = useCommentThread()
  const { userInfo } = useLogin()
  const query = useInfiniteComments({ target_type: targetType, target_id: targetId, page_size: 20, sort_by: 'create_time', sort_order: 'desc' }, true, userInfo?.user_id)
  const comments = [...new Map(query.data?.pages.flatMap((page) => page.list).map((comment) => [comment.comment_id, comment])).values()]

  if (query.isPending) return <p role="status" className="animate-pulse py-6 text-sm text-muted-foreground">正在加载评论…</p>

  return (
    <div className="space-y-5">
      <p className="text-sm text-muted-foreground">共 {query.data?.pages[0]?.total ?? 0} 条评论</p>
      {comments.map((comment) => <CommentItem key={comment.comment_id} comment={comment} />)}
      {!query.isError && !comments.length && <p className="py-6 text-center text-sm text-muted-foreground">还没有评论，来分享你的想法吧。</p>}
      {query.isError && (
        <div role="alert" className="space-y-2 text-sm">
          <p className="text-destructive">评论加载失败：{query.error.message}</p>
          <Button variant="outline" size="sm" disabled={query.isFetching} onClick={() => void (query.isFetchNextPageError ? query.fetchNextPage() : query.refetch())}>重试评论</Button>
        </div>
      )}
      {query.hasNextPage && !query.isFetchNextPageError && (
        <div className="flex justify-center">
          <Button variant="outline" disabled={query.isFetching} onClick={() => void query.fetchNextPage()}>
            {query.isFetchingNextPage ? '正在加载…' : '加载更多'}
          </Button>
        </div>
      )}
    </div>
  )
}
