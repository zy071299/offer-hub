import { useCallback, useEffect, useMemo, useRef, useState, type PropsWithChildren } from 'react'

import { LoginContext } from '@/components/provider/LoginContext'
import { queryKeys } from '@/hooks/useQuestionQueries'
import { queryClient } from '@/lib/query-client'
import { getUserInfo, mapPasswordAuthUserToUserInfo } from '@/services/auth'
import type { PasswordAuthUserDTO, UserInfo } from '@/types/auth'

export function LoginProvider({ children }: PropsWithChildren) {
  const [userInfo, setUserInfo] = useState<UserInfo | null>(null)
  const [showLoginDialog, setShowLoginDialog] = useState(false)
  // 登录、登出及新请求都会使旧请求失效，避免迟到的响应覆盖当前用户。
  const requestVersion = useRef(0)
  const invalidateRequests = useCallback(() => { requestVersion.current++ }, [])

  const login = useCallback((user: PasswordAuthUserDTO | UserInfo, token: string) => {
    const normalizedUser = 'userId' in user ? mapPasswordAuthUserToUserInfo(user) : user
    localStorage.setItem('token', token)
    requestVersion.current++
    setUserInfo(normalizedUser)
    setShowLoginDialog(false)
    void queryClient.invalidateQueries({ queryKey: queryKeys.details() })
  }, [])

  const logout = useCallback(() => {
    requestVersion.current++
    localStorage.removeItem('token')
    setUserInfo(null)
    setShowLoginDialog(false)
  }, [])

  const refreshUserInfo = useCallback(async () => {
    const version = ++requestVersion.current
    const token = localStorage.getItem('token')
    if (!token) {
      setUserInfo(null)
      return
    }
    try {
      const user = await getUserInfo()
      if (version === requestVersion.current && localStorage.getItem('token') === token) {
        setUserInfo(user)
      }
    } catch (error) {
      if (version === requestVersion.current && localStorage.getItem('token') === token) {
        localStorage.removeItem('token')
        setUserInfo(null)
      } else if (version === requestVersion.current && !localStorage.getItem('token')) {
        // HTTP 401 时 axios 拦截器可能已经删除 token。
        setUserInfo(null)
      }
      throw error
    }
  }, [])

  useEffect(() => {
    void refreshUserInfo().catch(() => {
      // 初始化失败已清理状态；主动刷新仍通过 Promise 向调用方报告错误。
    })
    return invalidateRequests
  }, [refreshUserInfo, invalidateRequests])

  const value = useMemo(() => ({
    isLoggedIn: userInfo !== null,
    userInfo,
    login,
    logout,
    showLoginDialog,
    setShowLoginDialog,
    refreshUserInfo,
  }), [userInfo, login, logout, showLoginDialog, refreshUserInfo])

  return <LoginContext.Provider value={value}>{children}</LoginContext.Provider>
}
