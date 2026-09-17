import type { Credentials, PasswordUpdate, User } from './types'

export class ApiError extends Error {
  status: number
  constructor(message: string, status: number) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

const REQUEST_TIMEOUT_MS = 20_000

// request performs a JSON API call. Every request is bounded by a timeout so a
// stalled backend can never leave UI in an infinite loading state.
export async function request<T>(
  method: string,
  path: string,
  body?: unknown,
  timeoutMs: number = REQUEST_TIMEOUT_MS,
): Promise<T> {
  const controller = new AbortController()
  const timer = setTimeout(() => controller.abort(), timeoutMs)
  let res: Response
  try {
    res = await fetch(path, {
      method,
      credentials: 'include',
      headers: body ? { 'Content-Type': 'application/json' } : undefined,
      body: body ? JSON.stringify(body) : undefined,
      signal: controller.signal,
    })
  } catch (e) {
    if (controller.signal.aborted) {
      throw new ApiError('Превышено время ожидания', 0)
    }
    throw e
  } finally {
    clearTimeout(timer)
  }

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
  changePassword: (p: PasswordUpdate) => request<void>('PUT', '/api/auth/password', p),
}
