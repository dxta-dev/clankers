package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

type QueryResult map[string]interface{}

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

func (s *Store) GetSessions(limit int) ([]Session, error) {
	query := `SELECT id, title, project_path, project_name, model, provider,
		prompt_tokens, completion_tokens, cost, created_at, updated_at
		FROM sessions ORDER BY created_at DESC`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var s Session
		var title sql.NullString
		var projectPath sql.NullString
		var projectName sql.NullString
		var model sql.NullString
		var provider sql.NullString
		var promptTokens sql.NullInt64
		var completionTokens sql.NullInt64
		var cost sql.NullFloat64
		var createdAt sql.NullInt64
		var updatedAt sql.NullInt64

		err := rows.Scan(
			&s.ID, &title, &projectPath, &projectName, &model, &provider,
			&promptTokens, &completionTokens, &cost, &createdAt, &updatedAt,
		)
		if err != nil {
			return nil, err
		}

		if title.Valid {
			s.Title = &title.String
		}
		if projectPath.Valid {
			s.ProjectPath = &projectPath.String
		}
		if projectName.Valid {
			s.ProjectName = &projectName.String
		}
		if model.Valid {
			s.Model = &model.String
		}
		if provider.Valid {
			s.Provider = &provider.String
		}
		if promptTokens.Valid {
			s.PromptTokens = &promptTokens.Int64
		}
		if completionTokens.Valid {
			s.CompletionTokens = &completionTokens.Int64
		}
		if cost.Valid {
			s.Cost = &cost.Float64
		}
		if createdAt.Valid {
			s.CreatedAt = &createdAt.Int64
		}
		if updatedAt.Valid {
			s.UpdatedAt = &updatedAt.Int64
		}

		sessions = append(sessions, s)
	}

	return sessions, rows.Err()
}

func (s *Store) GetSessionByID(id string) (*Session, []Message, error) {
	var session Session
	var title sql.NullString
	var projectPath sql.NullString
	var projectName sql.NullString
	var model sql.NullString
	var provider sql.NullString
	var promptTokens sql.NullInt64
	var completionTokens sql.NullInt64
	var cost sql.NullFloat64
	var createdAt sql.NullInt64
	var updatedAt sql.NullInt64

	err := s.db.QueryRow(`
		SELECT id, title, project_path, project_name, model, provider,
			prompt_tokens, completion_tokens, cost, created_at, updated_at
		FROM sessions WHERE id = ?`, id).Scan(
		&session.ID, &title, &projectPath, &projectName, &model, &provider,
		&promptTokens, &completionTokens, &cost, &createdAt, &updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil, fmt.Errorf("session not found: %s", id)
	}
	if err != nil {
		return nil, nil, err
	}

	if title.Valid {
		session.Title = &title.String
	}
	if projectPath.Valid {
		session.ProjectPath = &projectPath.String
	}
	if projectName.Valid {
		session.ProjectName = &projectName.String
	}
	if model.Valid {
		session.Model = &model.String
	}
	if provider.Valid {
		session.Provider = &provider.String
	}
	if promptTokens.Valid {
		session.PromptTokens = &promptTokens.Int64
	}
	if completionTokens.Valid {
		session.CompletionTokens = &completionTokens.Int64
	}
	if cost.Valid {
		session.Cost = &cost.Float64
	}
	if createdAt.Valid {
		session.CreatedAt = &createdAt.Int64
	}
	if updatedAt.Valid {
		session.UpdatedAt = &updatedAt.Int64
	}

	messages, err := s.GetMessages(id)
	if err != nil {
		return nil, nil, err
	}

	return &session, messages, nil
}

func (s *Store) GetMessages(sessionID string) ([]Message, error) {
	rows, err := s.db.Query(`
		SELECT id, session_id, role, text_content, model,
			prompt_tokens, completion_tokens, duration_ms, created_at, completed_at
		FROM messages WHERE session_id = ? ORDER BY created_at ASC`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var m Message
		var model sql.NullString
		var promptTokens sql.NullInt64
		var completionTokens sql.NullInt64
		var durationMs sql.NullInt64
		var createdAt sql.NullInt64
		var completedAt sql.NullInt64

		err := rows.Scan(
			&m.ID, &m.SessionID, &m.Role, &m.TextContent, &model,
			&promptTokens, &completionTokens, &durationMs, &createdAt, &completedAt,
		)
		if err != nil {
			return nil, err
		}

		if model.Valid {
			m.Model = &model.String
		}
		if promptTokens.Valid {
			m.PromptTokens = &promptTokens.Int64
		}
		if completionTokens.Valid {
			m.CompletionTokens = &completionTokens.Int64
		}
		if durationMs.Valid {
			m.DurationMs = &durationMs.Int64
		}
		if createdAt.Valid {
			m.CreatedAt = &createdAt.Int64
		}
		if completedAt.Valid {
			m.CompletedAt = &completedAt.Int64
		}

		messages = append(messages, m)
	}

	return messages, rows.Err()
}

func (s *Store) ExecuteQuery(sql string) ([]QueryResult, error) {
	writeKeywords := []string{
		"INSERT", "UPDATE", "DELETE", "DROP", "CREATE", "ALTER", "TRUNCATE",
		"REPLACE", "MERGE", "UPSERT", "ATTACH", "DETACH", "REINDEX", "VACUUM",
		"PRAGMA", "BEGIN", "COMMIT", "ROLLBACK", "SAVEPOINT", "RELEASE",
	}

	upperSQL := strings.ToUpper(strings.TrimSpace(sql))
	for _, keyword := range writeKeywords {
		if strings.HasPrefix(upperSQL, keyword) || strings.Contains(upperSQL, " "+keyword+" ") {
			return nil, fmt.Errorf("write operations are not allowed from the CLI: %s statements are blocked", keyword)
		}
	}

	if !strings.HasPrefix(upperSQL, "SELECT") && !strings.HasPrefix(upperSQL, "WITH") {
		return nil, fmt.Errorf("only SELECT queries are allowed from the CLI")
	}

	rows, err := s.db.Query(sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []QueryResult
	for rows.Next() {
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		row := make(QueryResult)
		for i, col := range columns {
			val := values[i]
			switch v := val.(type) {
			case []byte:
				row[col] = string(v)
			case nil:
				row[col] = nil
			default:
				row[col] = v
			}
		}
		results = append(results, row)
	}

	return results, rows.Err()
}

func (s *Store) GetTableSchema(tableName string) ([]string, error) {
	rows, err := s.db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var cid int
		var name string
		var dataType string
		var notNull int
		var dfltValue sql.NullString
		var pk int

		if err := rows.Scan(&cid, &name, &dataType, &notNull, &dfltValue, &pk); err != nil {
			return nil, err
		}
		columns = append(columns, name)
	}

	return columns, rows.Err()
}

func (s *Store) SuggestColumnNames(tableName string, input string) ([]string, error) {
	columns, err := s.GetTableSchema(tableName)
	if err != nil {
		return nil, err
	}

	var suggestions []string
	lowerInput := strings.ToLower(input)
	for _, col := range columns {
		lowerCol := strings.ToLower(col)
		if strings.Contains(lowerCol, lowerInput) || strings.Contains(lowerInput, lowerCol) {
			suggestions = append(suggestions, col)
		}
	}

	return suggestions, nil
}
