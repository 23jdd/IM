package main

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite"
)

// LocalStore 用纯 Go SQLite 在本地持久化收发的消息，支持离线/历史加载。
type LocalStore struct {
	mu sync.Mutex
	db *sql.DB
}

func NewLocalStore() *LocalStore { return &LocalStore{} }

type LocalMessage struct {
	MsgId   string `json:"msg_id"`
	FromUid string `json:"from_uid"`
	Content string `json:"content"`
	Self    bool   `json:"self"`
	Status  string `json:"status"`
	Ts      int64  `json:"ts"`
}

// Init 打开（或创建）当前账号的本地消息库并建表。按 selfUid 分库，避免多账号混淆。
func (s *LocalStore) Init(selfUid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.db != nil {
		_ = s.db.Close()
		s.db = nil
	}

	dir, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	dir = filepath.Join(dir, "im-client")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	name := "messages.db"
	if selfUid != "" {
		name = "messages_" + selfUid + ".db"
	}

	db, err := sql.Open("sqlite", filepath.Join(dir, name))
	if err != nil {
		return err
	}
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS messages (
			id       INTEGER PRIMARY KEY AUTOINCREMENT,
			peer     TEXT NOT NULL,
			msg_id   TEXT,
			from_uid TEXT,
			content  TEXT,
			self     INTEGER,
			status   TEXT,
			ts       INTEGER
		);
		CREATE INDEX IF NOT EXISTS idx_peer_ts ON messages(peer, ts);
	`); err != nil {
		_ = db.Close()
		return err
	}
	s.db = db
	return nil
}

// SaveMessage 持久化一条消息。
func (s *LocalStore) SaveMessage(peer, msgId, fromUid, content string, self bool, status string, ts int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return errors.New("local store not initialized")
	}
	selfInt := 0
	if self {
		selfInt = 1
	}
	_, err := s.db.Exec(
		`INSERT INTO messages (peer, msg_id, from_uid, content, self, status, ts) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		peer, msgId, fromUid, content, selfInt, status, ts,
	)
	return err
}

// LoadMessages 按会话加载最近 limit 条消息（时间升序）。
func (s *LocalStore) LoadMessages(peer string, limit int) ([]LocalMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return nil, errors.New("local store not initialized")
	}
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.db.Query(
		`SELECT msg_id, from_uid, content, self, status, ts FROM messages WHERE peer = ? ORDER BY ts ASC LIMIT ?`,
		peer, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]LocalMessage, 0)
	for rows.Next() {
		var m LocalMessage
		var selfInt int
		if err := rows.Scan(&m.MsgId, &m.FromUid, &m.Content, &selfInt, &m.Status, &m.Ts); err != nil {
			return nil, err
		}
		m.Self = selfInt == 1
		out = append(out, m)
	}
	return out, rows.Err()
}
