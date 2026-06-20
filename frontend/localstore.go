package main

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite"
)

// LocalStore 用纯 Go SQLite 在本地持久化登录态与收发消息：
//   - session.db：当前登录态（token/uid/name/profile），替代 localStorage。
//   - messages_<uid>.db：按账号分库的消息历史。
type LocalStore struct {
	mu        sync.Mutex
	db        *sql.DB // 当前账号的消息库
	sessionDB *sql.DB // 登录态库（固定）
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

type Session struct {
	Token   string `json:"token"`
	Uid     string `json:"uid"`
	Name    string `json:"name"`
	Profile string `json:"profile"` // 序列化的 JSON 字符串
}

// dataDir 返回本地数据目录。优先使用环境变量 IM_DATA_DIR（便于同机多实例隔离测试），
// 否则用系统用户配置目录下的 im-client。
func dataDir() (string, error) {
	dir := os.Getenv("IM_DATA_DIR")
	if dir == "" {
		base, err := os.UserConfigDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(base, "im-client")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// ---- 登录态（session.db）----

func (s *LocalStore) openSession() error {
	if s.sessionDB != nil {
		return nil
	}
	dir, err := dataDir()
	if err != nil {
		return err
	}
	db, err := sql.Open("sqlite", filepath.Join(dir, "session.db"))
	if err != nil {
		return err
	}
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS session (
			id      INTEGER PRIMARY KEY CHECK (id = 1),
			token   TEXT,
			uid     TEXT,
			name    TEXT,
			profile TEXT
		);`); err != nil {
		_ = db.Close()
		return err
	}
	s.sessionDB = db
	return nil
}

// SaveSession 持久化登录态。
func (s *LocalStore) SaveSession(token, uid, name, profile string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.openSession(); err != nil {
		return err
	}
	_, err := s.sessionDB.Exec(
		`INSERT INTO session (id, token, uid, name, profile) VALUES (1, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET token=excluded.token, uid=excluded.uid, name=excluded.name, profile=excluded.profile`,
		token, uid, name, profile,
	)
	return err
}

// LoadSession 读取当前登录态（无则返回空 Session）。
func (s *LocalStore) LoadSession() (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.openSession(); err != nil {
		return nil, err
	}
	var ss Session
	err := s.sessionDB.QueryRow(`SELECT token, uid, name, profile FROM session WHERE id = 1`).
		Scan(&ss.Token, &ss.Uid, &ss.Name, &ss.Profile)
	if errors.Is(err, sql.ErrNoRows) {
		return &Session{}, nil
	}
	if err != nil {
		return nil, err
	}
	return &ss, nil
}

// ClearSession 登出时清除登录态。
func (s *LocalStore) ClearSession() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.openSession(); err != nil {
		return err
	}
	_, err := s.sessionDB.Exec(`DELETE FROM session WHERE id = 1`)
	return err
}

// ---- 消息历史（messages_<uid>.db，按账号分库）----

// Init 打开（或创建）当前账号的本地消息库并建表。
func (s *LocalStore) Init(selfUid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.db != nil {
		_ = s.db.Close()
		s.db = nil
	}

	dir, err := dataDir()
	if err != nil {
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
