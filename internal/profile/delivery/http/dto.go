package http

// UpdateProfileRequest is the request body for updating user profile.
type UpdateProfileRequest struct {
	Username    string `json:"username"`
	Status      string `json:"status"`
	Description string `json:"description"`
}

// ChangePasswordRequest is the request body for changing password.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// ChangeEmailRequest is the request body for changing email.
type ChangeEmailRequest struct {
	NewEmail string `json:"new_email"`
	Password string `json:"password"`
}
