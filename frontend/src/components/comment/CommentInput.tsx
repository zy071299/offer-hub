import { useId, useState, type FormEvent } from 'react'

import { useCommentThread } from '@/components/comment/comment-context'
import { Button } from '@/components/ui/button'
import { useAddComment } from '@/hooks/useCommentQueries'
import { useLogin } from '@/hooks/useLogin'

export default function CommentInput() {
  const { targetType, targetId } = useCommentThread()
  const { isLoggedIn, setShowLoginDialog } = useLogin()
  const [content, setContent] = useState('')
  const addComment = useAddComment()
  const inputId = useId()

  async function submit(event: FormEvent) {
    event.preventDefault()
    if (!isLoggedIn) { setShowLoginDialog(true); return }
    if (!content.trim() || addComment.isPending) return
    try {
      await addComment.mutateAsync({ target_type: targetType, target_id: targetId, content: content.trim() })
      setContent('')
    } catch { /* 保留草稿，错误由 mutation 状态展示。 */ }
  }

  return (
    <form aria-label="发表新评论" onSubmit={(event) => void submit(event)} className="space-y-3">
      <label htmlFor={inputId} className="sr-only">发表评论</label>
      <textarea
        id={inputId}
        aria-label="发表评论"
        value={content}
        onChange={(event) => setContent(event.target.value)}
        readOnly={!isLoggedIn}
        onFocus={() => { if (!isLoggedIn) setShowLoginDialog(true) }}
        disabled={addComment.isPending}
        rows={3}
        placeholder={isLoggedIn ? '分享你的理解，参与讨论…' : '登录后参与讨论'}
        className="w-full resize-y rounded-lg border bg-background p-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50"
      />
      {addComment.isError && <p role="alert" className="text-sm text-destructive">{addComment.error.message}</p>}
      <div className="flex justify-end">
        <Button type="submit" disabled={addComment.isPending || (isLoggedIn && !content.trim())}>
          {addComment.isPending ? '正在发表…' : '发表评论'}
        </Button>
      </div>
    </form>
  )
}
