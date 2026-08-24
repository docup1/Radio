import type { Credentials, PasswordUpdate, User } from './types'

export class ApiError extends Error {
  status: number
  constructor(message: string, status: number) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

export async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(path, {
    method,
    credentials: 'include',
    headers: body ? { 'Content-Type': 'application/json' } : undefined,
    body: body ? JSON.stringify(body) : undefined,
  })

  if (!res.ok) {
    let message = res.statusText
    try {
      const data = (await res.json()) as { error?: string }
      if (data?.error) message = data.error
    } catch {
      // ignore non-JSON error bodies
    }
    throw new ApiError(message, res.status)
  }

  if (res.status === 204) return undefined as T
  const text = await res.text()
  return (text ? JSON.parse(text) : undefined) as T
}

export const api = {
  register: (c: Credentials) => request<User>('POST', '/api/auth/register', c),
  login: (c: Credentials) => request<User>('POST', '/api/auth/login', c),
  logout: () => request<void>('POST', '/api/auth/logout'),
  me: () => request<User>('GET', '/api/auth/me'),
  changePassword: (p: PasswordUpdate) => request<void>('POST', '/api/auth/password', p),
  deleteAccount: () => request<void>('DELETE', '/api/auth/me'),
}
