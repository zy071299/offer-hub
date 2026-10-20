/** 接口文档（一）的通用响应结构。 */
export interface AuthApiResponse<T> {
  code: number
  msg: string
  data: T
}

export interface PasswordAuthParams {
  username: string
  password: string
}

export interface PasswordAuthFormValues extends PasswordAuthParams {
  confirmPassword: string
}

/** 密码登录接口返回的 camelCase DTO。 */
export interface PasswordAuthUserDTO {
  userId: string
  username: string
  nickName: string
  avatar: string
  sex: number
  vip: boolean
  phone: string
  email: string
  userStatus: number
  userType: number
}

export interface AuthLoginData {
  token: string
  userInfo: PasswordAuthUserDTO
}

/** 用户信息接口及后续 Context 使用的 snake_case 用户模型。 */
export interface UserInfo {
  user_id: string
  username: string
  nick_name: string
  avatar: string
  vip: boolean
  sex: number
  phone: string
  email: string
  introduction: string
  avatar_url: string
  user_status: number
  user_type: number
  create_time: string
  update_time: string
}

export interface AuthLogoutData {
  message: string
}
