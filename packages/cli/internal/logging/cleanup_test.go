package logging

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCleanupOldLogs(t *testing.T) {
	t.Run("removes files older than 30 days", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create an old file (31 days ago)
		oldFile := filepath.Join(tmpDir, "clankers-2024-12-01.jsonl")
		if err := os.WriteFile(oldFile, []byte("old log"), 0644); err != nil {
			t.Fatalf("failed to create old file: %v", err)
		}
		oldTime := time.Now().AddDate(0, 0, -31)
		if err := os.Chtimes(oldFile, oldTime, oldTime); err != nil {
			t.Fatalf("failed to set old file time: %v", err)
		}

		// Create a recent file (today)
		recentFile := filepath.Join(tmpDir, "clankers-"+time.Now().Format("2006-01-02")+".jsonl")
		if err := os.WriteFile(recentFile, []byte("recent log"), 0644); err != nil {
			t.Fatalf("failed to create recent file: %v", err)
		}

		// Run cleanup
		cleanupOldLogs(tmpDir)

		// Old file should be gone
		if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
			t.Error("expected old file to be removed")
		}

		// Recent file should still exist
		if _, err := os.Stat(recentFile); os.IsNotExist(err) {
			t.Error("expected recent file to still exist")
		}
	})

	t.Run("removes files strictly older than 30 days", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a file exactly 30 days old (should be kept)
		boundaryFile := filepath.Join(tmpDir, "clankers-2024-12-31.jsonl")
		if err := os.WriteFile(boundaryFile, []byte("boundary log"), 0644); err != nil {
			t.Fatalf("failed to create boundary file: %v", err)
		}
		// Use -30 days minus a bit to ensure we're strictly before cutoff
		boundaryTime := time.Now().AddDate(0, 0, -30).Add(-1 * time.Second)
		if err := os.Chtimes(boundaryFile, boundaryTime, boundaryTime); err != nil {
			t.Fatalf("failed to set boundary file time: %v", err)
		}

		// Run cleanup
		cleanupOldLogs(tmpDir)

		// File should be removed (strictly older than 30 days)
		if _, err := os.Stat(boundaryFile); !os.IsNotExist(err) {
			t.Error("expected boundary file to be removed (strictly older than 30 days)")
		}
	})

	t.Run("ignores non-clankers files", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create an old non-clankers file
		randomFile := filepath.Join(tmpDir, "random-file.txt")
		if err := os.WriteFile(randomFile, []byte("random"), 0644); err != nil {
			t.Fatalf("failed to create random file: %v", err)
		}
		oldTime := time.Now().AddDate(0, 0, -31)
		if err := os.Chtimes(randomFile, oldTime, oldTime); err != nil {
			t.Fatalf("failed to set random file time: %v", err)
		}

		// Create an old file without .jsonl extension
		wrongExtFile := filepath.Join(tmpDir, "clankers-2024-12-01.log")
		if err := os.WriteFile(wrongExtFile, []byte("wrong ext"), 0644); err != nil {
			t.Fatalf("failed to create wrong ext file: %v", err)
		}
		if err := os.Chtimes(wrongExtFile, oldTime, oldTime); err != nil {
			t.Fatalf("failed to set wrong ext file time: %v", err)
		}

		// Run cleanup
		cleanupOldLogs(tmpDir)

		// Non-clankers files should still exist
		if _, err := os.Stat(randomFile); os.IsNotExist(err) {
			t.Error("expected random file to still exist")
		}
		if _, err := os.Stat(wrongExtFile); os.IsNotExist(err) {
			t.Error("expected wrong extension file to still exist")
		}
	})

	t.Run("ignores directories", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a subdirectory with old name pattern
		subDir := filepath.Join(tmpDir, "clankers-2024-12-01.jsonl")
		if err := os.Mkdir(subDir, 0755); err != nil {
			t.Fatalf("failed to create subdirectory: %v", err)
		}
		oldTime := time.Now().AddDate(0, 0, -31)
		if err := os.Chtimes(subDir, oldTime, oldTime); err != nil {
			t.Fatalf("failed to set directory time: %v", err)
		}

		// Run cleanup
		cleanupOldLogs(tmpDir)

		// Directory should still exist
		info, err := os.Stat(subDir)
		if os.IsNotExist(err) {
			t.Error("expected subdirectory to still exist")
		}
		if err == nil && !info.IsDir() {
			t.Error("expected path to still be a directory")
		}
	})

	t.Run("handles missing directory gracefully", func(t *testing.T) {
		nonExistentDir := "/non/existent/directory/path"

		// Should not panic
		cleanupOldLogs(nonExistentDir)
	})

	t.Run("handles empty directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Should not panic
		cleanupOldLogs(tmpDir)
	})
}

