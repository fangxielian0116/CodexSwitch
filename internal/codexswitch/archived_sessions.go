package codexswitch

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

const archivedSessionsDirName = "archived_sessions"

func (s *Service) cleanupExpiredArchivedSessions(settings AppSettings) {
	if strings.TrimSpace(settings.CodexHomePath) == "" || settings.ArchivedSessionRetentionDays <= 0 {
		return
	}

	archiveDir := filepath.Join(settings.CodexHomePath, archivedSessionsDirName)
	entries, err := os.ReadDir(archiveDir)
	if err != nil {
		if !isNotFound(err) {
			s.logger.Warn("read archived sessions failed", "path", archiveDir, "error", err)
		}
		return
	}

	cutoff := s.now().Add(-time.Duration(settings.ArchivedSessionRetentionDays) * 24 * time.Hour)
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			s.logger.Warn("stat archived session failed", "name", entry.Name(), "error", err)
			continue
		}
		if !info.ModTime().Before(cutoff) {
			continue
		}

		path := filepath.Join(archiveDir, entry.Name())
		if err := os.RemoveAll(path); err != nil {
			s.logger.Warn("delete expired archived session failed", "path", path, "error", err)
		}
	}
}
