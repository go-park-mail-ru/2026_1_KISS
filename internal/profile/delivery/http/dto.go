package http

type UpdateProfileRequest struct {
	Username    string `json:"username"`
	Status      string `json:"status"`
	Description string `json:"description"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type ChangeEmailRequest struct {
	NewEmail string `json:"new_email"`
	Password string `json:"password"`
}
