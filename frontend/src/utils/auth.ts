export const TOKEN_KEY = 'todo_token'
export const TOKEN_EXPIRES_KEY = 'todo_token_expires'

export const loadToken = () => {
  const token = localStorage.getItem(TOKEN_KEY) || ''
  const expires = Number(localStorage.getItem(TOKEN_EXPIRES_KEY) || '0')
  if (!token || !expires) {
    return ''
  }
  if (Date.now() / 1000 > expires) {
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(TOKEN_EXPIRES_KEY)
    return ''
  }
  return token
}

export const saveToken = (token: string, expiresAt: number) => {
  localStorage.setItem(TOKEN_KEY, token)
  localStorage.setItem(TOKEN_EXPIRES_KEY, String(expiresAt))
}

export const clearToken = () => {
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(TOKEN_EXPIRES_KEY)
}
