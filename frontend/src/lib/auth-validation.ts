import { z } from 'zod'

export function createPasswordAuthSchema(mode: 'login' | 'register') {
  return z.object({
    username: z.string().trim().min(1, '请输入用户名').max(50, '用户名不能超过 50 个字符'),
    password: z.string().min(6, '密码至少需要 6 位'),
    confirmPassword: z.string(),
  }).refine((values) => mode !== 'register' || values.password === values.confirmPassword, {
    message: '两次密码不一致',
    path: ['confirmPassword'],
  })
}
