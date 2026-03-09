package handlers

import (
	"net/http"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/models"
)

type UserHandler struct {
	// TODO: user репозиторий
}

func NewUserHandler() *UserHandler {
	return &UserHandler{}
}

// Register
func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := DecodeJSON(r, &req); err != nil {
		RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Валидация
	if req.Email == "" || req.Password == "" || req.Name == "" {
		RespondError(w, http.StatusBadRequest, "All fields are required")
		return
	}

	// TODO: Проверка, email(не занят)
	// TODO: Хеширование пароля
	// TODO: Сохранение в БД

	user := &models.User{
		ID:        1,
		Email:     req.Email,
		Name:      req.Name,
		Status:    models.UserStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// TODO: создать сессию / JWT токен
	// http.SetCookie(w, &http.Cookie{
	//     Name:     "session_id",
	//     Value:    "some-token",
	//     Path:     "/",
	//     HttpOnly: true,
	// })

	RespondJSON(w, http.StatusCreated, user)
}

// Login
func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := DecodeJSON(r, &req); err != nil {
		RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// TODO: Поиск пользователя по email
	// TODO: Проверка пароль

	// Заглушка
	if req.Email != "test@example.com" || req.Password != "password123" {
		RespondError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	user := &models.User{
		ID:        1,
		Email:     req.Email,
		Name:      "Test User",
		Status:    models.UserStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// TODO: Сессия
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "test-session", // уникальный токен!!!!!!!!
		Path:     "/",
		HttpOnly: true,
		MaxAge:   3600 * 24,
	})

	RespondJSON(w, http.StatusOK, user)
}

// Logout
func (h *UserHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Удаление куки
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	RespondJSON(w, http.StatusOK, map[string]string{
		"message": "Logged out successfully",
	})
}

// GetMe
func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	// TODO: Получение пользователя из контекста после авторизации
	// user := middleware.GetUserFromContext(r.Context())
	// if user == nil {
	//     RespondError(w, http.StatusUnauthorized, "Not authenticated")
	//     return
	// }

	// Заглушка
	user := &models.User{
		ID:        1,
		Email:     "test@example.com",
		Name:      "Test User",
		Status:    models.UserStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	RespondJSON(w, http.StatusOK, user)
}
