package rpc

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/dxta-dev/clankers/internal/logging"
	"github.com/dxta-dev/clankers/internal/paths"
	"github.com/dxta-dev/clankers/internal/storage"
	"github.com/sourcegraph/jsonrpc2"
)

const version = "0.1.0"

type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type RequestEnvelope struct {
	SchemaVersion string     `json:"schemaVersion"`
	Client        ClientInfo `json:"client"`
}

type HealthResult struct {
	OK      bool   `json:"ok"`
	Version string `json:"version"`
}

type EnsureDbResult struct {
	DbPath  string `json:"dbPath"`
	Created bool   `json:"created"`
}

type GetDbPathResult struct {
	DbPath string `json:"dbPath"`
}

type OkResult struct {
	OK bool `json:"ok"`
}

type UpsertSessionParams struct {
	RequestEnvelope
	Session storage.Session `json:"session"`
}

type UpsertMessageParams struct {
	RequestEnvelope
	Message storage.Message `json:"message"`
}

type UpsertToolParams struct {
	RequestEnvelope
	Tool storage.Tool `json:"tool"`
}

type UpsertSessionErrorParams struct {
	RequestEnvelope
	SessionError storage.SessionError `json:"sessionError"`
}

type UpsertCompactionEventParams struct {
	RequestEnvelope
	CompactionEvent storage.CompactionEvent `json:"compactionEvent"`
}

type LogWriteParams struct {
	RequestEnvelope
	Entry logging.LogEntry `json:"entry"`
}

type Handler struct {
	store  *storage.Store
	logger *logging.Logger
}

func NewHandler(store *storage.Store, logger *logging.Logger) *Handler {
	return &Handler{store: store, logger: logger}
}

func (h *Handler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	var result any
	var err error

	switch req.Method {
	case "health":
		result = h.health()
	case "ensureDb":
		result, err = h.ensureDb()
	case "getDbPath":
		result = h.getDbPath()
	case "upsertSession":
		result, err = h.upsertSession(req.Params)
	case "upsertMessage":
		result, err = h.upsertMessage(req.Params)
	case "upsertTool":
		result, err = h.upsertTool(req.Params)
	case "upsertSessionError":
		result, err = h.upsertSessionError(req.Params)
	case "upsertCompactionEvent":
		result, err = h.upsertCompactionEvent(req.Params)
	case "log.write":
		result, err = h.logWrite(req.Params)
	default:
		err = &jsonrpc2.Error{
			Code:    jsonrpc2.CodeMethodNotFound,
			Message: "method not found: " + req.Method,
		}
	}

	if err != nil {
		var rpcErr *jsonrpc2.Error
		if errors.As(err, &rpcErr) {
			conn.ReplyWithError(ctx, req.ID, rpcErr)
		} else {
			conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
				Code:    jsonrpc2.CodeInternalError,
				Message: err.Error(),
			})
		}
		return
	}

	conn.Reply(ctx, req.ID, result)
}

func (h *Handler) health() *HealthResult {
	return &HealthResult{OK: true, Version: version}
}

func (h *Handler) ensureDb() (*EnsureDbResult, error) {
	dbPath := paths.GetDbPath()
	created, err := storage.EnsureDb(dbPath)
	if err != nil {
		return nil, err
	}
	return &EnsureDbResult{DbPath: dbPath, Created: created}, nil
}

func (h *Handler) getDbPath() *GetDbPathResult {
	return &GetDbPathResult{DbPath: paths.GetDbPath()}
}

func (h *Handler) upsertSession(params *json.RawMessage) (*OkResult, error) {
	if params == nil {
		return nil, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: "missing params",
		}
	}

	var p UpsertSessionParams
	if err := json.Unmarshal(*params, &p); err != nil {
		return nil, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: "invalid params: " + err.Error(),
		}
	}

	if p.Session.ID == "" {
		data := json.RawMessage(`{"field": "id"}`)
		return nil, &jsonrpc2.Error{
			Code:    4001,
			Message: "invalid session payload",
			Data:    &data,
		}
	}

	if err := h.store.UpsertSession(&p.Session); err != nil {
		return nil, err
	}

	return &OkResult{OK: true}, nil
}

