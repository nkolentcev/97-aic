package storage

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Storage представляет хранилище данных
type Storage struct {
	db *sql.DB
}

// Message представляет сообщение чата
type Message struct {
	ID        int64     `json:"id"`
	SessionID string    `json:"session_id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// RequestLog представляет лог запроса/ответа
type RequestLog struct {
	ID           int64     `json:"id"`
	SessionID    string    `json:"session_id"`
	RequestJSON  string    `json:"request_json"`
	ResponseJSON string    `json:"response_json"`
	StatusCode   int       `json:"status_code"`
	DurationMs   int64     `json:"duration_ms"`
	CreatedAt    time.Time `json:"created_at"`
}

// New создает новое хранилище
func New(dbPath string) (*Storage, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия БД: %w", err)
	}

	// Проверяем подключение
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ошибка подключения к БД: %w", err)
	}

	s := &Storage{db: db}

	// Создаем таблицы
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("ошибка миграции: %w", err)
	}

	return s, nil
}

// migrate создает необходимые таблицы
func (s *Storage) migrate() error {
	// Таблица сообщений
	messagesSQL := `
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT NOT NULL,
		role TEXT NOT NULL CHECK(role IN ('user', 'assistant')),
		content TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_messages_session ON messages(session_id);
	CREATE INDEX IF NOT EXISTS idx_messages_created ON messages(created_at);
	`

	// Таблица логов запросов/ответов
	requestLogsSQL := `
	CREATE TABLE IF NOT EXISTS request_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT,
		request_json TEXT NOT NULL,
		response_json TEXT,
		status_code INTEGER,
		duration_ms INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_request_logs_session ON request_logs(session_id);
	CREATE INDEX IF NOT EXISTS idx_request_logs_created ON request_logs(created_at);
	`

	if _, err := s.db.Exec(messagesSQL); err != nil {
		return fmt.Errorf("ошибка создания таблицы messages: %w", err)
	}

	if _, err := s.db.Exec(requestLogsSQL); err != nil {
		return fmt.Errorf("ошибка создания таблицы request_logs: %w", err)
	}

	return nil
}

// SaveMessage сохраняет сообщение
func (s *Storage) SaveMessage(sessionID, role, content string) (*Message, error) {
	result, err := s.db.Exec(
		"INSERT INTO messages (session_id, role, content) VALUES (?, ?, ?)",
		sessionID, role, content,
	)
	if err != nil {
		return nil, fmt.Errorf("ошибка сохранения сообщения: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("ошибка получения ID: %w", err)
	}

	return &Message{
		ID:        id,
		SessionID: sessionID,
		Role:      role,
		Content:   content,
		CreatedAt: time.Now(),
	}, nil
}

// GetMessages возвращает сообщения сессии
func (s *Storage) GetMessages(sessionID string, limit int) ([]Message, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := s.db.Query(
		"SELECT id, session_id, role, content, created_at FROM messages WHERE session_id = ? ORDER BY created_at ASC LIMIT ?",
		sessionID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения сообщений: %w", err)
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		if err := rows.Scan(&msg.ID, &msg.SessionID, &msg.Role, &msg.Content, &msg.CreatedAt); err != nil {
			return nil, fmt.Errorf("ошибка сканирования сообщения: %w", err)
		}
		messages = append(messages, msg)
	}

	return messages, rows.Err()
}

// SaveRequestLog сохраняет лог запроса
func (s *Storage) SaveRequestLog(sessionID, requestJSON, responseJSON string, statusCode int, durationMs int64) (*RequestLog, error) {
	result, err := s.db.Exec(
		"INSERT INTO request_logs (session_id, request_json, response_json, status_code, duration_ms) VALUES (?, ?, ?, ?, ?)",
		sessionID, requestJSON, responseJSON, statusCode, durationMs,
	)
	if err != nil {
		return nil, fmt.Errorf("ошибка сохранения лога: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("ошибка получения ID: %w", err)
	}

	return &RequestLog{
		ID:           id,
		SessionID:    sessionID,
		RequestJSON:  requestJSON,
		ResponseJSON: responseJSON,
		StatusCode:   statusCode,
		DurationMs:   durationMs,
		CreatedAt:    time.Now(),
	}, nil
}

// GetRequestLogs возвращает логи запросов
func (s *Storage) GetRequestLogs(sessionID string, limit int) ([]RequestLog, error) {
	if limit <= 0 {
		limit = 100
	}

	query := "SELECT id, session_id, request_json, response_json, status_code, duration_ms, created_at FROM request_logs"
	args := []interface{}{}

	if sessionID != "" {
		query += " WHERE session_id = ?"
		args = append(args, sessionID)
	}
	query += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения логов: %w", err)
	}
	defer rows.Close()

	var logs []RequestLog
	for rows.Next() {
		var log RequestLog
		var sessionIDNull sql.NullString
		var responseJSONNull sql.NullString
		var statusCodeNull sql.NullInt64
		var durationMsNull sql.NullInt64

		if err := rows.Scan(&log.ID, &sessionIDNull, &log.RequestJSON, &responseJSONNull, &statusCodeNull, &durationMsNull, &log.CreatedAt); err != nil {
			return nil, fmt.Errorf("ошибка сканирования лога: %w", err)
		}

		if sessionIDNull.Valid {
			log.SessionID = sessionIDNull.String
		}
		if responseJSONNull.Valid {
			log.ResponseJSON = responseJSONNull.String
		}
		if statusCodeNull.Valid {
			log.StatusCode = int(statusCodeNull.Int64)
		}
		if durationMsNull.Valid {
			log.DurationMs = durationMsNull.Int64
		}

		logs = append(logs, log)
	}

	return logs, rows.Err()
}

// Close закрывает соединение с БД
func (s *Storage) Close() error {
	return s.db.Close()
}
