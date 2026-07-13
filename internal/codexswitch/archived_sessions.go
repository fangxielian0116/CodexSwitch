package codexswitch

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const archivedSessionsDirName = "archived_sessions"
const codexStateDBPattern = "state_*.sqlite"

func (s *Service) cleanupExpiredArchivedSessions(settings AppSettings) {
	if strings.TrimSpace(settings.CodexHomePath) == "" || settings.ArchivedSessionRetentionDays <= 0 {
		return
	}

	archiveDir := filepath.Join(settings.CodexHomePath, archivedSessionsDirName)
	cutoff := s.now().Add(-time.Duration(settings.ArchivedSessionRetentionDays) * 24 * time.Hour)
	expiredPaths := make(map[string]struct{})

	entries, err := os.ReadDir(archiveDir)
	if err != nil {
		if !isNotFound(err) {
			s.logger.Warn("read archived sessions failed", "path", archiveDir, "error", err)
			return
		}
	} else {
		for _, entry := range entries {
			info, err := entry.Info()
			if err != nil {
				s.logger.Warn("stat archived session failed", "name", entry.Name(), "error", err)
				continue
			}
			if info.ModTime().Before(cutoff) {
				expiredPaths[filepath.Clean(filepath.Join(archiveDir, entry.Name()))] = struct{}{}
			}
		}
	}

	if err := cleanupExpiredArchivedSessionIndexes(settings.CodexHomePath, archiveDir, cutoff, expiredPaths); err != nil {
		s.logger.Warn("delete expired archived session indexes failed", "error", err)
		return
	}

	for path := range expiredPaths {
		if err := os.RemoveAll(path); err != nil {
			s.logger.Warn("delete expired archived session failed", "path", path, "error", err)
		}
	}
}

func cleanupExpiredArchivedSessionIndexes(
	codexHome string,
	archiveDir string,
	cutoff time.Time,
	expiredPaths map[string]struct{},
) error {
	stateDBPath, err := latestCodexStateDBPath(codexHome)
	if err != nil || stateDBPath == "" {
		return err
	}

	db, err := sql.Open("sqlite", stateDBPath)
	if err != nil {
		return fmt.Errorf("open Codex state database: %w", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	if _, err := db.Exec("PRAGMA busy_timeout = 5000"); err != nil {
		return fmt.Errorf("configure Codex state database: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("enable Codex state foreign keys: %w", err)
	}

	var threadsTableExists int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'threads'`).Scan(&threadsTableExists); err != nil {
		return fmt.Errorf("inspect Codex state database: %w", err)
	}
	if threadsTableExists == 0 {
		return nil
	}

	rows, err := db.Query(`SELECT id, rollout_path, archived_at FROM threads WHERE archived = 1`)
	if err != nil {
		return fmt.Errorf("read archived session indexes: %w", err)
	}

	var expiredIDs []string
	for rows.Next() {
		var id string
		var rolloutPath string
		var archivedAt sql.NullInt64
		if err := rows.Scan(&id, &rolloutPath, &archivedAt); err != nil {
			rows.Close()
			return fmt.Errorf("scan archived session index: %w", err)
		}

		rolloutPath = filepath.Clean(rolloutPath)
		if !isPathWithinDirectory(rolloutPath, archiveDir) {
			continue
		}
		if containsPath(expiredPaths, rolloutPath) {
			expiredIDs = append(expiredIDs, id)
			continue
		}

		if _, err := os.Stat(rolloutPath); isNotFound(err) && archivedAt.Valid && time.Unix(archivedAt.Int64, 0).Before(cutoff) {
			expiredIDs = append(expiredIDs, id)
		}
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close archived session indexes: %w", err)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read archived session indexes: %w", err)
	}
	if len(expiredIDs) == 0 {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin archived session index cleanup: %w", err)
	}
	defer tx.Rollback()

	for _, id := range expiredIDs {
		if _, err := tx.Exec(`DELETE FROM threads WHERE id = ? AND archived = 1`, id); err != nil {
			return fmt.Errorf("delete archived session index %s: %w", id, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit archived session index cleanup: %w", err)
	}
	return nil
}

func latestCodexStateDBPath(codexHome string) (string, error) {
	matches, err := filepath.Glob(filepath.Join(codexHome, codexStateDBPattern))
	if err != nil {
		return "", fmt.Errorf("find Codex state database: %w", err)
	}

	var latestPath string
	var latestModTime time.Time
	for _, path := range matches {
		info, err := os.Stat(path)
		if err != nil {
			return "", fmt.Errorf("stat Codex state database %s: %w", path, err)
		}
		if latestPath == "" || info.ModTime().After(latestModTime) {
			latestPath = path
			latestModTime = info.ModTime()
		}
	}
	return latestPath, nil
}

func containsPath(paths map[string]struct{}, target string) bool {
	for path := range paths {
		if strings.EqualFold(filepath.Clean(path), filepath.Clean(target)) {
			return true
		}
	}
	return false
}

func isPathWithinDirectory(path string, directory string) bool {
	relative, err := filepath.Rel(filepath.Clean(directory), filepath.Clean(path))
	if err != nil {
		return false
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
