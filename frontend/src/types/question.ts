/** 通用业务响应：code 为 0 表示成功，失败时 data 为 null。 */
export interface QuestionApiResponse<T> {
  code: number
  msg: string
  data: T | null
}

export interface GetAllQuestionListParams {
  job_name?: string
}

export interface GetQuestionListParams {
  bank_id?: string
  keyword?: string
  difficulty?: 1 | 2 | 3
  tags?: string[]
  job_name?: string
  /** 仅登录用户有效。 */
  user_tag?: 1 | 2 | 3
  sort_by?: 'create_time' | 'view_count' | 'thumbs_up_count' | 'dislike_count'
  sort_order?: 'asc' | 'desc'
  /** 默认 1。 */
  page?: number
  /** 默认 20。 */
  page_size?: number
}

export interface GetQuestionMetaListParams extends GetQuestionListParams {
  /** 与题目列表共用过滤条件，默认页码为 1。 */
  page?: number
}

/** 页面筛选条件，与题目列表请求参数保持一致。 */
export interface QuestionFilters extends GetQuestionListParams {
  bank_id?: string
}

export interface GetQuestionDetailParams {
  question_id: string
}

export interface GetHotQuestionListParams {
  /** 默认 10。 */
  limit?: number
  job_name?: string
}

export interface QuestionBank {
  bank_id: string
  bank_name: string
  bank_logo: string
  desc: string
  count: number
  order: number
}

export interface QuestionBankSeries {
  series_id: string
  series_name: string
  order: number
  bank_list: QuestionBank[]
}

export interface QuestionBankGroup {
  job_name: string
  series_list: QuestionBankSeries[]
}

export interface QuestionMeta {
  question_id: string
  title: string
}

/** 列表与详情共用；游客内容由服务端截断，用户字段返回零值。 */
export interface Question extends QuestionMeta {
  bank_list: string[]
  content: string
  difficulty: 1 | 2 | 3
  tags: string[]
  /** 1 正常、2 隐藏、3 删除。 */
  status: 1 | 2 | 3
  vip: boolean
  hot_degree: number
  view_count: number
  thumbs_up_count: number
  dislike_count: number
  order: number
  /** 0 未标记、1 已掌握、2 晚点再刷、3 未掌握。 */
  user_tag: 0 | 1 | 2 | 3
  user_liked: boolean
  /** YYYY-MM-DD HH:mm:ss */
  create_time: string
  /** YYYY-MM-DD HH:mm:ss */
  update_time: string
}

export interface QuestionListData {
  total: number
  list: Question[]
}

export interface QuestionMetaListData {
  total: number
  list: QuestionMeta[]
}

export interface HotQuestion extends QuestionMeta {
  bank_list: string[]
  view_count: number
}

export interface HotQuestionListData {
  list: HotQuestion[]
}

export interface GetAllQuestionListResponse extends QuestionApiResponse<QuestionBankGroup[]> {
  data: QuestionBankGroup[] | null
}
export interface GetQuestionListResponse extends QuestionApiResponse<QuestionListData> {
  data: QuestionListData | null
}
export interface GetQuestionMetaListResponse extends QuestionApiResponse<QuestionMetaListData> {
  data: QuestionMetaListData | null
}
export interface GetQuestionDetailResponse extends QuestionApiResponse<Question> {
  data: Question | null
}
export interface GetHotQuestionListResponse extends QuestionApiResponse<HotQuestionListData> {
  data: HotQuestionListData | null
}

/** 文档中的标记接口类型；本次不封装 POST 请求。 */
export interface TagQuestionParams {
  question_id: string
  tag: 0 | 1 | 2 | 3
}

export interface TagQuestionResponse extends QuestionApiResponse<null> {
  data: null
}
