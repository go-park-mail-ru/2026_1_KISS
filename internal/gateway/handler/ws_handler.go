package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/google/uuid"
	"github.com/mailru/easyjson"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/grpcutil"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/logger"
	pbauth "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/auth"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/notebook"
	pbrunner "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/runner"
)

type WSHandler struct {
	authClient   pbauth.AuthServiceClient
	nbClient     pb.NotebookServiceClient
	runnerClient pbrunner.RunnerServiceClient
}

func NewWSHandler(authClient pbauth.AuthServiceClient, nbClient pb.NotebookServiceClient, runnerClient ...pbrunner.RunnerServiceClient) *WSHandler {
	h := &WSHandler{authClient: authClient, nbClient: nbClient}
	if len(runnerClient) > 0 {
		h.runnerClient = runnerClient[0]
	}
	return h
}

func (h *WSHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/ws/notebooks/{id}", h.HandleNotebook)
}

func (h *WSHandler) HandleNotebook(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authenticate(w, r)
	if !ok {
		return
	}

	notebookID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		logger.Warn(r.Context(), "ws.HandleNotebook", "error", "invalid notebook id", "user_id", userID, "raw_id", r.PathValue("id"))
		http.Error(w, "invalid notebook id", http.StatusBadRequest)
		return
	}

	if !h.checkReadAccess(w, r.Context(), userID, notebookID) {
		return
	}

	wsConn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		logger.Error(r.Context(), "ws.HandleNotebook", "error", err, "stage", "accept", "user_id", userID, "notebook_id", notebookID)
		return
	}
	defer wsConn.CloseNow() //nolint:errcheck

	connID := uuid.NewString()
	startedAt := time.Now()
	logger.Info(r.Context(), "ws.HandleNotebook", "stage", "open", "user_id", userID, "notebook_id", notebookID, "conn_id", connID)

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	stream, err := h.nbClient.SubscribeNotebook(ctx, &pb.SubscribeNotebookRequest{
		NotebookId: notebookID,
		UserId:     userID,
	})
	if err != nil {
		logger.Error(ctx, "ws.HandleNotebook", "error", err, "stage", "subscribe", "user_id", userID, "notebook_id", notebookID, "conn_id", connID)
		_ = wsConn.Close(websocket.StatusInternalError, "subscribe failed")
		return
	}

	writeDone := make(chan struct{})
	go func() {
		defer close(writeDone)
		h.writePump(ctx, wsConn, stream, userID, notebookID, connID)
	}()

	h.readPump(ctx, wsConn, userID, notebookID, connID)
	cancel()
	<-writeDone

	logger.Info(r.Context(), "ws.HandleNotebook",
		"stage", "close",
		"user_id", userID,
		"notebook_id", notebookID,
		"conn_id", connID,
		"duration", time.Since(startedAt).String(),
	)
}

func (h *WSHandler) authenticate(w http.ResponseWriter, r *http.Request) (int64, bool) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		logger.Warn(r.Context(), "ws.authenticate", "error", "no session cookie", "remote_addr", r.RemoteAddr)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return 0, false
	}
	resp, err := h.authClient.ValidateSession(r.Context(), &pbauth.ValidateSessionRequest{SessionId: cookie.Value})
	if err != nil {
		logger.Warn(r.Context(), "ws.authenticate", "error", err, "remote_addr", r.RemoteAddr)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return 0, false
	}
	return resp.GetUser().GetId(), true
}

