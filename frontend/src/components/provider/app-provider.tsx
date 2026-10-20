import type { PropsWithChildren } from 'react'
import { QueryClientProvider } from '@tanstack/react-query'

import { queryClient } from '@/lib/query-client'
import { LoginProvider } from '@/components/provider/LoginProvider'
import AuthLoginDialog from '@/components/AuthLoginDialog'

export function AppProvider({ children }: PropsWithChildren) {
  return (
    <QueryClientProvider client={queryClient}>
      <LoginProvider>
        {children}
        <AuthLoginDialog />
      </LoginProvider>
    </QueryClientProvider>
  )
}
