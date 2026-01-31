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
	source TEXT,
	status TEXT,
	prompt_tokens INTEGER,
	completion_tokens INTEGER,
	cost REAL,
	message_count INTEGER,
	tool_call_count INTEGER,
	permission_mode TEXT,
	created_at INTEGER,
	updated_at INTEGER,
	ended_at INTEGER
);

CREATE TABLE IF NOT EXISTS messages (
	id TEXT PRIMARY KEY,
	session_id TEXT,
	role TEXT,
	text_content TEXT,
	model TEXT,
	source TEXT,
	prompt_tokens INTEGER,
	completion_tokens INTEGER,
	duration_ms INTEGER,
	created_at INTEGER,
	completed_at INTEGER,
	FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS tools (
	id TEXT PRIMARY KEY,
	session_id TEXT NOT NULL,
	message_id TEXT,
	tool_name TEXT NOT NULL,
	tool_input TEXT,
	tool_output TEXT,
	file_path TEXT,
	success BOOLEAN,
	error_message TEXT,
	duration_ms INTEGER,
	created_at INTEGER NOT NULL,
	FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_tools_session ON tools(session_id);
CREATE INDEX IF NOT EXISTS idx_tools_name ON tools(tool_name);
CREATE INDEX IF NOT EXISTS idx_tools_file ON tools(file_path);

CREATE TABLE IF NOT EXISTS session_errors (
	id TEXT PRIMARY KEY,
	session_id TEXT NOT NULL,
	error_type TEXT,
	error_message TEXT,
	created_at INTEGER NOT NULL,
	FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_session_errors_session ON session_errors(session_id);

CREATE TABLE IF NOT EXISTS compaction_events (
	id TEXT PRIMARY KEY,
	session_id TEXT NOT NULL,
	tokens_before INTEGER,
	tokens_after INTEGER,
	messages_before INTEGER,
	messages_after INTEGER,
	created_at INTEGER NOT NULL,
	FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_compaction_session ON compaction_events(session_id);
`

const upsertSessionSQL = `
INSERT INTO sessions (
	id, title, project_path, project_name, model, provider, source, status,
	prompt_tokens, completion_tokens, cost, message_count, tool_call_count,
	permission_mode, created_at, updated_at, ended_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	title = CASE WHEN excluded.title IS NOT NULL AND excluded.title != ''
	             THEN excluded.title ELSE sessions.title END,
	model = CASE WHEN excluded.model IS NOT NULL AND excluded.model != ''
	             THEN excluded.model ELSE sessions.model END,
	provider = CASE WHEN excluded.provider IS NOT NULL AND excluded.provider != ''
	                THEN excluded.provider ELSE sessions.provider END,
	source = CASE WHEN excluded.source IS NOT NULL AND excluded.source != ''
	              THEN excluded.source ELSE sessions.source END,
	status = CASE WHEN excluded.status IS NOT NULL AND excluded.status != ''
	              THEN excluded.status ELSE sessions.status END,
	permission_mode = CASE WHEN excluded.permission_mode IS NOT NULL AND excluded.permission_mode != ''
	                       THEN excluded.permission_mode ELSE sessions.permission_mode END,
	created_at = COALESCE(sessions.created_at, excluded.created_at),
	project_path = excluded.project_path,
	project_name = excluded.project_name,
	prompt_tokens = excluded.prompt_tokens,
	completion_tokens = excluded.completion_tokens,
	cost = excluded.cost,
	message_count = COALESCE(excluded.message_count, sessions.message_count),
	tool_call_count = COALESCE(excluded.tool_call_count, sessions.tool_call_count),
	updated_at = excluded.updated_at,
	ended_at = COALESCE(excluded.ended_at, sessions.ended_at);
`

const upsertMessageSQL = `
INSERT INTO messages (
	id, session_id, role, text_content, model, source,
	prompt_tokens, completion_tokens, duration_ms,
	created_at, completed_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	text_content = CASE WHEN excluded.text_content IS NOT NULL AND excluded.text_content != ''
	                    THEN excluded.text_content ELSE messages.text_content END,
	source = CASE WHEN excluded.source IS NOT NULL AND excluded.source != ''
	              THEN excluded.source ELSE messages.source END,
	created_at = COALESCE(messages.created_at, excluded.created_at),
	session_id = excluded.session_id,
	role = excluded.role,
	model = excluded.model,
	prompt_tokens = excluded.prompt_tokens,
	completion_tokens = excluded.completion_tokens,
	duration_ms = excluded.duration_ms,
	completed_at = excluded.completed_at;
`

const upsertToolSQL = `
INSERT INTO tools (
	id, session_id, message_id, tool_name, tool_input, tool_output,
	file_path, success, error_message, duration_ms, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	tool_output = excluded.tool_output,
	success = excluded.success,
	error_message = excluded.error_message,
	duration_ms = excluded.duration_ms,
	message_id = COALESCE(excluded.message_id, tools.message_id);
`

const upsertSessionErrorSQL = `
INSERT INTO session_errors (
	id, session_id, error_type, error_message, created_at
) VALUES (?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	error_type = excluded.error_type,
	error_message = excluded.error_message;
`

const upsertCompactionEventSQL = `
INSERT INTO compaction_events (
	id, session_id, tokens_before, tokens_after, messages_before, messages_after, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	tokens_before = excluded.tokens_before,
	tokens_after = excluded.tokens_after,
	messages_before = excluded.messages_before,
	messages_after = excluded.messages_after;
`

type Store struct {
	db                 *sql.DB
	upsertSession      *sql.Stmt
	upsertMessage      *sql.Stmt
	upsertTool         *sql.Stmt
	upsertSessionError *sql.Stmt
	upsertCompaction   *sql.Stmt
}

type Session struct {
	ID               string   `json:"id"`
	Title            *string  `json:"title,omitempty"`
	ProjectPath      *string  `json:"projectPath,omitempty"`
	ProjectName      *string  `json:"projectName,omitempty"`
	Model            *string  `json:"model,omitempty"`
	Provider         *string  `json:"provider,omitempty"`
	Source           *string  `json:"source,omitempty"`
	Status           *string  `json:"status,omitempty"`
	PromptTokens     *int64   `json:"promptTokens,omitempty"`
	CompletionTokens *int64   `json:"completionTokens,omitempty"`
	Cost             *float64 `json:"cost,omitempty"`
	MessageCount     *int64   `json:"messageCount,omitempty"`
	ToolCallCount    *int64   `json:"toolCallCount,omitempty"`
	PermissionMode   *string  `json:"permissionMode,omitempty"`
	CreatedAt        *int64   `json:"createdAt,omitempty"`
	UpdatedAt        *int64   `json:"updatedAt,omitempty"`
	EndedAt          *int64   `json:"endedAt,omitempty"`
}

type Message struct {
	ID               string  `json:"id"`
	SessionID        string  `json:"sessionId"`
	Role             string  `json:"role"`
	TextContent      string  `json:"textContent"`
	Model            *string `json:"model,omitempty"`
	Source           *string `json:"source,omitempty"`
	PromptTokens     *int64  `json:"promptTokens,omitempty"`
	CompletionTokens *int64  `json:"completionTokens,omitempty"`
	DurationMs       *int64  `json:"durationMs,omitempty"`
	CreatedAt        *int64  `json:"createdAt,omitempty"`
	CompletedAt      *int64  `json:"completedAt,omitempty"`
}

type Tool struct {
	ID           string  `json:"id"`
	SessionID    string  `json:"sessionId"`
	MessageID    *string `json:"messageId,omitempty"`
	ToolName     string  `json:"toolName"`
	ToolInput    *string `json:"toolInput,omitempty"`
	ToolOutput   *string `json:"toolOutput,omitempty"`
	FilePath     *string `json:"filePath,omitempty"`
	Success      *bool   `json:"success,omitempty"`
	ErrorMessage *string `json:"errorMessage,omitempty"`
	DurationMs   *int64  `json:"durationMs,omitempty"`
	CreatedAt    int64   `json:"createdAt"`
}

type SessionError struct {
	ID           string  `json:"id"`
	SessionID    string  `json:"sessionId"`
	ErrorType    *string `json:"errorType,omitempty"`
	ErrorMessage *string `json:"errorMessage,omitempty"`
	CreatedAt    int64   `json:"createdAt"`
}

type CompactionEvent struct {
	ID             string `json:"id"`
	SessionID      string `json:"sessionId"`
	TokensBefore   *int64 `json:"tokensBefore,omitempty"`
	TokensAfter    *int64 `json:"tokensAfter,omitempty"`
	MessagesBefore *int64 `json:"messagesBefore,omitempty"`
	MessagesAfter  *int64 `json:"messagesAfter,omitempty"`
	CreatedAt      int64  `json:"createdAt"`
}

type QueryResult map[string]interface{}

var sqlitePragmas = []string{
	"PRAGMA journal_mode = WAL;",
	"PRAGMA foreign_keys = ON;",
	"PRAGMA busy_timeout = 5000;",
}

func configureDb(db *sql.DB) error {
	db.SetMaxOpenConns(1)
	for _, pragma := range sqlitePragmas {
		if _, err := db.Exec(pragma); err != nil {
			return err
		}
	}
	return nil
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

	if err := configureDb(db); err != nil {
		return false, err
	}

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

	if err := configureDb(db); err != nil {
		db.Close()
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

	upsertTool, err := db.Prepare(upsertToolSQL)
	if err != nil {
		upsertSession.Close()
		upsertMessage.Close()
		db.Close()
		return nil, err
	}

	upsertSessionError, err := db.Prepare(upsertSessionErrorSQL)
	if err != nil {
		upsertSession.Close()
		upsertMessage.Close()
		upsertTool.Close()
		db.Close()
		return nil, err
	}

	upsertCompaction, err := db.Prepare(upsertCompactionEventSQL)
	if err != nil {
		upsertSession.Close()
		upsertMessage.Close()
		upsertTool.Close()
		upsertSessionError.Close()
		db.Close()
		return nil, err
	}

	return &Store{
		db:                 db,
		upsertSession:      upsertSession,
		upsertMessage:      upsertMessage,
		upsertTool:         upsertTool,
		upsertSessionError: upsertSessionError,
		upsertCompaction:   upsertCompaction,
	}, nil
}

func (s *Store) Close() error {
	s.upsertSession.Close()
	s.upsertMessage.Close()
	s.upsertTool.Close()
	s.upsertSessionError.Close()
	s.upsertCompaction.Close()
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
	messageCount := int64(0)
	if session.MessageCount != nil {
		messageCount = *session.MessageCount
	}
	toolCallCount := int64(0)
	if session.ToolCallCount != nil {
		toolCallCount = *session.ToolCallCount
	}

	_, err := s.upsertSession.Exec(
		session.ID,
		title,
		session.ProjectPath,
		session.ProjectName,
		session.Model,
		session.Provider,
		session.Source,
		session.Status,
		promptTokens,
		completionTokens,
		cost,
		messageCount,
		toolCallCount,
		session.PermissionMode,
		session.CreatedAt,
		session.UpdatedAt,
		session.EndedAt,
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
		msg.Source,
		promptTokens,
		completionTokens,
		msg.DurationMs,
		msg.CreatedAt,
		msg.CompletedAt,
	)
	return err
}

func (s *Store) UpsertTool(tool *Tool) error {
	_, err := s.upsertTool.Exec(
		tool.ID,
		tool.SessionID,
		tool.MessageID,
		tool.ToolName,
		tool.ToolInput,
		tool.ToolOutput,
		tool.FilePath,
		tool.Success,
		tool.ErrorMessage,
		tool.DurationMs,
		tool.CreatedAt,
	)
	return err
}

func (s *Store) UpsertSessionError(errRecord *SessionError) error {
	_, err := s.upsertSessionError.Exec(
		errRecord.ID,
		errRecord.SessionID,
		errRecord.ErrorType,
		errRecord.ErrorMessage,
		errRecord.CreatedAt,
	)
	return err
}

func (s *Store) UpsertCompactionEvent(event *CompactionEvent) error {
	_, err := s.upsertCompaction.Exec(
		event.ID,
		event.SessionID,
		event.TokensBefore,
		event.TokensAfter,
		event.MessagesBefore,
		event.MessagesAfter,
		event.CreatedAt,
	)
	return err
}

func (s *Store) GetSessions(limit int) ([]Session, error) {
	query := `SELECT id, title, project_path, project_name, model, provider, source, status,
		prompt_tokens, completion_tokens, cost, message_count, tool_call_count,
		permission_mode, created_at, updated_at, ended_at
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
		var source sql.NullString
		var status sql.NullString
		var promptTokens sql.NullInt64
		var completionTokens sql.NullInt64
		var cost sql.NullFloat64
		var messageCount sql.NullInt64
		var toolCallCount sql.NullInt64
		var permissionMode sql.NullString
		var createdAt sql.NullInt64
		var updatedAt sql.NullInt64
		var endedAt sql.NullInt64

		err := rows.Scan(
			&s.ID, &title, &projectPath, &projectName, &model, &provider, &source, &status,
			&promptTokens, &completionTokens, &cost, &messageCount, &toolCallCount,
			&permissionMode, &createdAt, &updatedAt, &endedAt,
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
		if source.Valid {
			s.Source = &source.String
		}
		if status.Valid {
			s.Status = &status.String
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
		if messageCount.Valid {
			s.MessageCount = &messageCount.Int64
		}
		if toolCallCount.Valid {
			s.ToolCallCount = &toolCallCount.Int64
		}
		if permissionMode.Valid {
			s.PermissionMode = &permissionMode.String
		}
		if createdAt.Valid {
			s.CreatedAt = &createdAt.Int64
		}
		if updatedAt.Valid {
			s.UpdatedAt = &updatedAt.Int64
		}
		if endedAt.Valid {
			s.EndedAt = &endedAt.Int64
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
	var source sql.NullString
	var status sql.NullString
	var promptTokens sql.NullInt64
	var completionTokens sql.NullInt64
	var cost sql.NullFloat64
	var messageCount sql.NullInt64
	var toolCallCount sql.NullInt64
	var permissionMode sql.NullString
	var createdAt sql.NullInt64
	var updatedAt sql.NullInt64
	var endedAt sql.NullInt64

	err := s.db.QueryRow(`
		SELECT id, title, project_path, project_name, model, provider, source, status,
			prompt_tokens, completion_tokens, cost, message_count, tool_call_count,
			permission_mode, created_at, updated_at, ended_at
		FROM sessions WHERE id = ?`, id).Scan(
		&session.ID, &title, &projectPath, &projectName, &model, &provider, &source, &status,
		&promptTokens, &completionTokens, &cost, &messageCount, &toolCallCount,
		&permissionMode, &createdAt, &updatedAt, &endedAt,
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
	if source.Valid {
		session.Source = &source.String
	}
	if status.Valid {
		session.Status = &status.String
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
	if messageCount.Valid {
		session.MessageCount = &messageCount.Int64
	}
	if toolCallCount.Valid {
		session.ToolCallCount = &toolCallCount.Int64
	}
	if permissionMode.Valid {
		session.PermissionMode = &permissionMode.String
	}
	if createdAt.Valid {
		session.CreatedAt = &createdAt.Int64
	}
	if updatedAt.Valid {
		session.UpdatedAt = &updatedAt.Int64
	}
	if endedAt.Valid {
		session.EndedAt = &endedAt.Int64
	}

	messages, err := s.GetMessages(id)
	if err != nil {
		return nil, nil, err
	}

	return &session, messages, nil
}

func (s *Store) GetMessages(sessionID string) ([]Message, error) {
	rows, err := s.db.Query(`
		SELECT id, session_id, role, text_content, model, source,
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
		var source sql.NullString
		var promptTokens sql.NullInt64
		var completionTokens sql.NullInt64
		var durationMs sql.NullInt64
		var createdAt sql.NullInt64
		var completedAt sql.NullInt64

		err := rows.Scan(
			&m.ID, &m.SessionID, &m.Role, &m.TextContent, &model, &source,
			&promptTokens, &completionTokens, &durationMs, &createdAt, &completedAt,
		)
		if err != nil {
			return nil, err
		}

		if model.Valid {
			m.Model = &model.String
		}
		if source.Valid {
			m.Source = &source.String
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
