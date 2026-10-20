import { BookOpen, Check } from 'lucide-react'

import UsernamePasswordAuthForms from '@/components/auth/UsernamePasswordAuthForms'
import { Dialog, DialogContent, DialogDescription, DialogTitle } from '@/components/ui/dialog'
import { useLogin } from '@/hooks/useLogin'

export default function AuthLoginDialog() {
  const { showLoginDialog, setShowLoginDialog } = useLogin()

  return (
    <Dialog open={showLoginDialog} onOpenChange={setShowLoginDialog}>
      <DialogContent className="max-h-[90dvh] max-w-4xl gap-0 overflow-y-auto p-0 md:grid-cols-[0.9fr_1.1fr] [&>button]:bg-background/90 [&>button]:text-foreground">
        <DialogTitle className="sr-only">Offer Hub 登录与注册</DialogTitle>
        <DialogDescription className="sr-only">使用用户名和密码登录，或切换到注册创建账号。</DialogDescription>
        <section className="flex flex-col justify-between gap-6 rounded-t-xl bg-gradient-to-br from-emerald-950 via-emerald-800 to-teal-600 p-7 text-white md:rounded-l-xl md:rounded-tr-none md:p-10">
          <div className="flex items-center gap-3">
            <div className="rounded-xl bg-white/15 p-2.5"><BookOpen className="size-6" aria-hidden="true" /></div>
            <span className="text-xl font-semibold tracking-tight">Offer Hub</span>
          </div>
          <div className="space-y-6">
            <h2 className="text-2xl font-semibold leading-snug md:text-3xl">把每一道题，<br />变成下一次的底气。</h2>
            <ul className="space-y-3 text-sm text-emerald-50">
              {['按领域探索，找到适合你的题库', '关键词搜索，高效定位知识点', '阅读题目解析，巩固面试基础'].map((feature) => (
                <li key={feature} className="flex items-start gap-2"><Check className="mt-0.5 size-4 shrink-0" aria-hidden="true" /><span>{feature}</span></li>
              ))}
            </ul>
          </div>
          <p className="text-xs text-emerald-100/80">让准备更有方向，让进步看得见。</p>
        </section>
        <section aria-label="账号登录与注册" className="min-w-0 p-7 pt-10 sm:p-10 md:py-14">
          <UsernamePasswordAuthForms />
        </section>
      </DialogContent>
    </Dialog>
  )
}
