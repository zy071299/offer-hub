import {
  useMutation,
  useInfiniteQuery,
  useQuery,
  useQueryClient,
  type UseMutationResult,
  type UseQueryResult,
  type InfiniteData,
} from '@tanstack/react-query'

import { getCommentList, postCommentAdd, postCommentDelete, postCommentUpdate, thumbsUpComment } from '@/services/comment'
import type {
  CommentAddData,
  CommentListData,
  CommentUpdateData,
  GetCommentListParams,
  PostCommentAddParams,
  PostCommentDeleteParams,
  PostCommentUpdateParams,
  CommentInfo,
  CommentLikeData,
  ThumbsUpCommentParams,
} from '@/types/comment'

/** 列表 key 包含目标、分页、排序及回复参数，支持按前缀批量失效。 */
export const queryKeys = {
  all: ['comments'] as const,
  lists: () => [...queryKeys.all, 'list'] as const,
  list: (params: GetCommentListParams) => [...queryKeys.lists(), params] as const,
  infinite: (params: Omit<GetCommentListParams, 'page'>, viewerId: string = 'guest') =>
    [...queryKeys.lists(), 'infinite', viewerId, params] as const,
}

/** viewerId 隔离游客和不同用户的 user_liked 缓存。 */
export function useInfiniteComments(
  params: Omit<GetCommentListParams, 'page'>,
  enabled: boolean = true,
  viewerId: string = 'guest',
) {
  return useInfiniteQuery({
    queryKey: queryKeys.infinite(params, viewerId),
    queryFn: ({ pageParam }) => getCommentList({ ...params, page: pageParam }),
    initialPageParam: 1,
    getNextPageParam: (lastPage, pages, lastPageParam) => {
      const loaded = pages.reduce((total, page) => total + page.list.length, 0)
      return lastPage.list.length > 0 && loaded < lastPage.total ? lastPageParam + 1 : undefined
    },
    enabled: enabled && params.target_id.trim().length > 0,
  })
}

export function useThumbsUpComment(): UseMutationResult<CommentLikeData, Error, ThumbsUpCommentParams> {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: thumbsUpComment,
    onSuccess: (result, params) => {
      const updateComments = (comments: CommentInfo[]): CommentInfo[] => comments.map((comment) => ({
        ...comment,
        ...(comment.comment_id === params.comment_id ? { thumbs_up: result.count, user_liked: result.liked } : {}),
        sub_comments: updateComments(comment.sub_comments),
      }))
      queryClient.setQueriesData<CommentListData | InfiniteData<CommentListData>>(
        { queryKey: queryKeys.lists() },
        (cached) => {
          if (!cached) return cached
          if ('pages' in cached) {
            return { ...cached, pages: cached.pages.map((page) => ({ ...page, list: updateComments(page.list) })) }
          }
          return { ...cached, list: updateComments(cached.list) }
        },
      )
    },
  })
}

export function useComments(
  params: GetCommentListParams,
  enabled: boolean = true,
): UseQueryResult<CommentListData, Error> {
  return useQuery({
    queryKey: queryKeys.list(params),
    queryFn: () => getCommentList(params),
    enabled: enabled && params.target_id.trim().length > 0,
  })
}

/** 成功后刷新评论列表；返回失效 Promise，让 mutation 等待活跃列表刷新。 */
export function useAddComment(): UseMutationResult<CommentAddData, Error, PostCommentAddParams> {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: postCommentAdd,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.lists() }),
  })
}

export function useUpdateComment(): UseMutationResult<CommentUpdateData, Error, PostCommentUpdateParams> {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: postCommentUpdate,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.lists() }),
  })
}

export function useDeleteComment(): UseMutationResult<void, Error, PostCommentDeleteParams> {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: postCommentDelete,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.lists() }),
  })
}
