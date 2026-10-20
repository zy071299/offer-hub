import { useState } from 'react'

import CommentInput from '@/components/comment/CommentInput'
import CommentList from '@/components/comment/CommentList'
import { CommentThreadContext, type CommentsProps } from '@/components/comment/comment-context'
import { useLogin } from '@/hooks/useLogin'
import type { CommentInfo } from '@/types/comment'

function CommentThread({ targetType, targetId }: CommentsProps) {
  const [replyingTo, setReplyingTo] = useState<string | null>(null)
  const [editingComment, setEditingComment] = useState<CommentInfo | null>(null)

  return (
    <CommentThreadContext.Provider value={{ targetType, targetId, replyingTo, setReplyingTo, editingComment, setEditingComment }}>
      <section aria-label="评论区" className="space-y-6 border-t pt-8">
        <h2 className="text-xl font-semibold">评论与讨论</h2>
        <CommentInput />
        <CommentList />
      </section>
    </CommentThreadContext.Provider>
  )
}

export default function Comments(props: CommentsProps) {
  const { userInfo } = useLogin()
  // 切换题目或登录用户时重置草稿、回复框和编辑状态。
  return <CommentThread key={JSON.stringify([props.targetType, props.targetId, userInfo?.user_id])} {...props} />
}
