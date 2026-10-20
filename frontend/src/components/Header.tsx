import { useRef, useState } from 'react'
import * as DropdownMenu from '@radix-ui/react-dropdown-menu'
import { BookOpen, ChevronDown } from 'lucide-react'
import { Link } from 'react-router-dom'

import { Button } from '@/components/ui/button'
import { useLogin } from '@/hooks/useLogin'
import { postAuthLogout } from '@/services/auth'

function UserAvatar({ src, name }: { src: string; name: string }) {
  const [failed, setFailed] = useState(false)

  return (
    <span className="flex size-9 shrink-0 items-center justify-center overflow-hidden rounded-full bg-emerald-100 text-sm font-semibold text-emerald-800">
      {src && !failed ? (
        <img src={src} alt={`${name}的头像`} onError={() => setFailed(true)} className="size-full object-cover" />
      ) : (
        <span aria-hidden="true">{Array.from(name)[0] || '用'}</span>
      )}
    </span>
  )
}

export default function Header() {
  const { isLoggedIn, userInfo, setShowLoginDialog, logout } = useLogin()
  const [isLoggingOut, setIsLoggingOut] = useState(false)
  const [logoutError, setLogoutError] = useState('')
  const logoutPending = useRef(false)
  const displayName = userInfo?.nick_name?.trim() || userInfo?.username || '用户'
  const avatar = userInfo?.avatar_url || userInfo?.avatar || ''

  async function handleLogout() {
    if (logoutPending.current) return
    logoutPending.current = true
    setIsLoggingOut(true)
    setLogoutError('')
    try {
      // 必须保留 token，直到请求发出并完成，才能让拦截器附加 Authorization。
      await postAuthLogout()
    } catch {
      setLogoutError('已退出当前设备，服务器暂未确认退出。')
    } finally {
      logout()
      logoutPending.current = false
      setIsLoggingOut(false)
    }
  }

  return (
    <header className="border-b bg-background">
      <div className="mx-auto flex h-16 max-w-7xl items-center justify-between gap-4 px-6">
        <Link to="/" className="flex shrink-0 items-center gap-2 rounded-sm text-lg font-semibold tracking-tight focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">
          <BookOpen className="size-6 text-emerald-700" aria-hidden="true" />
          Offer Hub
        </Link>
        <div className="flex min-w-0 items-center gap-4">
          <Link to="/collections" className="shrink-0 rounded-sm text-sm text-muted-foreground hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">
            题库
          </Link>
          {!isLoggedIn ? (
            <Button onClick={() => { setLogoutError(''); setShowLoginDialog(true) }}>
              登录
            </Button>
          ) : (
            <DropdownMenu.Root>
              <DropdownMenu.Trigger asChild>
                <button
                  type="button"
                  aria-label={`${displayName}，打开用户菜单`}
                  aria-busy={isLoggingOut}
                  disabled={isLoggingOut}
                  className="flex min-w-0 items-center gap-2 rounded-lg p-1 text-sm hover:bg-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-60"
                >
                  <UserAvatar key={avatar} src={avatar} name={displayName} />
                  <span className="max-w-24 truncate sm:max-w-40">{isLoggingOut ? '退出中…' : displayName}</span>
                  <ChevronDown className="size-4 shrink-0 text-muted-foreground" aria-hidden="true" />
                </button>
              </DropdownMenu.Trigger>
              <DropdownMenu.Portal>
                <DropdownMenu.Content
                  align="end"
                  sideOffset={8}
                  collisionPadding={12}
                  className="z-50 min-w-40 rounded-lg border bg-popover p-1 text-popover-foreground shadow-md"
                >
                  <DropdownMenu.Item
                    disabled={isLoggingOut}
                    onSelect={handleLogout}
                    className="cursor-pointer rounded-md px-3 py-2 text-sm outline-none focus:bg-accent data-[disabled]:pointer-events-none data-[disabled]:opacity-50"
                  >
                    退出登录
                  </DropdownMenu.Item>
                </DropdownMenu.Content>
              </DropdownMenu.Portal>
            </DropdownMenu.Root>
          )}
        </div>
      </div>
      {logoutError && <p role="status" className="mx-auto max-w-7xl px-6 pb-3 text-sm text-destructive">{logoutError}</p>}
    </header>
  )
}