func (h *WSHandler) checkReadAccess(w http.ResponseWriter, ctx context.Context, userID, notebookID int64) bool {
	_, err := h.nbClient.GetByID(ctx, &pb.GetNotebookRequest{UserId: userID, NotebookId: notebookID})
	if err == nil {
		return true
	}
	domainErr := grpcutil.GRPCToDomainError(err)
	switch domainErr {
	case domain.ErrNotFound:
		logger.Warn(ctx, "ws.checkReadAccess", "result", "not_found", "user_id", userID, "notebook_id", notebookID)
		http.Error(w, "not found", http.StatusNotFound)
	case domain.ErrForbidden:
		logger.Warn(ctx, "ws.checkReadAccess", "result", "forbidden", "user_id", userID, "notebook_id", notebookID)
		http.Error(w, "forbidden", http.StatusForbidden)
	default:
		logger.Error(ctx, "ws.checkReadAccess", "error", err, "user_id", userID, "notebook_id", notebookID)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
	return false
}

type wsIncoming struct {
	Type          string `json:"type"`
	BlockID       int64  `json:"block_id,omitempty"`
	Content       string `json:"content,omitempty"`
	Language      string `json:"language,omitempty"`
	BlockType     string `json:"block_type,omitempty"`
	BlockPosition int32  `json:"block_position,omitempty"`
}

type wsOutgoing struct {
	Type    string          `json:"type"`
	Block   json.RawMessage `json:"block,omitempty"`
	BlockID int64           `json:"block_id,omitempty"`
	ActorID int64           `json:"actor_id,omitempty"`
	Message string          `json:"message,omitempty"`
}

type wsBlock struct {
	ID         int64  `json:"id"`
	NotebookID int64  `json:"notebook_id"`
	Type       string `json:"type"`
	Language   string `json:"language"`
	Content    string `json:"content"`
	Position   int32  `json:"position"`
	CreatedAt  int64  `json:"created_at"`
	UpdatedAt  int64  `json:"updated_at"`
}

type wsExecuteResult struct {
	BlockID    int64    `json:"block_id"`
	Position   int32    `json:"position"`
	Stdout     []string `json:"stdout"`
	Stderr     []string `json:"stderr"`
	Result     string   `json:"result"`
	Error      string   `json:"error"`
	ExecutedAt int64    `json:"executed_at"`
	DurationNs int64    `json:"duration_ns"`
}

func (h *WSHandler) readPump(ctx context.Context, conn *websocket.Conn, userID, notebookID int64, connID string) {
	for {
		var msg wsIncoming
		if err := wsjson.Read(ctx, conn, &msg); err != nil {
			// Нормальный путь — клиент закрыл вкладку или потерял сеть.
			// Логируем как Info, чтобы не спамить Error-лог.
			if ctx.Err() == nil {
				logger.Info(ctx, "ws.readPump", "stage", "exit", "reason", err.Error(), "user_id", userID, "notebook_id", notebookID, "conn_id", connID)
			}
			return
		}
		switch msg.Type {
		case "ping":
			_ = wsjson.Write(ctx, conn, wsOutgoing{Type: "pong"})
		case "update_block", "add_block", "delete_block":
			logger.Info(ctx, "ws.readPump",
				"stage", "mutation",
				"type", msg.Type,
				"user_id", userID,
				"notebook_id", notebookID,
				"block_id", msg.BlockID,
				"conn_id", connID,
			)
			if err := h.handleMutation(ctx, msg, userID, notebookID); err != nil {
				logger.Warn(ctx, "ws.readPump",
					"stage", "mutation_failed",
					"type", msg.Type,
					"error", err,
					"user_id", userID,
					"notebook_id", notebookID,
					"block_id", msg.BlockID,
					"conn_id", connID,
				)
				_ = wsjson.Write(ctx, conn, errorEvent(err))
			}
		case "execute_block":
			go h.handleExecuteBlock(ctx, conn, msg, userID, notebookID, connID)
		default:
			logger.Warn(ctx, "ws.readPump", "stage", "unknown_type", "type", msg.Type, "user_id", userID, "notebook_id", notebookID, "conn_id", connID)
		}
	}
}

func (h *WSHandler) handleMutation(ctx context.Context, msg wsIncoming, userID, notebookID int64) error {
	switch msg.Type {
	case "update_block":
		_, err := h.nbClient.UpdateBlock(ctx, &pb.UpdateBlockRequest{
			UserId: userID, NotebookId: notebookID, BlockId: msg.BlockID,
			Content: msg.Content, Language: msg.Language,
		})
		return err
	case "add_block":
		_, err := h.nbClient.AddBlock(ctx, &pb.AddBlockRequest{
			UserId: userID, NotebookId: notebookID,
			Type: msg.BlockType, Language: msg.Language,
		})
		return err
	case "delete_block":
		_, err := h.nbClient.DeleteBlock(ctx, &pb.DeleteBlockRequest{
			UserId: userID, NotebookId: notebookID, BlockId: msg.BlockID,
		})
		return err
	}
	return nil
}

func (h *WSHandler) handleExecuteBlock(ctx context.Context, conn *websocket.Conn, msg wsIncoming, userID, notebookID int64, connID string) {
	if h.runnerClient == nil {
		_ = wsjson.Write(ctx, conn, wsOutgoing{Type: "error", Message: "runner not configured"})
		return
	}

	logger.Info(ctx, "ws.executeBlock", "user_id", userID, "notebook_id", notebookID, "block_position", msg.BlockPosition, "conn_id", connID)

	stream, err := h.runnerClient.ExecuteBlockStream(ctx, &pbrunner.ExecuteBlockRequest{
		NotebookId:    notebookID,
		BlockPosition: msg.BlockPosition,
		UserId:        userID,
	})
	if err != nil {
		_ = wsjson.Write(ctx, conn, wsOutgoing{Type: "execute_error", Message: grpcutil.GRPCToDomainError(err).Error()})
		return
	}

	for {
		chunk, err := stream.Recv()
		if err != nil {
			if !errors.Is(err, io.EOF) {
				_ = wsjson.Write(ctx, conn, wsOutgoing{Type: "execute_error", Message: err.Error()})
			}
			return
		}

		switch chunk.GetChunkType() {
		case "complete":
			result := chunk.GetFinalResult()
			raw, _ := easyjson.Marshal(wsExecuteResult{
				BlockID:    result.GetBlockId(),
				Position:   result.GetPosition(),
				Stdout:     result.GetStdout(),
				Stderr:     result.GetStderr(),
				Result:     result.GetResult(),
				Error:      result.GetError(),
				ExecutedAt: result.GetExecutedAt(),
				DurationNs: result.GetDurationNs(),
			})
			_ = wsjson.Write(ctx, conn, wsOutgoing{Type: "execute_completed", Block: raw})
			return
		default:
			_ = wsjson.Write(ctx, conn, wsOutgoing{Type: chunk.GetChunkType() + "_chunk", Message: chunk.GetData()})
		}
	}
}

func (h *WSHandler) writePump(ctx context.Context, conn *websocket.Conn, stream pb.NotebookService_SubscribeNotebookClient, userID, notebookID int64, connID string) {
	for {
		event, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) || ctx.Err() != nil {
				return
			}
			logger.Error(ctx, "ws.writePump",
				"stage", "stream_recv_failed",
				"error", err,
				"user_id", userID,
				"notebook_id", notebookID,
				"conn_id", connID,
			)
			_ = conn.Close(websocket.StatusInternalError, "stream error")
			return
		}
		if err := wsjson.Write(ctx, conn, eventToWS(event)); err != nil {
			// При закрытии WS клиентом ошибка ожидаема — Warn вместо Error.
			logger.Warn(ctx, "ws.writePump",
				"stage", "ws_write_failed",
				"error", err,
				"user_id", userID,
				"notebook_id", notebookID,
				"conn_id", connID,
			)
			return
		}
	}
}

