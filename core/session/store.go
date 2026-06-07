package session

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	db  *sql.DB
	mu  sync.RWMutex
	dir string
}

func NewStore(dataDir string) (*Store, error) {
	storeDir := filepath.Join(dataDir, "sessions")
	os.MkdirAll(storeDir, 0755)
	dbPath := filepath.Join(storeDir, "sessions.db")

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	s := &Store{db: db, dir: storeDir}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			data TEXT NOT NULL,
			title TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_sessions_updated ON sessions(updated_at DESC);
	`)
	return err
}

func (s *Store) SaveRaw(id, data string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Extract title quickly
	var title string
	var sess struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal([]byte(data), &sess); err == nil {
		title = sess.Title
	}

	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO sessions (id, data, title, updated_at) VALUES (?, ?, ?, ?)`,
		id, data, title, time.Now().Format(time.RFC3339),
	)
	return err
}

func (s *Store) Save(session *Session) error {
	data, _ := json.Marshal(session)
	return s.SaveRaw(session.ID, string(data))
}

func (s *Store) Load(id string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var data string
	err := s.db.QueryRow("SELECT data FROM sessions WHERE id = ?", id).Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found: %s", id)
		}
		return nil, err
	}

	var session Session
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, fmt.Errorf("failed to parse session: %w", err)
	}
	session.store = s
	return &session, nil
}

func (s *Store) List() ([]*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query("SELECT data FROM sessions ORDER BY updated_at DESC LIMIT 20")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			continue
		}
		var session Session
		if err := json.Unmarshal([]byte(data), &session); err != nil {
			continue
		}
		session.store = s
		sessions = append(sessions, &session)
	}
	return sessions, nil
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec("DELETE FROM sessions WHERE id = ?", id)
	return err
}

func (s *Store) Close() error {
	return s.db.Close()
}
