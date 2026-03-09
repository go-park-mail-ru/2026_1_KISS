package handlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/models"
)

type FileHandler struct {
}

func NewFileHandler() *FileHandler {
	return &FileHandler{}
}

// Список файлов пользователя
func (h *FileHandler) ListFiles(w http.ResponseWriter, r *http.Request) {
	// TODO: Получать userID
	userID := int64(1)

	// TODO: Заменить на запрос к БД
	files := []models.IPYNBFile{
		{
			ID:                  1,
			OwnerID:             userID,
			Title:               "Мой блокнот",
			NbFormat:            4,
			NbFormatMinor:       0,
			ProgrammingLanguage: models.LangPython,
			CreatedAt:           time.Now().Add(-24 * time.Hour),
			UpdatedAt:           time.Now(),
		},
		{
			ID:                  2,
			OwnerID:             userID,
			Title:               "Анализ данных",
			NbFormat:            4,
			NbFormatMinor:       0,
			ProgrammingLanguage: models.LangR,
			CreatedAt:           time.Now().Add(-12 * time.Hour),
			UpdatedAt:           time.Now(),
		},
	}

	RespondJSON(w, http.StatusOK, files)
}

// Создание нового файла
func (h *FileHandler) CreateFile(w http.ResponseWriter, r *http.Request) {
	// TODO: Получать userID
	userID := int64(1)

	var req models.CreateFileRequest
	if err := DecodeJSON(r, &req); err != nil {
		RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Title == "" {
		RespondError(w, http.StatusBadRequest, "Title is required")
		return
	}

	if req.ProgrammingLanguage == "" {
		RespondError(w, http.StatusBadRequest, "Programming language is required")
		return
	}

	nbFormat := 4
	if req.NbFormat != nil {
		nbFormat = *req.NbFormat
	}

	nbFormatMinor := 0
	if req.NbFormatMinor != nil {
		nbFormatMinor = *req.NbFormatMinor
	}

	if nbFormat != 4 {
		RespondError(w, http.StatusBadRequest, "Only nbformat=4 is supported")
		return
	}

	if nbFormatMinor != 0 {
		RespondError(w, http.StatusBadRequest, "Only nbformat_minor=0 is supported")
		return
	}

	// Файл
	file := &models.IPYNBFile{
		ID:                  3,
		OwnerID:             userID,
		Title:               req.Title,
		NbFormat:            nbFormat,
		NbFormatMinor:       nbFormatMinor,
		ProgrammingLanguage: req.ProgrammingLanguage,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	RespondJSON(w, http.StatusCreated, file)
}

// Страница файла/основной рабочий экран пользователя
func (h *FileHandler) GetFile(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		RespondError(w, http.StatusBadRequest, "Invalid file ID")
		return
	}

	fileID, err := strconv.ParseInt(pathParts[3], 10, 64)
	if err != nil {
		RespondError(w, http.StatusBadRequest, "Invalid file ID")
		return
	}

	// TODO: Проверка, на пренадлежность файла определенному пользователю
	// userID := middleware.GetUserID(r.Context())

	// Файл
	file := &models.IPYNBFile{
		ID:                  fileID,
		OwnerID:             1,
		Title:               "Тестовый блокнот",
		NbFormat:            4,
		NbFormatMinor:       0,
		ProgrammingLanguage: models.LangPython,
		CreatedAt:           time.Now().Add(-24 * time.Hour),
		UpdatedAt:           time.Now(),
	}

	// Ячейки файла
	cells := []models.IPYNBCell{
		{
			ID:         1,
			FileID:     fileID,
			OrderIndex: 0,
			CellType:   models.CellTypeMarkdown,
			Source:     "# Мой первый блокнот\nЭто пример markdown ячейки",
			CreatedAt:  time.Now().Add(-23 * time.Hour),
			UpdatedAt:  time.Now().Add(-23 * time.Hour),
		},
		{
			ID:         2,
			FileID:     fileID,
			OrderIndex: 1,
			CellType:   models.CellTypeCode,
			Source:     "print('Hello, World!')",
			CreatedAt:  time.Now().Add(-22 * time.Hour),
			UpdatedAt:  time.Now().Add(-22 * time.Hour),
		},
	}

	response := struct {
		File    *models.IPYNBFile  `json:"file"`
		Cells   []models.IPYNBCell `json:"cells"`
		IsOwner bool               `json:"is_owner"`
	}{
		File:    file,
		Cells:   cells,
		IsOwner: true,
	}

	RespondJSON(w, http.StatusOK, response)
}

// Удаление файла
func (h *FileHandler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		RespondError(w, http.StatusBadRequest, "Invalid file ID")
		return
	}

	fileID, err := strconv.ParseInt(pathParts[3], 10, 64)
	if err != nil {
		RespondError(w, http.StatusBadRequest, "Invalid file ID")
		return
	}
	_ = fileID

	// TODO: Проверка прав и удаление из БД

	RespondJSON(w, http.StatusOK, map[string]string{
		"message": "File deleted successfully",
	})
}

