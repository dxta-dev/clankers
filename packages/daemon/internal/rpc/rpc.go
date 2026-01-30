package rpc

import (
	"context"
	"encoding/json"
	"errors"

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

type Handler struct {
	store *storage.Store
}

func NewHandler(store *storage.Store) *Handler {
	return &Handler{store: store}
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
