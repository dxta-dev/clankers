package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureDb(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	t.Run("creates DB if not exists", func(t *testing.T) {
		created, err := EnsureDb(dbPath)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if !created {
			t.Error("expected created to be true for new database")
		}

		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			t.Error("expected database file to exist")
		}

		os.Remove(dbPath)
	})

	t.Run("returns false if DB already exists", func(t *testing.T) {
		_, err := EnsureDb(dbPath)
		if err != nil {
			t.Fatalf("expected no error creating DB, got %v", err)
		}

		created, err := EnsureDb(dbPath)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if created {
			t.Error("expected created to be false for existing database")
		}
	})

	t.Run("creates parent directories if needed", func(t *testing.T) {
		nestedDbPath := filepath.Join(tmpDir, "nested", "deep", "test.db")

		_, err := EnsureDb(nestedDbPath)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if _, err := os.Stat(filepath.Dir(nestedDbPath)); os.IsNotExist(err) {
			t.Error("expected parent directories to be created")
		}

		if _, err := os.Stat(nestedDbPath); os.IsNotExist(err) {
			t.Error("expected database file to exist")
		}
	})
}

func TestEnsureDbExists(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "existing.db")

	_, err := EnsureDb(dbPath)
	if err != nil {
		t.Fatalf("failed to create initial database: %v", err)
	}

	created, err := EnsureDb(dbPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if created {
		t.Error("expected created to be false when DB already exists")
	}
}

func TestOpen(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "open_test.db")

	_, err := EnsureDb(dbPath)
	if err != nil {
		t.Fatalf("failed to ensure DB: %v", err)
	}

	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("expected no error opening database, got %v", err)
	}

	if store == nil {
		t.Fatal("expected store to not be nil")
	}

	if store.db == nil {
		t.Error("expected store.db to not be nil")
	}

	if store.upsertSession == nil {
		t.Error("expected store.upsertSession to not be nil")
	}

	if store.upsertMessage == nil {
		t.Error("expected store.upsertMessage to not be nil")
	}

	store.Close()
}

func TestClose(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "close_test.db")

	_, err := EnsureDb(dbPath)
	if err != nil {
		t.Fatalf("failed to ensure DB: %v", err)
	}

	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	err = store.Close()
	if err != nil {
		t.Errorf("expected no error closing database, got %v", err)
	}

}

func TestStoreUpsertSession(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "upsert_session_test.db")

	_, err := EnsureDb(dbPath)
	if err != nil {
		t.Fatalf("failed to ensure DB: %v", err)
	}

	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer store.Close()

	t.Run("inserts new session", func(t *testing.T) {
		session := &Session{
			ID:               "session-1",
			Title:            strPtr("Test Session"),
			ProjectPath:      strPtr("/test/path"),
			ProjectName:      strPtr("test-project"),
			Model:            strPtr("gpt-4"),
			Provider:         strPtr("openai"),
			PromptTokens:     int64Ptr(100),
			CompletionTokens: int64Ptr(50),
			Cost:             float64Ptr(0.005),
			CreatedAt:        int64Ptr(1234567890),
			UpdatedAt:        int64Ptr(1234567891),
		}

		err := store.UpsertSession(session)
		if err != nil {
			t.Fatalf("expected no error upserting session, got %v", err)
		}
	})

	t.Run("updates existing session", func(t *testing.T) {
		session := &Session{
			ID:    "session-1",
			Title: strPtr("Updated Title"),
		}

		err := store.UpsertSession(session)
		if err != nil {
			t.Fatalf("expected no error updating session, got %v", err)
		}
	})
}

func TestStoreUpsertMessage(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "upsert_message_test.db")

	_, err := EnsureDb(dbPath)
	if err != nil {
		t.Fatalf("failed to ensure DB: %v", err)
	}

	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer store.Close()

	session := &Session{
		ID: "msg-test-session",
	}
	if err := store.UpsertSession(session); err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	t.Run("inserts new message", func(t *testing.T) {
		msg := &Message{
			ID:               "msg-1",
			SessionID:        "msg-test-session",
			Role:             "user",
			TextContent:      "Hello, world!",
			Model:            strPtr("gpt-4"),
			PromptTokens:     int64Ptr(10),
			CompletionTokens: int64Ptr(20),
			DurationMs:       int64Ptr(500),
			CreatedAt:        int64Ptr(1234567890),
			CompletedAt:      int64Ptr(1234567895),
		}

		err := store.UpsertMessage(msg)
		if err != nil {
			t.Fatalf("expected no error upserting message, got %v", err)
		}
	})

	t.Run("updates existing message", func(t *testing.T) {
		msg := &Message{
			ID:          "msg-1",
			SessionID:   "msg-test-session",
			Role:        "user",
			TextContent: "Updated message content",
		}

		err := store.UpsertMessage(msg)
		if err != nil {
			t.Fatalf("expected no error updating message, got %v", err)
		}
	})
}

func TestGetSessions(t *testing.T) {
	store := createStore(t)

	sessionOneCreatedAt := int64(100)
	sessionTwoCreatedAt := int64(200)
	if err := store.UpsertSession(&Session{
		ID:        "session-1",
		Title:     strPtr("First"),
		CreatedAt: &sessionOneCreatedAt,
	}); err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	if err := store.UpsertSession(&Session{
		ID:        "session-2",
		Title:     strPtr("Second"),
		CreatedAt: &sessionTwoCreatedAt,
	}); err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	t.Run("returns sessions ordered by created_at desc", func(t *testing.T) {
		sessions, err := store.GetSessions(0)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(sessions) != 2 {
			t.Fatalf("expected 2 sessions, got %d", len(sessions))
		}
		if sessions[0].ID != "session-2" {
			t.Errorf("expected session-2 first, got %s", sessions[0].ID)
		}
	})

	t.Run("respects limit", func(t *testing.T) {
		sessions, err := store.GetSessions(1)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(sessions) != 1 {
			t.Fatalf("expected 1 session, got %d", len(sessions))
		}
		if sessions[0].ID != "session-2" {
			t.Errorf("expected session-2 first, got %s", sessions[0].ID)
		}
	})
}

