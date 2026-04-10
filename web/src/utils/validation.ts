// 表单验证相关常量和工具函数

export const MIN_PASSWORD_LENGTH = 6

export const passwordRules = [
  { required: true, message: '请输入密码', trigger: 'blur' },
  { min: MIN_PASSWORD_LENGTH, message: `密码长度不能少于${MIN_PASSWORD_LENGTH}位`, trigger: 'blur' },
]

export function isPasswordValid(password: string): boolean {
  return password.length >= MIN_PASSWORD_LENGTH
}

export function getPasswordError(password: string): string {
  if (!password) return '请输入密码'
  if (password.length < MIN_PASSWORD_LENGTH) return `密码至少需要${MIN_PASSWORD_LENGTH}位`
  return ''
}

export const requiredRule = (message: string) => ({
  required: true,
  message,
  trigger: 'blur',
})
