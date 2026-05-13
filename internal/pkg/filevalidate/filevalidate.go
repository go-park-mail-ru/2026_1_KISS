package filevalidate

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

var blockedExtensions = map[string]struct{}{
	".exe": {}, ".bat": {}, ".cmd": {}, ".com": {}, ".scr": {},
	".msi": {}, ".dll": {}, ".ps1": {}, ".vbs": {}, ".jar": {},
	".sh": {}, ".app": {}, ".dmg": {}, ".pkg": {}, ".deb": {}, ".rpm": {},
}

var mimeToExpectedExt = map[string]string{
	"image/jpeg":                   ".jpg",
	"image/png":                    ".png",
	"image/gif":                    ".gif",
	"image/webp":                   ".webp",
	"image/bmp":                    ".bmp",
	"application/pdf":              ".pdf",
	"application/zip":              ".zip",
	"application/x-7z-compressed":  ".7z",
	"application/x-rar-compressed": ".rar",
	"application/gzip":             ".gz",
	"application/x-tar":            ".tar",
}

var commonAllowed = map[string]struct{}{
	"image/jpeg": {}, "image/png": {}, "image/gif": {}, "image/webp": {}, "image/bmp": {},
	"image/svg+xml":   {},
	"application/pdf": {},
	"text/plain":      {}, "text/csv": {}, "text/markdown": {}, "text/html": {}, "text/xml": {},
	"application/json": {}, "application/xml": {},
	"application/zip": {}, "application/x-7z-compressed": {}, "application/x-rar-compressed": {},
	"application/gzip": {}, "application/x-tar": {},
	"application/msword": {},
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document":   {},
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         {},
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": {},
	"application/vnd.ms-excel": {}, "application/vnd.ms-powerpoint": {},
	"audio/mpeg": {}, "audio/wav": {}, "audio/ogg": {}, "audio/x-wav": {},
	"video/mp4": {}, "video/webm": {}, "video/quicktime": {},
	"application/octet-stream": {},
}

var avatarsAllowed = map[string]struct{}{
	"image/jpeg": {}, "image/png": {}, "image/gif": {}, "image/webp": {},
}

func allowedFor(category domain.FileCategory) map[string]struct{} {
	if category == domain.FileCategoryAvatar {
		return avatarsAllowed
	}
	return commonAllowed
}

func Validate(category domain.FileCategory, filename, sniffedMIME string) error {
	ext := strings.ToLower(filepath.Ext(filename))
	if _, blocked := blockedExtensions[ext]; blocked {
		return fmt.Errorf("%w: file type %q is not allowed", domain.ErrInvalidInput, ext)
	}

	allowed := allowedFor(category)
	if _, ok := allowed[sniffedMIME]; !ok {
		return fmt.Errorf("%w: mime type %q is not allowed for category %q", domain.ErrInvalidInput, sniffedMIME, category)
	}

	if expected, known := mimeToExpectedExt[sniffedMIME]; known && ext != "" && ext != expected {
		if !isCompatibleExt(sniffedMIME, ext) {
			return fmt.Errorf("%w: file extension %q does not match content type %q", domain.ErrInvalidInput, ext, sniffedMIME)
		}
	}

	return nil
}

func isCompatibleExt(mime, ext string) bool {
	switch mime {
	case "image/jpeg":
		return ext == ".jpg" || ext == ".jpeg"
	case "application/zip":
		return ext == ".zip" || ext == ".docx" || ext == ".xlsx" || ext == ".pptx" || ext == ".jar"
	case "application/x-tar":
		return ext == ".tar" || ext == ".tgz"
	case "application/gzip":
		return ext == ".gz" || ext == ".tgz"
	}
	return false
}
