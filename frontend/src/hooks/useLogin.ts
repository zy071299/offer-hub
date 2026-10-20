import { useContext } from 'react'

import { LoginContext } from '@/components/provider/LoginContext'

export function useLogin() {
  const context = useContext(LoginContext)
  if (!context) {
    throw new Error('useLogin 必须在 LoginProvider 内使用')
  }
  return context
}
