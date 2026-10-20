import { useEffect, useId, useMemo, useRef, useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { LoaderCircle } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { useLogin } from '@/hooks/useLogin'
import { createPasswordAuthSchema } from '@/lib/auth-validation'
import { postAuthLogin, postAuthRegister } from '@/services/auth'
import type { PasswordAuthFormValues } from '@/types/auth'

export default function UsernamePasswordAuthForms() {
  const [mode, setMode] = useState<'login' | 'register'>('login')
  const [notice, setNotice] = useState('')
  const { login } = useLogin()
  const id = useId()
  const mounted = useRef(true)
  const schema = useMemo(() => createPasswordAuthSchema(mode), [mode])
  const {
    register, handleSubmit, reset, getValues, setError,
    formState: { errors, isSubmitting },
  } = useForm<PasswordAuthFormValues>({
    resolver: zodResolver(schema),
    defaultValues: { username: '', password: '', confirmPassword: '' },
  })
  const isRegister = mode === 'register'
  const inputClass = 'h-11 w-full rounded-lg border bg-background px-3 text-sm outline-none transition-shadow focus-visible:border-emerald-600 focus-visible:ring-2 focus-visible:ring-emerald-600/20 aria-invalid:border-destructive disabled:opacity-50'

  useEffect(() => {
    mounted.current = true
    return () => { mounted.current = false }
  }, [])

  function switchMode() {
    reset({ username: getValues('username'), password: '', confirmPassword: '' })
    setNotice('')
    setMode(isRegister ? 'login' : 'register')
  }

  const onSubmit = handleSubmit(async (values) => {
    setNotice('')
    try {
      if (isRegister) {
        await postAuthRegister(values.username, values.password)
        if (!mounted.current) return
        reset({ username: values.username, password: '', confirmPassword: '' })
        setMode('login')
        setNotice('注册成功，请使用新账号登录。')
      } else {
        const data = await postAuthLogin(values.username, values.password)
        if (!mounted.current) return
        login(data.userInfo, data.token)
      }
    } catch (error) {
      if (!mounted.current) return
      setError('root.server', {
        message: error instanceof Error ? error.message : '请求失败，请稍后重试。',
      })
    }
  })

  return (
    <div className="space-y-6">
      <div className="space-y-2">
        <h2 className="text-2xl font-semibold tracking-tight">{isRegister ? '创建你的账号' : '欢迎回来'}</h2>
        <p className="text-sm leading-6 text-muted-foreground">
          {isRegister ? '注册 Offer Hub，开始你的面试准备之旅。' : '登录 Offer Hub，继续你的学习之旅。'}
        </p>
      </div>
      <form noValidate onSubmit={onSubmit} className="space-y-5">
        <fieldset disabled={isSubmitting} className="min-w-0 space-y-5">
          <div className="space-y-2">
            <label htmlFor={`${id}-username`} className="text-sm font-medium">用户名</label>
            <input
              id={`${id}-username`}
              {...register('username')}
              autoComplete="username"
              placeholder="请输入用户名"
              aria-invalid={Boolean(errors.username)}
              aria-describedby={errors.username ? `${id}-username-error` : undefined}
              className={inputClass}
            />
            {errors.username && <p id={`${id}-username-error`} role="alert" className="text-xs text-destructive">{errors.username.message}</p>}
          </div>
          <div className="space-y-2">
            <label htmlFor={`${id}-password`} className="text-sm font-medium">密码</label>
            <input
              id={`${id}-password`}
              {...register('password')}
              type="password"
              autoComplete={isRegister ? 'new-password' : 'current-password'}
              placeholder="请输入至少 6 位密码"
              aria-invalid={Boolean(errors.password)}
              aria-describedby={errors.password ? `${id}-password-error` : undefined}
              className={inputClass}
            />
            {errors.password && <p id={`${id}-password-error`} role="alert" className="text-xs text-destructive">{errors.password.message}</p>}
          </div>
          {isRegister && (
            <div className="space-y-2">
              <label htmlFor={`${id}-confirm`} className="text-sm font-medium">确认密码</label>
              <input
                id={`${id}-confirm`}
                {...register('confirmPassword')}
                type="password"
                autoComplete="new-password"
                placeholder="请再次输入密码"
                aria-invalid={Boolean(errors.confirmPassword)}
                aria-describedby={errors.confirmPassword ? `${id}-confirm-error` : undefined}
                className={inputClass}
              />
              {errors.confirmPassword && <p id={`${id}-confirm-error`} role="alert" className="text-xs text-destructive">{errors.confirmPassword.message}</p>}
            </div>
          )}
        </fieldset>
        {errors.root?.server && <p role="alert" className="break-words rounded-lg bg-destructive/10 p-3 text-sm text-destructive">{errors.root.server.message}</p>}
        {notice && <p role="status" className="rounded-lg bg-emerald-50 p-3 text-sm text-emerald-800">{notice}</p>}
        <Button type="submit" disabled={isSubmitting} className="h-11 w-full bg-emerald-700 text-white hover:bg-emerald-800" aria-busy={isSubmitting}>
          {isSubmitting && <LoaderCircle className="animate-spin" aria-hidden="true" />}
          {isSubmitting ? (isRegister ? '注册中…' : '登录中…') : (isRegister ? '注册' : '登录')}
        </Button>
      </form>
      <p className="text-center text-sm text-muted-foreground">
        {isRegister ? '已经有账号？' : '还没有账号？'}{' '}
        <button type="button" disabled={isSubmitting} onClick={switchMode} className="rounded-sm font-medium text-emerald-700 underline-offset-4 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-600 disabled:opacity-50">
          {isRegister ? '去登录' : '立即注册'}
        </button>
      </p>
    </div>
  )
}
