import { apiClient } from '@/lib/axios'
import type {
  CommentApiResponse,
  CommentAddData,
  CommentListData,
  CommentUpdateData,
  GetCommentListParams,
  GetCommentListResponse,
  PostCommentAddParams,
  PostCommentAddResponse,
  PostCommentDeleteParams,
  PostCommentDeleteResponse,
  PostCommentUpdateParams,
  PostCommentUpdateResponse,
  ThumbsUpCommentParams,
  ThumbsUpCommentResponse,
  CommentLikeData,
  CommentLikeRequest,
} from '@/types/comment'

function unwrapResponse<T>(response: CommentApiResponse<T>): T {
  if (response.code !== 0) {
    throw new Error(response.msg)
  }
  if (response.data === null) {
    throw new Error('评论接口返回数据为空')
  }
  return response.data
}

/** 游客可访问；parent_id 非空时获取对应父评论的回复列表。 */
export async function getCommentList(params: GetCommentListParams): Promise<CommentListData> {
  const { data } = await apiClient.get<GetCommentListResponse>('/api/v1/open/list_comments', { params })
  return unwrapResponse(data)
}

/** 以下写入接口的 Authorization 由 axios 拦截器自动添加。 */
export async function postCommentAdd(params: PostCommentAddParams): Promise<CommentAddData> {
  const { data } = await apiClient.post<PostCommentAddResponse>('/api/v1/comment/add', params)
  return unwrapResponse(data)
}

export async function postCommentDelete(params: PostCommentDeleteParams): Promise<void> {
  const { data } = await apiClient.post<PostCommentDeleteResponse>('/api/v1/comment/delete', params)
  if (data.code !== 0) {
    throw new Error(data.msg)
  }
}

export async function postCommentUpdate(params: PostCommentUpdateParams): Promise<CommentUpdateData> {
  const { data } = await apiClient.post<PostCommentUpdateResponse>('/api/v1/comment/update', params)
  return unwrapResponse(data)
}

export async function thumbsUpComment(params: ThumbsUpCommentParams): Promise<CommentLikeData> {
  const body: CommentLikeRequest = { target_type: 3, target_id: params.comment_id }
  const { data } = await apiClient.post<ThumbsUpCommentResponse>('/api/v1/interaction/like', body)
  return unwrapResponse(data)
}
