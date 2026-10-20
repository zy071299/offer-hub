import { apiClient } from '@/lib/axios'
import type {
  GetAllQuestionListParams,
  GetAllQuestionListResponse,
  GetHotQuestionListParams,
  GetHotQuestionListResponse,
  GetQuestionDetailParams,
  GetQuestionDetailResponse,
  GetQuestionListParams,
  GetQuestionListResponse,
  GetQuestionMetaListParams,
  GetQuestionMetaListResponse,
  HotQuestionListData,
  Question,
  QuestionApiResponse,
  QuestionBankGroup,
  QuestionListData,
  QuestionMetaListData,
} from '@/types/question'

export class QuestionApiError extends Error {
  readonly code: number

  constructor(code: number, message: string) {
    super(message)
    this.name = 'QuestionApiError'
    this.code = code
  }
}

function unwrapResponse<T>(response: QuestionApiResponse<T>): T {
  if (response.code !== 0) {
    throw new QuestionApiError(response.code, response.msg || '题目接口请求失败')
  }
  if (response.data === null) {
    throw new Error('题目接口返回数据为空')
  }
  return response.data
}

// Go 接口通过重复的 tags 参数绑定数组，而不是 tags[]。
const listParamsSerializer = { indexes: null }

/** 按职位方向分组的系列及题库全量列表。 */
export async function getAllQuestionList(
  params: GetAllQuestionListParams = {},
): Promise<QuestionBankGroup[]> {
  const { data } = await apiClient.get<GetAllQuestionListResponse>(
    '/api/v1/question/all/list', { params },
  )
  return unwrapResponse(data)
}

/** 支持分页、过滤和排序的题目列表。 */
export async function getQuestionList(
  params: GetQuestionListParams = {},
): Promise<QuestionListData> {
  const { data } = await apiClient.get<GetQuestionListResponse>(
    '/api/v1/question/list', { params, paramsSerializer: listParamsSerializer },
  )
  return unwrapResponse(data)
}

/** 与题目列表使用相同过滤条件，只返回 ID 和标题。 */
export async function getQuestionMetaList(
  params: GetQuestionMetaListParams = {},
): Promise<QuestionMetaListData> {
  const { data } = await apiClient.get<GetQuestionMetaListResponse>(
    '/api/v1/question/meta/list', { params, paramsSerializer: listParamsSerializer },
  )
  return unwrapResponse(data)
}

export async function getQuestionDetail(
  params: GetQuestionDetailParams,
): Promise<Question> {
  const { data } = await apiClient.get<GetQuestionDetailResponse>(
    '/api/v1/question/detail', { params },
  )
  return unwrapResponse(data)
}

/** 按热度降序返回热门题目。 */
export async function getHotQuestionList(
  params: GetHotQuestionListParams = {},
): Promise<HotQuestionListData> {
  const { data } = await apiClient.get<GetHotQuestionListResponse>(
    '/api/v1/question/hot/list', { params },
  )
  return unwrapResponse(data)
}
