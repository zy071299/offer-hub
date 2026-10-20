import { useId, useState, type FormEvent } from 'react'

import { useCommentThread } from '@/components/comment/comment-context'
import { Button } from '@/components/ui/button'
import { useAddComment, useUpdateComment } from '@/hooks/useCommentQueries'
import { useLogin } from '@/hooks/useLogin'
import type { CommentInfo } from '@/types/comment'

interface ReplyBoxEnhancedProps {
  comment: CommentInfo
  mode?: 'reply' | 'edit'
}

export default function ReplyBoxEnhanced({ comment, mode = 'reply' }: ReplyBoxEnhancedProps) {
  const { targetType, targetId, setReplyingTo, setEditingComment } = useCommentThread()
  const { isLoggedIn, setShowLoginDialog } = useLogin()
  const [content, setContent] = useState(mode === 'edit' ? comment.content : '')
  const addComment = useAddComment()
  const updateComment = useUpdateComment()
  const mutation = mode === 'edit' ? updateComment : addComment
  const inputId = useId()

  function close() {
    if (mode === 'edit') setEditingComment(null)
    else setReplyingTo(null)
  }

  async function submit(event: FormEvent) {
    event.preventDefault()
    if (!isLoggedIn) { setShowLoginDialog(true); return }
    if (!content.trim() || mutation.isPending) return
    try {
      if (mode === 'edit') {
        await updateComment.mutateAsync({ comment_id: comment.comment_id, content: content.trim() })
      } else {
        await addComment.mutateAsync({
          target_type: targetType, target_id: targetId,
          parent_id: comment.parent_id || comment.comment_id,
          reply_to: comment.user_id, content: content.trim(),
        })
      }
      close()
    } catch { /* 请求失败保留输入，允许重试。 */ }
  }

  return (
    <form aria-label={mode === 'edit' ? '编辑评论' : '回复评论'} onSubmit={(event) => void submit(event)} className="mt-4 space-y-3 rounded-lg bg-muted/40 p-3">
      <label htmlFor={inputId} className="block text-sm text-muted-foreground">
        {mode === 'edit' ? '编辑评论' : `回复 @${comment.user_name || '用户'}`}
      </label>
      <textarea id={inputId} value={content} onChange={(event) => setContent(event.target.value)} disabled={mutation.isPending} rows={3} autoFocus className="w-full resize-y rounded-md border bg-background p-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring" />
      {mutation.isError && <p role="alert" className="text-sm text-destructive">{mutation.error.message}</p>}
      <div className="flex justify-end gap-2">
        <Button type="button" variant="ghost" size="sm" onClick={close} disabled={mutation.isPending}>取消</Button>
        <Button type="submit" size="sm" disabled={mutation.isPending || !content.trim()}>
          {mutation.isPending ? '正在提交…' : mode === 'edit' ? '保存修改' : '提交回复'}
        </Button>
      </div>
    </form>
  )
}
