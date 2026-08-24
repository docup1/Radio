export interface Credentials {
  username: string
  password: string
}

export interface User {
  id: string
  username: string
}

export interface PasswordUpdate {
  current_password: string
  new_password: string
}

export interface ApiErrorBody {
  error: string
}