func TestGetSessionByID(t *testing.T) {
	store := createStore(t)

	createdAt := int64(300)
	if err := store.UpsertSession(&Session{
		ID:        "session-abc",
		Title:     strPtr("Find Me"),
		CreatedAt: &createdAt,
	}); err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	messageCreatedAt := int64(301)
	if err := store.UpsertMessage(&Message{
		ID:          "msg-abc",
		SessionID:   "session-abc",
		Role:        "assistant",
		TextContent: "Hello",
		CreatedAt:   &messageCreatedAt,
	}); err != nil {
		t.Fatalf("failed to create message: %v", err)
	}

	t.Run("returns session and messages", func(t *testing.T) {
		session, messages, err := store.GetSessionByID("session-abc")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if session == nil || session.ID != "session-abc" {
			t.Fatalf("expected session-abc, got %+v", session)
		}
		if len(messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(messages))
		}
		if messages[0].ID != "msg-abc" {
			t.Errorf("expected msg-abc, got %s", messages[0].ID)
		}
	})

	t.Run("returns error when missing", func(t *testing.T) {
		_, _, err := store.GetSessionByID("missing-session")
		if err == nil {
			t.Fatal("expected error for missing session")
		}
		if !strings.Contains(err.Error(), "session not found") {
			t.Fatalf("expected not found error, got %v", err)
		}
	})
}

func TestGetMessages(t *testing.T) {
	store := createStore(t)

	if err := store.UpsertSession(&Session{ID: "session-msgs"}); err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	firstCreated := int64(10)
	secondCreated := int64(20)
	if err := store.UpsertMessage(&Message{
		ID:          "msg-1",
		SessionID:   "session-msgs",
		Role:        "user",
		TextContent: "First",
		CreatedAt:   &firstCreated,
	}); err != nil {
		t.Fatalf("failed to create message: %v", err)
	}
	if err := store.UpsertMessage(&Message{
		ID:          "msg-2",
		SessionID:   "session-msgs",
		Role:        "assistant",
		TextContent: "Second",
		CreatedAt:   &secondCreated,
	}); err != nil {
		t.Fatalf("failed to create message: %v", err)
	}

	messages, err := store.GetMessages("session-msgs")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}
	if messages[0].ID != "msg-1" || messages[1].ID != "msg-2" {
		t.Fatalf("expected messages ordered by created_at asc")
	}
}

func TestExecuteQuery(t *testing.T) {
	store := createStore(t)

	createdAt := int64(400)
	if err := store.UpsertSession(&Session{
		ID:        "session-query",
		Title:     strPtr("Query Me"),
		CreatedAt: &createdAt,
	}); err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	results, err := store.ExecuteQuery("SELECT id, title FROM sessions WHERE id = 'session-query'")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0]["id"] != "session-query" {
		t.Errorf("expected id session-query, got %v", results[0]["id"])
	}
	if results[0]["title"] != "Query Me" {
		t.Errorf("expected title Query Me, got %v", results[0]["title"])
	}
}

func TestExecuteQueryBlocksWrites(t *testing.T) {
	store := createStore(t)

	keywords := []string{
		"INSERT", "UPDATE", "DELETE", "DROP", "CREATE", "ALTER", "TRUNCATE",
		"REPLACE", "MERGE", "UPSERT", "ATTACH", "DETACH", "REINDEX", "VACUUM",
		"PRAGMA", "BEGIN", "COMMIT", "ROLLBACK", "SAVEPOINT", "RELEASE",
	}

	for _, keyword := range keywords {
		_, err := store.ExecuteQuery(keyword + " sessions")
		if err == nil {
			t.Fatalf("expected error for %s", keyword)
		}
		if !strings.Contains(err.Error(), keyword) {
			t.Fatalf("expected error to mention %s, got %v", keyword, err)
		}
	}
}

func TestGetTableSchema(t *testing.T) {
	store := createStore(t)

	columns, err := store.GetTableSchema("sessions")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(columns) == 0 {
		t.Fatal("expected columns to be returned")
	}

	columnSet := make(map[string]bool)
	for _, col := range columns {
		columnSet[col] = true
	}

	for _, col := range []string{"id", "title", "project_path", "created_at"} {
		if !columnSet[col] {
			t.Fatalf("expected column %s to exist", col)
		}
	}
}

func TestSuggestColumnNames(t *testing.T) {
	store := createStore(t)

	suggestions, err := store.SuggestColumnNames("sessions", "proj")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	suggestionSet := make(map[string]bool)
	for _, suggestion := range suggestions {
		suggestionSet[suggestion] = true
	}

	if !suggestionSet["project_path"] || !suggestionSet["project_name"] {
		t.Fatalf("expected project_path and project_name suggestions, got %v", suggestions)
	}
}

func createStore(t *testing.T) *Store {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "query_test.db")

	_, err := EnsureDb(dbPath)
	if err != nil {
		t.Fatalf("failed to ensure DB: %v", err)
	}

	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	t.Cleanup(func() {
		store.Close()
	})

	return store
}

// Helper functions
func strPtr(s string) *string {
	return &s
}

func int64Ptr(i int64) *int64 {
	return &i
}

func float64Ptr(f float64) *float64 {
	return &f
}
