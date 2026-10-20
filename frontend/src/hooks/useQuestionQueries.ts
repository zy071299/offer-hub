import { useQuery, type UseQueryResult } from '@tanstack/react-query'

import {
  getAllQuestionList,
  getHotQuestionList,
  getQuestionDetail,
  getQuestionList,
  getQuestionMetaList,
} from '@/services/question'
import type {
  GetAllQuestionListParams,
  GetQuestionDetailParams,
  GetQuestionMetaListParams,
  HotQuestionListData,
  Question,
  QuestionBankGroup,
  QuestionFilters,
  QuestionListData,
  QuestionMetaListData,
} from '@/types/question'

/** 前缀 key 可用于批量失效，具体 key 包含影响请求结果的全部参数。 */
export const queryKeys = {
  all: ['questions'] as const,
  banks: (jobName?: string) => [...queryKeys.all, 'banks', { job_name: jobName }] as const,
  lists: () => [...queryKeys.all, 'list'] as const,
  list: (filters: QuestionFilters) => [...queryKeys.lists(), filters] as const,
  details: () => [...queryKeys.all, 'detail'] as const,
  detail: (questionId: string) => [...queryKeys.details(), questionId] as const,
  bankLists: () => [...queryKeys.all, 'bank-list'] as const,
  bankList: (bankId: string, keyword?: string) =>
    [...queryKeys.bankLists(), { bank_id: bankId, keyword }] as const,
  hot: () => [...queryKeys.all, 'hot'] as const,
}

export function useQuestionBanks(jobName?: string): UseQueryResult<QuestionBankGroup[], Error> {
  const params: GetAllQuestionListParams = { job_name: jobName }
  return useQuery({
    queryKey: queryKeys.banks(jobName),
    queryFn: () => getAllQuestionList(params),
  })
}

export function useQuestionList(
  filters: QuestionFilters,
  enabled: boolean = true,
): UseQueryResult<QuestionListData, Error> {
  return useQuery({
    queryKey: queryKeys.list(filters),
    queryFn: () => getQuestionList(filters),
    enabled,
  })
}

export function useQuestionDetail(
  questionId: string,
  enabled: boolean = true,
): UseQueryResult<Question, Error> {
  const params: GetQuestionDetailParams = { question_id: questionId }
  return useQuery({
    queryKey: queryKeys.detail(questionId),
    queryFn: () => getQuestionDetail(params),
    enabled: enabled && questionId.trim().length > 0,
  })
}

/** 题库导航使用元数据列表；分页沿用接口默认值（第 1 页，每页 20 条）。 */
export function useBankQuestionList(
  bankId: string,
  keyword?: string,
  enabled: boolean = true,
): UseQueryResult<QuestionMetaListData, Error> {
  const params: GetQuestionMetaListParams = { bank_id: bankId, keyword }
  return useQuery({
    queryKey: queryKeys.bankList(bankId, keyword),
    queryFn: () => getQuestionMetaList(params),
    enabled: enabled && bankId.trim().length > 0,
  })
}

export function useHotQuestions(): UseQueryResult<HotQuestionListData, Error> {
  return useQuery({
    queryKey: queryKeys.hot(),
    queryFn: () => getHotQuestionList(),
  })
}