func (h *Handler) upsertMessage(params *json.RawMessage) (*OkResult, error) {
	if params == nil {
		return nil, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: "missing params",
		}
	}

	var p UpsertMessageParams
	if err := json.Unmarshal(*params, &p); err != nil {
		return nil, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: "invalid params: " + err.Error(),
		}
	}

	if p.Message.ID == "" {
		data := json.RawMessage(`{"field": "id"}`)
		return nil, &jsonrpc2.Error{
			Code:    4001,
			Message: "invalid message payload",
			Data:    &data,
		}
	}
	if p.Message.SessionID == "" {
		data := json.RawMessage(`{"field": "sessionId"}`)
		return nil, &jsonrpc2.Error{
			Code:    4001,
			Message: "invalid message payload",
			Data:    &data,
		}
	}

	if err := h.store.UpsertMessage(&p.Message); err != nil {
		return nil, err
	}

	return &OkResult{OK: true}, nil
}

func (h *Handler) upsertTool(params *json.RawMessage) (*OkResult, error) {
	if params == nil {
		return nil, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: "missing params",
		}
	}

	var p UpsertToolParams
	if err := json.Unmarshal(*params, &p); err != nil {
		return nil, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: "invalid params: " + err.Error(),
		}
	}

	if p.Tool.ID == "" {
		data := json.RawMessage(`{"field": "id"}`)
		return nil, &jsonrpc2.Error{
			Code:    4001,
			Message: "invalid tool payload",
			Data:    &data,
		}
	}
	if p.Tool.SessionID == "" {
		data := json.RawMessage(`{"field": "sessionId"}`)
		return nil, &jsonrpc2.Error{
			Code:    4001,
			Message: "invalid tool payload",
			Data:    &data,
		}
	}
	if p.Tool.ToolName == "" {
		data := json.RawMessage(`{"field": "toolName"}`)
		return nil, &jsonrpc2.Error{
			Code:    4001,
			Message: "invalid tool payload",
			Data:    &data,
		}
	}

	if err := h.store.UpsertTool(&p.Tool); err != nil {
		return nil, err
	}

	return &OkResult{OK: true}, nil
}

func (h *Handler) upsertSessionError(params *json.RawMessage) (*OkResult, error) {
	if params == nil {
		return nil, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: "missing params",
		}
	}

	var p UpsertSessionErrorParams
	if err := json.Unmarshal(*params, &p); err != nil {
		return nil, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: "invalid params: " + err.Error(),
		}
	}

	if p.SessionError.ID == "" {
		data := json.RawMessage(`{"field": "id"}`)
		return nil, &jsonrpc2.Error{
			Code:    4001,
			Message: "invalid session error payload",
			Data:    &data,
		}
	}
	if p.SessionError.SessionID == "" {
		data := json.RawMessage(`{"field": "sessionId"}`)
		return nil, &jsonrpc2.Error{
			Code:    4001,
			Message: "invalid session error payload",
			Data:    &data,
		}
	}

	if err := h.store.UpsertSessionError(&p.SessionError); err != nil {
		return nil, err
	}

	return &OkResult{OK: true}, nil
}

func (h *Handler) upsertCompactionEvent(params *json.RawMessage) (*OkResult, error) {
	if params == nil {
		return nil, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: "missing params",
		}
	}

	var p UpsertCompactionEventParams
	if err := json.Unmarshal(*params, &p); err != nil {
		return nil, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: "invalid params: " + err.Error(),
		}
	}

	if p.CompactionEvent.ID == "" {
		data := json.RawMessage(`{"field": "id"}`)
		return nil, &jsonrpc2.Error{
			Code:    4001,
			Message: "invalid compaction event payload",
			Data:    &data,
		}
	}
	if p.CompactionEvent.SessionID == "" {
		data := json.RawMessage(`{"field": "sessionId"}`)
		return nil, &jsonrpc2.Error{
			Code:    4001,
			Message: "invalid compaction event payload",
			Data:    &data,
		}
	}

	if err := h.store.UpsertCompactionEvent(&p.CompactionEvent); err != nil {
		return nil, err
	}

	return &OkResult{OK: true}, nil
}

func (h *Handler) logWrite(params *json.RawMessage) (*OkResult, error) {
	if params == nil {
		return nil, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: "missing params",
		}
	}

	var p LogWriteParams
	if err := json.Unmarshal(*params, &p); err != nil {
		return nil, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: "invalid params: " + err.Error(),
		}
	}

	// Set component from client name if not already set
	if p.Entry.Component == "" {
		p.Entry.Component = p.Client.Name
	}

	// Write to log (filtering happens inside logger)
	if err := h.logger.Write(p.Entry); err != nil {
		return nil, err
	}

	return &OkResult{OK: true}, nil
}