// Создание ячейки
func (h *FileHandler) CreateCell(w http.ResponseWriter, r *http.Request) {

	log.Println("=== CreateCell called ===")
	log.Printf("Full URL: %s", r.URL.Path)

	pathParts := strings.Split(r.URL.Path, "/")
	log.Printf("Path parts: %v", pathParts)
	log.Printf("Number of parts: %d", len(pathParts))

	// Проверим каждый элемент
	for i, part := range pathParts {
		log.Printf("part[%d]: '%s'", i, part)
	}

	if len(pathParts) < 5 {
		RespondError(w, http.StatusBadRequest, "Invalid file ID")
		return
	}

	log.Printf("Trying to parse fileID from part[3]: '%s'", pathParts[3])

	fileID, err := strconv.ParseInt(pathParts[3], 10, 64)

	log.Printf("Parsed fileID: %d, error: %v", fileID, err)

	if err != nil {
		RespondError(w, http.StatusBadRequest, "Invalid file ID")
		return
	}

	var req models.CreateCellRequest
	if err := DecodeJSON(r, &req); err != nil {
		RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	log.Printf("Request: cell_type=%s, source=%s", req.CellType, req.Source)

	if req.CellType != models.CellTypeCode &&
		req.CellType != models.CellTypeMarkdown &&
		req.CellType != models.CellTypeRaw {
		log.Printf("ERROR invalid cell type: %s", req.CellType)
		RespondError(w, http.StatusBadRequest, "Cell type must be 'code', 'markdown', or 'raw'")
		return
	}

	orderIndex := 2

	// Ячейка
	cell := &models.IPYNBCell{
		ID:         3,
		FileID:     fileID,
		OrderIndex: orderIndex,
		CellType:   req.CellType,
		Source:     req.Source,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	log.Printf("SUCCESS: created cell for fileID=%d", fileID)

	RespondJSON(w, http.StatusCreated, cell)
}

// Обновление ячейки
func (h *FileHandler) UpdateCell(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 6 {
		RespondError(w, http.StatusBadRequest, "Invalid cell ID")
		return
	}
	fileID, _ := strconv.ParseInt(pathParts[3], 10, 64)
	cellID, err := strconv.ParseInt(pathParts[5], 10, 64)
	if err != nil {
		RespondError(w, http.StatusBadRequest, "Invalid cell ID")
		return
	}

	var req struct {
		Source     *string          `json:"source,omitempty"`
		CellType   *models.CellType `json:"cell_type,omitempty"`
		OrderIndex *int             `json:"order_index,omitempty"`
	}

	if err := DecodeJSON(r, &req); err != nil {
		RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Обновить ячейку
	cell := &models.IPYNBCell{
		ID:        cellID,
		FileID:    fileID,
		CellType:  models.CellTypeCode,
		Source:    *req.Source,
		UpdatedAt: time.Now(),
	}

	RespondJSON(w, http.StatusOK, cell)
}

// Удаление ячейк
func (h *FileHandler) DeleteCell(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 6 {
		RespondError(w, http.StatusBadRequest, "Invalid cell ID")
		return
	}

	cellID, err := strconv.ParseInt(pathParts[5], 10, 64)
	if err != nil {
		RespondError(w, http.StatusBadRequest, "Invalid cell ID")
		return
	}
	_ = cellID

	RespondJSON(w, http.StatusOK, map[string]string{
		"message": "Cell deleted successfully",
	})
}
