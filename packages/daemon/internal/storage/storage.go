package storage

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const schemaSQL = `
CREATE TABLE IF NOT EXISTS sessions (
	id TEXT PRIMARY KEY,
	title TEXT,
	project_path TEXT,
	project_name TEXT,
	model TEXT,
	provider TEXT,
	prompt_tokens INTEGER,
	completion_tokens INTEGER,
	cost REAL,
	created_at INTEGER,
	updated_at INTEGER
);

CREATE TABLE IF NOT EXISTS messages (
	id TEXT PRIMARY KEY,
	session_id TEXT,
	role TEXT,
	text_content TEXT,
	model TEXT,
	prompt_tokens INTEGER,
	completion_tokens INTEGER,
	duration_ms INTEGER,
	created_at INTEGER,
	completed_at INTEGER,
	FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);
`

const upsertSessionSQL = `
INSERT INTO sessions (
	id, title, project_path, project_name, model, provider,
	prompt_tokens, completion_tokens, cost, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	title=excluded.title,
	project_path=excluded.project_path,
	project_name=excluded.project_name,
	model=excluded.model,
	provider=excluded.provider,
	prompt_tokens=excluded.prompt_tokens,
	completion_tokens=excluded.completion_tokens,
	cost=excluded.cost,
	created_at=excluded.created_at,
	updated_at=excluded.updated_at;
`

const upsertMessageSQL = `
INSERT INTO messages (
	id, session_id, role, text_content, model,
	prompt_tokens, completion_tokens, duration_ms,
	created_at, completed_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	session_id=excluded.session_id,
	role=excluded.role,
	text_content=excluded.text_content,
	model=excluded.model,
	prompt_tokens=excluded.prompt_tokens,
	completion_tokens=excluded.completion_tokens,
	duration_ms=excluded.duration_ms,
	created_at=excluded.created_at,
	completed_at=excluded.completed_at;
`

type Store struct {
	db            *sql.DB
	upsertSession *sql.Stmt
	upsertMessage *sql.Stmt
}

type Session struct {
	ID               string   `json:"id"`
	Title            *string  `json:"title,omitempty"`
	ProjectPath      *string  `json:"projectPath,omitempty"`
	ProjectName      *string  `json:"projectName,omitempty"`
	Model            *string  `json:"model,omitempty"`
	Provider         *string  `json:"provider,omitempty"`
	PromptTokens     *int64   `json:"promptTokens,omitempty"`
	CompletionTokens *int64   `json:"completionTokens,omitempty"`
	Cost             *float64 `json:"cost,omitempty"`
	CreatedAt        *int64   `json:"createdAt,omitempty"`
	UpdatedAt        *int64   `json:"updatedAt,omitempty"`
}

type Message struct {
	ID               string  `json:"id"`
	SessionID        string  `json:"sessionId"`
	Role             string  `json:"role"`
	TextContent      string  `json:"textContent"`
	Model            *string `json:"model,omitempty"`
	PromptTokens     *int64  `json:"promptTokens,omitempty"`
	CompletionTokens *int64  `json:"completionTokens,omitempty"`
	DurationMs       *int64  `json:"durationMs,omitempty"`
	CreatedAt        *int64  `json:"createdAt,omitempty"`
	CompletedAt      *int64  `json:"completedAt,omitempty"`
}

func EnsureDb(dbPath string) (bool, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return false, err
	}

	_, err := os.Stat(dbPath)
	created := os.IsNotExist(err)

	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_foreign_keys=ON")
	if err != nil {
		return false, err
	}
	defer db.Close()

	if _, err := db.Exec(schemaSQL); err != nil {
		return false, err
	}

	return created, nil
}

func Open(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_foreign_keys=ON")
	if err != nil {
		return nil, err
	}

	upsertSession, err := db.Prepare(upsertSessionSQL)
	if err != nil {
		db.Close()
		return nil, err
	}

	upsertMessage, err := db.Prepare(upsertMessageSQL)
	if err != nil {
		upsertSession.Close()
		db.Close()
		return nil, err
	}

	return &Store{
		db:            db,
		upsertSession: upsertSession,
		upsertMessage: upsertMessage,
	}, nil
}

func (s *Store) Close() error {
	s.upsertSession.Close()
	s.upsertMessage.Close()
	return s.db.Close()
}

func (s *Store) UpsertSession(session *Session) error {
	title := "Untitled Session"
	if session.Title != nil {
		title = *session.Title
	}
	promptTokens := int64(0)
	if session.PromptTokens != nil {
		promptTokens = *session.PromptTokens
	}
	completionTokens := int64(0)
	if session.CompletionTokens != nil {
		completionTokens = *session.CompletionTokens
	}
	cost := float64(0)
	if session.Cost != nil {
		cost = *session.Cost
	}

	_, err := s.upsertSession.Exec(
		session.ID,
		title,
		session.ProjectPath,
		session.ProjectName,
		session.Model,
		session.Provider,
		promptTokens,
		completionTokens,
		cost,
		session.CreatedAt,
		session.UpdatedAt,
	)
	return err
}

func (s *Store) UpsertMessage(msg *Message) error {
	promptTokens := int64(0)
	if msg.PromptTokens != nil {
		promptTokens = *msg.PromptTokens
	}
	completionTokens := int64(0)
	if msg.CompletionTokens != nil {
		completionTokens = *msg.CompletionTokens
	}

	_, err := s.upsertMessage.Exec(
		msg.ID,
		msg.SessionID,
		msg.Role,
		msg.TextContent,
		msg.Model,
		promptTokens,
		completionTokens,
		msg.DurationMs,
		msg.CreatedAt,
		msg.CompletedAt,
	)
	return err
}
