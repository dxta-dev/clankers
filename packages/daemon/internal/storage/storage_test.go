package storage

import (
	"os"
	"path/filepath"
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

		// Verify file was created
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			t.Error("expected database file to exist")
		}

		// Clean up
		os.Remove(dbPath)
	})

	t.Run("returns false if DB already exists", func(t *testing.T) {
		// First call - create the DB
		_, err := EnsureDb(dbPath)
		if err != nil {
			t.Fatalf("expected no error creating DB, got %v", err)
		}

		// Second call - DB exists
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

		// Verify all directories were created
		if _, err := os.Stat(filepath.Dir(nestedDbPath)); os.IsNotExist(err) {
			t.Error("expected parent directories to be created")
		}

		// Verify file was created
		if _, err := os.Stat(nestedDbPath); os.IsNotExist(err) {
			t.Error("expected database file to exist")
		}
	})
}

func TestEnsureDbExists(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "existing.db")

	// Create the database first
	_, err := EnsureDb(dbPath)
	if err != nil {
		t.Fatalf("failed to create initial database: %v", err)
	}

	// Now call EnsureDb again
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

	// Ensure DB exists first
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

	// Clean up
	store.Close()
}

func TestClose(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "close_test.db")

	// Ensure DB exists and open it
	_, err := EnsureDb(dbPath)
	if err != nil {
		t.Fatalf("failed to ensure DB: %v", err)
	}

	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	// Close should succeed without error
	err = store.Close()
	if err != nil {
		t.Errorf("expected no error closing database, got %v", err)
	}

	// Verify the prepared statements are closed by trying to use them
	// (This would fail if not properly closed, but we can't easily test that here)
}

func TestStoreUpsertSession(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "upsert_session_test.db")

	// Ensure DB exists and open it
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

	// Ensure DB exists and open it
	_, err := EnsureDb(dbPath)
	if err != nil {
		t.Fatalf("failed to ensure DB: %v", err)
	}

	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer store.Close()

	// First, create a session to reference
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
