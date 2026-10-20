import { useState } from 'react'
import { ThumbsUp } from 'lucide-react'

import ReplyBoxEnhanced from '@/components/comment/ReplyBoxEnhanced'
import SubComments from '@/components/comment/SubComments'
import { useCommentThread } from '@/components/comment/comment-context'
import { Button } from '@/components/ui/button'
import { useDeleteComment, useThumbsUpComment } from '@/hooks/useCommentQueries'
import { useLogin } from '@/hooks/useLogin'
import type { CommentInfo } from '@/types/comment'

interface CommentItemProps {
  comment: CommentInfo
  isSub?: boolean
}

function CommentAvatar({ src, name }: { src: string; name: string }) {
  const [failed, setFailed] = useState(false)
  return src && !failed
    ? <img src={src} alt={`${name}的头像`} onError={() => setFailed(true)} className="size-9 shrink-0 rounded-full object-cover" />
    : <span aria-hidden="true" className="flex size-9 shrink-0 items-center justify-center rounded-full bg-secondary text-sm font-medium">{Array.from(name)[0]}</span>
}

export default function CommentItem({ comment, isSub = false }: CommentItemProps) {
  const { replyingTo, setReplyingTo, editingComment, setEditingComment } = useCommentThread()
  const { isLoggedIn, userInfo, setShowLoginDialog } = useLogin()
  const [expanded, setExpanded] = useState(false)
  const deleteComment = useDeleteComment()
  const like = useThumbsUpComment()
  const isOwner = isLoggedIn && userInfo?.user_id === comment.user_id
  const name = comment.user_name || '用户'
  const isEditing = editingComment?.comment_id === comment.comment_id

  function requireLogin() {
    if (!isLoggedIn) { setShowLoginDialog(true); return false }
    return true
  }

  async function remove() {
    if (!requireLogin() || !isOwner || deleteComment.isPending) return
    try {
      await deleteComment.mutateAsync({ comment_id: comment.comment_id })
      if (replyingTo === comment.comment_id) setReplyingTo(null)
      if (isEditing) setEditingComment(null)
    } catch { /* 显示 mutation 错误，不移除评论。 */ }
  }

  async function thumbsUp() {
    if (!requireLogin() || like.isPending || comment.user_liked) return
    try { await like.mutateAsync({ comment_id: comment.comment_id }) }
    catch { /* 点赞失败保留原计数。 */ }
  }

  return (
    <div data-comment-id={comment.comment_id} className={isSub ? 'min-w-0' : 'min-w-0 border-b pb-5 last:border-b-0'}>
      <div className="flex items-start gap-3">
        <CommentAvatar key={comment.user_avatar} src={comment.user_avatar} name={name} />
        <div className="min-w-0 flex-1 space-y-2">
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
            <span className="break-words text-sm font-medium">{name}</span>
            <time className="text-xs text-muted-foreground">{comment.create_time}</time>
          </div>
          {isEditing ? <ReplyBoxEnhanced key={`edit-${comment.comment_id}`} mode="edit" comment={comment} /> : (
            <p className="whitespace-pre-wrap break-words text-sm leading-7">
              {comment.reply_to_name && <span className="text-primary">回复 @{comment.reply_to_name}：</span>}
              {comment.content}
            </p>
          )}
          <div className="flex flex-wrap items-center gap-1">
            <Button type="button" size="sm" variant="ghost" aria-label="点赞评论" aria-pressed={comment.user_liked} disabled={like.isPending || comment.user_liked} onClick={() => void thumbsUp()}>
              <ThumbsUp aria-hidden="true" />{comment.thumbs_up}
            </Button>
            <Button type="button" size="sm" variant="ghost" onClick={() => {
              if (!requireLogin()) return
              setEditingComment(null)
              setReplyingTo(replyingTo === comment.comment_id ? null : comment.comment_id)
            }}>回复</Button>
            {isOwner && <>
              <Button type="button" size="sm" variant="ghost" disabled={deleteComment.isPending} onClick={() => { setReplyingTo(null); setEditingComment(comment) }}>编辑</Button>
              <Button type="button" size="sm" variant="ghost" disabled={deleteComment.isPending} onClick={() => void remove()}>{deleteComment.isPending ? '正在删除…' : '删除'}</Button>
            </>}
          </div>
          {(deleteComment.isError || like.isError) && <p role="alert" className="text-sm text-destructive">{deleteComment.error?.message || like.error?.message}</p>}
          {replyingTo === comment.comment_id && <ReplyBoxEnhanced key={`reply-${comment.comment_id}`} comment={comment} />}
          {!isSub && comment.sub_comment_total > 0 && (
            <Button type="button" size="sm" variant="link" aria-expanded={expanded} onClick={() => setExpanded(!expanded)}>
              {expanded ? '收起回复' : `查看 ${comment.sub_comment_total} 条回复`}
            </Button>
          )}
          {!isSub && expanded && <SubComments parentId={comment.comment_id} />}
        </div>
      </div>
    </div>
  )
}
