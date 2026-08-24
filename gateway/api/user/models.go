package user

// Credentials is the register/login payload.
type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// AuthResponse describes the authenticated user.
type AuthResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

// PasswordUpdate is the change-password payload.
type PasswordUpdate struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// ErrorResponse is the standard error envelope.
type ErrorResponse struct {
	Error string `json:"error"`
}
