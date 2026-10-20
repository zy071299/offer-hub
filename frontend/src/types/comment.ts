/** 评论目标类型，与后端接口约定一致。 */
export enum TargetType {
  QUESTION = 1,
  EXPERIENCE = 2,
}

/** 评论接口通用响应；业务失败时 data 可能为 null。 */
export interface CommentApiResponse<T> {
  code: number
  msg: string
  data: T | null
}

export interface GetCommentListParams {
  /** 1：题目；2：面经。 */
  target_type: 1 | 2
  target_id: string
  parent_id?: string
  sort_by?: 'create_time' | 'view_count' | 'thumbs_up'
  sort_order?: 'asc' | 'desc'
  /** 默认 1。 */
  page?: number
  /** 默认 20。 */
  page_size?: number
  /** 默认 1。 */
  sub_comment_page?: number
  /** 默认 5。 */
  sub_comment_size?: number
}

export interface CommentInfo {
  comment_id: string
  user_id: string
  user_name: string
  user_avatar: string
  content: string
  parent_id: string
  reply_to: string
  reply_to_name: string
  status: number
  thumbs_up: number
  sub_comment_total: number
  user_liked: boolean
  sub_comments: CommentInfo[]
  create_time: string
  update_time: string
}

export interface CommentListData {
  total: number
  list: CommentInfo[]
}

export interface PostCommentAddParams {
  target_type: 1 | 2
  target_id: string
  parent_id?: string
  reply_to?: string
  content: string
}

export interface CommentAddData {
  comment_id: string
  comment: CommentInfo
}

export interface PostCommentDeleteParams {
  comment_id: string
}

export interface PostCommentUpdateParams {
  comment_id: string
  content: string
}

export interface CommentUpdateData {
  comment_id: string
}

export interface GetCommentListResponse extends CommentApiResponse<CommentListData> {
  data: CommentListData | null
}

export interface PostCommentAddResponse extends CommentApiResponse<CommentAddData> {
  data: CommentAddData | null
}

export interface PostCommentUpdateResponse extends CommentApiResponse<CommentUpdateData> {
  data: CommentUpdateData | null
}

/** 删除接口按文档不返回 data 字段。 */
export interface PostCommentDeleteResponse {
  code: number
  msg: string
}

export interface ThumbsUpCommentParams {
  comment_id: string
}

export interface CommentLikeData {
  liked: boolean
  count: number
}

export interface ThumbsUpCommentResponse extends CommentApiResponse<CommentLikeData> {
  data: CommentLikeData | null
}

export interface CommentLikeRequest {
  target_type: 3
  target_id: string
}
