import { createContext } from 'react'

import type { PasswordAuthUserDTO, UserInfo } from '@/types/auth'

export interface LoginContextValue {
  isLoggedIn: boolean
  userInfo: UserInfo | null
  login: (userInfo: PasswordAuthUserDTO | UserInfo, token: string) => void
  logout: () => void
  showLoginDialog: boolean
  setShowLoginDialog: (show: boolean) => void
  refreshUserInfo: () => Promise<void>
}

export const LoginContext = createContext<LoginContextValue | undefined>(undefined)
