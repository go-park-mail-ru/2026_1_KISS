package httputil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/mailru/easyjson"
)

type errorResponse struct {
	Error string `json:"error,omitempty"`
}

func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if data == nil {
		_, _ = w.Write([]byte(`{}`))
		return
	}

	if m, ok := data.(easyjson.Marshaler); ok {
		buf, err := easyjson.Marshal(m)
		if err != nil {
			slog.Error("easyjson encode failed", "error", err)
			return
		}
		wrapped := append([]byte(`{"data":`), buf...)
		wrapped = append(wrapped, '}')
		_, _ = w.Write(wrapped)
		return
	}

	buf, err := json.Marshal(data)
	if err != nil {
		slog.Error("json encode failed", "error", err)
		return
	}
	wrapped := append([]byte(`{"data":`), buf...)
	wrapped = append(wrapped, '}')
	_, _ = w.Write(wrapped)
}

func Error(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := easyjson.MarshalToWriter(errorResponse{Error: message}, w); err != nil {
		slog.Error("easyjson encode failed", "error", err)
	}
}

func DecodeJSON(r *http.Request, dst interface{}) error {
	if r.Body == nil {
		return fmt.Errorf("empty request body")
	}
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}
	if u, ok := dst.(easyjson.Unmarshaler); ok {
		return easyjson.Unmarshal(body, u)
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}
