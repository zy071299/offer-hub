import { apiClient } from '@/lib/axios'
import type {
  AuthApiResponse,
  AuthLoginData,
  AuthLogoutData,
  PasswordAuthParams,
  PasswordAuthUserDTO,
  UserInfo,
} from '@/types/auth'

function unwrapResponse<T>(response: AuthApiResponse<T>): T {
  if (response.code !== 0) {
    throw new Error(response.msg)
  }
  return response.data
}

/** 注册成功时返回文档约定的 null。 */
export async function postAuthRegister(username: string, password: string): Promise<null> {
  const params: PasswordAuthParams = { username, password }
  const { data } = await apiClient.post<AuthApiResponse<null>>('/auth/register', params)
  return unwrapResponse(data)
}

/** 返回原始登录数据；token 保存和 Context 更新由调用方负责。 */
export async function postAuthLogin(username: string, password: string): Promise<AuthLoginData> {
  const params: PasswordAuthParams = { username, password }
  const { data } = await apiClient.post<AuthApiResponse<AuthLoginData>>('/auth/login', params)
  return unwrapResponse(data)
}

/** Authorization 由 axios 请求拦截器从 localStorage 读取并添加。 */
export async function postAuthLogout(): Promise<AuthLogoutData> {
  const { data } = await apiClient.post<AuthApiResponse<AuthLogoutData>>('/auth/logout')
  return unwrapResponse(data)
}

export async function getUserInfo(): Promise<UserInfo> {
  const { data } = await apiClient.get<AuthApiResponse<UserInfo>>('/api/v1/user_info/get_user_info')
  return unwrapResponse(data)
}

/** 登录未提供简介和时间字段，先留空，后续可通过 getUserInfo 补全。 */
export function mapPasswordAuthUserToUserInfo(user: PasswordAuthUserDTO): UserInfo {
  return {
    user_id: user.userId,
    username: user.username,
    nick_name: user.nickName,
    avatar: user.avatar,
    vip: user.vip,
    sex: user.sex,
    phone: user.phone,
    email: user.email,
    introduction: '',
    avatar_url: user.avatar,
    user_status: user.userStatus,
    user_type: user.userType,
    create_time: '',
    update_time: '',
  }
}
