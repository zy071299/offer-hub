import { createContext, useContext } from 'react'
import type { Dispatch, SetStateAction } from 'react'

import type { CommentInfo, GetCommentListParams } from '@/types/comment'

export interface CommentsProps {
  targetType: GetCommentListParams['target_type']
  targetId: string
}

interface CommentThreadContextValue extends CommentsProps {
  replyingTo: string | null
  setReplyingTo: Dispatch<SetStateAction<string | null>>
  editingComment: CommentInfo | null
  setEditingComment: Dispatch<SetStateAction<CommentInfo | null>>
}

export const CommentThreadContext = createContext<CommentThreadContextValue | null>(null)

export function useCommentThread() {
  const context = useContext(CommentThreadContext)
  if (!context) throw new Error('评论组件必须在 Comments 内使用')
  return context
}