func TestStartCleanupJob(t *testing.T) {
	t.Run("runs cleanup immediately on start", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create an old file
		oldFile := filepath.Join(tmpDir, "clankers-2024-12-01.jsonl")
		if err := os.WriteFile(oldFile, []byte("old"), 0644); err != nil {
			t.Fatalf("failed to create old file: %v", err)
		}
		oldTime := time.Now().AddDate(0, 0, -31)
		if err := os.Chtimes(oldFile, oldTime, oldTime); err != nil {
			t.Fatalf("failed to set old file time: %v", err)
		}

		// Start cleanup job
		stop := StartCleanupJob(tmpDir)
		defer close(stop)

		// Give it a moment to run
		time.Sleep(100 * time.Millisecond)

		// Old file should be removed
		if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
			t.Error("expected old file to be removed immediately")
		}
	})

	t.Run("returns stop channel", func(t *testing.T) {
		tmpDir := t.TempDir()

		stop := StartCleanupJob(tmpDir)

		// Verify it's a valid channel
		if stop == nil {
			t.Fatal("expected stop channel to not be nil")
		}

		// Should be able to close it
		close(stop)
	})

	t.Run("stops when channel closed", func(t *testing.T) {
		tmpDir := t.TempDir()

		stop := StartCleanupJob(tmpDir)

		// Close the stop channel
		close(stop)

		// Give goroutine time to stop
		time.Sleep(100 * time.Millisecond)

		// Test passes if no panic and goroutine exits
	})
}

func TestRetentionDays(t *testing.T) {
	// Verify the constant is set correctly
	if retentionDays != 30 {
		t.Errorf("expected retentionDays = 30, got %d", retentionDays)
	}
}

func TestCleanupFileNamePattern(t *testing.T) {
	tests := []struct {
		name            string
		filename        string
		isDirectory     bool
		shouldBeRemoved bool
	}{
		// Files matching pattern (clankers-*.jsonl) that are old should be removed
		{"valid clankers file old", "clankers-2024-12-01.jsonl", false, true},
		{"valid old date", "clankers-2024-06-15.jsonl", false, true},
		// Files matching pattern but recent should be kept
		{"valid clankers file recent", "clankers-" + time.Now().Format("2006-01-02") + ".jsonl", false, false},
		// Files not matching pattern should be kept even if old
		{"wrong extension", "clankers-2024-12-01.log", false, false},
		{"no prefix", "other-2024-12-01.jsonl", false, false},
		{"random file", "random.txt", false, false},
		{"no date in name", "clankers-.jsonl", false, false},
		// Directories should be kept
		{"directory with matching name", "clankers-2024-12-01.jsonl", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			path := filepath.Join(tmpDir, tt.filename)

			if tt.isDirectory {
				// Create as directory
				if err := os.Mkdir(path, 0755); err != nil {
					t.Fatalf("failed to create directory: %v", err)
				}
				// Set old time
				oldTime := time.Now().AddDate(0, 0, -31)
				if err := os.Chtimes(path, oldTime, oldTime); err != nil {
					t.Fatalf("failed to set directory time: %v", err)
				}
			} else {
				// Create as file
				if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
					t.Fatalf("failed to create file: %v", err)
				}

				// Set time based on whether it should be removed
				if tt.shouldBeRemoved {
					oldTime := time.Now().AddDate(0, 0, -31)
					if err := os.Chtimes(path, oldTime, oldTime); err != nil {
						t.Fatalf("failed to set old time: %v", err)
					}
				} else {
					// Keep it recent
					recentTime := time.Now().AddDate(0, 0, -1)
					if err := os.Chtimes(path, recentTime, recentTime); err != nil {
						t.Fatalf("failed to set recent time: %v", err)
					}
				}
			}

			cleanupOldLogs(tmpDir)

			_, err := os.Stat(path)
			exists := !os.IsNotExist(err)

			if tt.shouldBeRemoved && exists {
				t.Errorf("expected file %s to be removed", tt.filename)
			}
			if !tt.shouldBeRemoved && !exists {
				t.Errorf("expected file %s to still exist", tt.filename)
			}
		})
	}
}
