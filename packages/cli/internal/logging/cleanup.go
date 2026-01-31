package logging

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

const retentionDays = 30

func cleanupOldLogs(logDir string) {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)

	entries, err := os.ReadDir(logDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasPrefix(name, "clankers-") || !strings.HasSuffix(name, ".jsonl") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			os.Remove(filepath.Join(logDir, name))
		}
	}
}

// StartCleanupJob runs cleanup immediately, then every 24 hours.
// Returns a channel to stop the background goroutine.
func StartCleanupJob(logDir string) chan<- struct{} {
	// Run immediately
	cleanupOldLogs(logDir)

	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				cleanupOldLogs(logDir)
			case <-stop:
				return
			}
		}
	}()

	return stop
}