func eventToWS(e *pb.NotebookEvent) wsOutgoing {
	out := wsOutgoing{ActorID: e.GetActorId()}
	switch e.GetType() {
	case pb.NotebookEvent_BLOCK_ADDED:
		out.Type = "block_added"
		out.Block = marshalBlock(e.GetBlock())
	case pb.NotebookEvent_BLOCK_UPDATED:
		out.Type = "block_updated"
		out.Block = marshalBlock(e.GetBlock())
	case pb.NotebookEvent_BLOCK_DELETED:
		out.Type = "block_deleted"
		out.BlockID = e.GetDeletedBlockId()
	case pb.NotebookEvent_NOTEBOOK_UPDATED:
		out.Type = "notebook_updated"
	}
	return out
}

func errorEvent(err error) wsOutgoing {
	msg := "internal error"
	switch grpcutil.GRPCToDomainError(err) {
	case domain.ErrForbidden:
		msg = "forbidden"
	case domain.ErrNotFound:
		msg = "not found"
	case domain.ErrInvalidInput:
		msg = "invalid input"
	}
	return wsOutgoing{Type: "error", Message: msg}
}

func marshalBlock(b *pb.BlockInfo) json.RawMessage {
	if b == nil {
		return nil
	}
	raw, _ := easyjson.Marshal(wsBlock{
		ID:         b.GetId(),
		NotebookID: b.GetNotebookId(),
		Type:       b.GetType(),
		Language:   b.GetLanguage(),
		Content:    b.GetContent(),
		Position:   b.GetPosition(),
		CreatedAt:  b.GetCreatedAt(),
		UpdatedAt:  b.GetUpdatedAt(),
	})
	return raw
}
