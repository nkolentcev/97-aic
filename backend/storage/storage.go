package storage

import (
	"database/sql"
	"fmt"
	"strings"
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

const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleSummary   = "summary"
)

// RequestLog представляет лог запроса/ответа
type RequestLog struct {
	ID           int64     `json:"id"`
	SessionID    string    `json:"session_id"`
	RequestJSON  string    `json:"request_json"`
	ResponseJSON string    `json:"response_json"`
	StatusCode   int       `json:"status_code"`
	DurationMs   int64     `json:"duration_ms"`
	TokensInput  *int      `json:"tokens_input,omitempty"`
	TokensOutput *int      `json:"tokens_output,omitempty"`
	TokensTotal  *int      `json:"tokens_total,omitempty"`
	Cost         *float64  `json:"cost,omitempty"`
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
		role TEXT NOT NULL CHECK(role IN ('user', 'assistant', 'summary')),
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

	// Миграция: расширяем CHECK(role) для messages (старые БД могли иметь только user/assistant)
	if err := s.migrateMessagesRoleConstraint(); err != nil {
		return fmt.Errorf("ошибка миграции роли messages: %w", err)
	}

	if _, err := s.db.Exec(requestLogsSQL); err != nil {
		return fmt.Errorf("ошибка создания таблицы request_logs: %w", err)
	}

	// Миграция: добавляем новые поля для токенов и стоимости
	if err := s.migrateTokensFields(); err != nil {
		return fmt.Errorf("ошибка миграции полей токенов: %w", err)
	}

	return nil
}

// migrateTokensFields добавляет поля для токенов и стоимости в существующую таблицу
func (s *Storage) migrateTokensFields() error {
	// Проверяем существование колонок и добавляем их, если их нет
	// SQLite не поддерживает IF NOT EXISTS для ALTER TABLE, поэтому используем проверку через PRAGMA
	columns := []struct {
		name string
		sql  string
	}{
		{"tokens_input", "ALTER TABLE request_logs ADD COLUMN tokens_input INTEGER"},
		{"tokens_output", "ALTER TABLE request_logs ADD COLUMN tokens_output INTEGER"},
		{"tokens_total", "ALTER TABLE request_logs ADD COLUMN tokens_total INTEGER"},
		{"cost", "ALTER TABLE request_logs ADD COLUMN cost REAL"},
	}

	for _, col := range columns {
		// Проверяем существование колонки
		rows, err := s.db.Query("PRAGMA table_info(request_logs)")
		if err != nil {
			return fmt.Errorf("ошибка проверки структуры таблицы: %w", err)
		}

		columnExists := false
		for rows.Next() {
			var cid int
			var name string
			var dataType string
			var notNull int
			var defaultValue interface{}
			var pk int

			if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
				rows.Close()
				return fmt.Errorf("ошибка сканирования структуры таблицы: %w", err)
			}

			if name == col.name {
				columnExists = true
				break
			}
		}
		rows.Close()

		// Добавляем колонку, если её нет
		if !columnExists {
			if _, err := s.db.Exec(col.sql); err != nil {
				// Игнорируем ошибку, если колонка уже существует (может быть race condition)
				if !strings.Contains(err.Error(), "duplicate column") {
					return fmt.Errorf("ошибка добавления колонки %s: %w", col.name, err)
				}
			}
		}
	}

	return nil
}

// migrateMessagesRoleConstraint пересоздает таблицу messages, если CHECK(role) не допускает summary.
func (s *Storage) migrateMessagesRoleConstraint() error {
	// Быстрая проверка: пробуем вставить роль summary в транзакции и откатываем.
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("ошибка начала транзакции: %w", err)
	}
	if _, err := tx.Exec("INSERT INTO messages (session_id, role, content) VALUES (?, ?, ?)", "_role_probe_", RoleSummary, "probe"); err == nil {
		_ = tx.Rollback()
		return nil
	}
	_ = tx.Rollback()

	// Пересоздаем таблицу, сохраняя данные.
	// SQLite не позволяет ALTER CHECK-constraint, поэтому делаем copy+swap.
	_, err = s.db.Exec(`
BEGIN;
CREATE TABLE IF NOT EXISTS messages_new (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	session_id TEXT NOT NULL,
	role TEXT NOT NULL CHECK(role IN ('user', 'assistant', 'summary')),
	content TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO messages_new (id, session_id, role, content, created_at)
SELECT id, session_id, role, content, created_at FROM messages;
DROP TABLE messages;
ALTER TABLE messages_new RENAME TO messages;
CREATE INDEX IF NOT EXISTS idx_messages_session ON messages(session_id);
CREATE INDEX IF NOT EXISTS idx_messages_created ON messages(created_at);
COMMIT;
`)
	if err != nil {
		// На всякий случай откатываем, если BEGIN уже прошел
		_, _ = s.db.Exec("ROLLBACK;")
		return fmt.Errorf("ошибка пересоздания таблицы messages: %w", err)
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
		"SELECT id, session_id, role, content, created_at FROM messages WHERE session_id = ? ORDER BY id ASC LIMIT ?",
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

// GetLatestSummary возвращает последнее summary для сессии (если есть).
func (s *Storage) GetLatestSummary(sessionID string) (*Message, error) {
	row := s.db.QueryRow(
		"SELECT id, session_id, role, content, created_at FROM messages WHERE session_id = ? AND role = ? ORDER BY id DESC LIMIT 1",
		sessionID, RoleSummary,
	)
	var msg Message
	if err := row.Scan(&msg.ID, &msg.SessionID, &msg.Role, &msg.Content, &msg.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("ошибка получения summary: %w", err)
	}
	return &msg, nil
}

// CountNonSummaryMessages возвращает количество user/assistant сообщений в сессии.
func (s *Storage) CountNonSummaryMessages(sessionID string) (int, error) {
	row := s.db.QueryRow(
		"SELECT COUNT(1) FROM messages WHERE session_id = ? AND role IN (?, ?)",
		sessionID, RoleUser, RoleAssistant,
	)
	var cnt int
	if err := row.Scan(&cnt); err != nil {
		return 0, fmt.Errorf("ошибка подсчета сообщений: %w", err)
	}
	return cnt, nil
}

// GetOldestNonSummaryMessages возвращает самые ранние user/assistant сообщения, исключая keepLast последних.
func (s *Storage) GetOldestNonSummaryMessages(sessionID string, batchSize int, keepLast int) ([]Message, error) {
	if batchSize <= 0 {
		return nil, fmt.Errorf("batchSize должен быть > 0")
	}
	if keepLast < 0 {
		keepLast = 0
	}

	// Выбираем самые ранние сообщения из "головы", исключив keepLast последних по id.
	rows, err := s.db.Query(
		`
SELECT id, session_id, role, content, created_at
FROM messages
WHERE session_id = ?
  AND role IN (?, ?)
  AND id NOT IN (
    SELECT id FROM messages
    WHERE session_id = ?
      AND role IN (?, ?)
    ORDER BY id DESC
    LIMIT ?
  )
ORDER BY id ASC
LIMIT ?
`,
		sessionID, RoleUser, RoleAssistant,
		sessionID, RoleUser, RoleAssistant, keepLast,
		batchSize,
	)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения сообщений для компрессии: %w", err)
	}
	defer rows.Close()

	var msgs []Message
	for rows.Next() {
		var msg Message
		if err := rows.Scan(&msg.ID, &msg.SessionID, &msg.Role, &msg.Content, &msg.CreatedAt); err != nil {
			return nil, fmt.Errorf("ошибка сканирования сообщения: %w", err)
		}
		msgs = append(msgs, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка чтения сообщений: %w", err)
	}
	return msgs, nil
}

// UpsertSummary создает или обновляет summary сообщение.
func (s *Storage) UpsertSummary(sessionID string, content string) (*Message, error) {
	existing, err := s.GetLatestSummary(sessionID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return s.SaveMessage(sessionID, RoleSummary, content)
	}
	if _, err := s.db.Exec("UPDATE messages SET content = ?, created_at = CURRENT_TIMESTAMP WHERE id = ?", content, existing.ID); err != nil {
		return nil, fmt.Errorf("ошибка обновления summary: %w", err)
	}
	existing.Content = content
	existing.CreatedAt = time.Now()
	return existing, nil
}

// DeleteMessagesByIDs удаляет сообщения по списку id (в рамках сессии).
func (s *Storage) DeleteMessagesByIDs(sessionID string, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(ids)), ",")
	args := make([]interface{}, 0, len(ids)+1)
	args = append(args, sessionID)
	for _, id := range ids {
		args = append(args, id)
	}
	query := fmt.Sprintf("DELETE FROM messages WHERE session_id = ? AND id IN (%s)", placeholders)
	if _, err := s.db.Exec(query, args...); err != nil {
		return fmt.Errorf("ошибка удаления сообщений: %w", err)
	}
	return nil
}

// SaveRequestLog сохраняет лог запроса
func (s *Storage) SaveRequestLog(sessionID, requestJSON, responseJSON string, statusCode int, durationMs int64, tokensInput, tokensOutput, tokensTotal *int, cost *float64) (*RequestLog, error) {
	result, err := s.db.Exec(
		"INSERT INTO request_logs (session_id, request_json, response_json, status_code, duration_ms, tokens_input, tokens_output, tokens_total, cost) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		sessionID, requestJSON, responseJSON, statusCode, durationMs, tokensInput, tokensOutput, tokensTotal, cost,
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
		TokensInput:  tokensInput,
		TokensOutput: tokensOutput,
		TokensTotal:  tokensTotal,
		Cost:         cost,
		CreatedAt:    time.Now(),
	}, nil
}

// GetRequestLogs возвращает логи запросов
func (s *Storage) GetRequestLogs(sessionID string, limit int) ([]RequestLog, error) {
	if limit <= 0 {
		limit = 100
	}

	query := "SELECT id, session_id, request_json, response_json, status_code, duration_ms, tokens_input, tokens_output, tokens_total, cost, created_at FROM request_logs"
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
		var tokensInputNull sql.NullInt64
		var tokensOutputNull sql.NullInt64
		var tokensTotalNull sql.NullInt64
		var costNull sql.NullFloat64

		if err := rows.Scan(&log.ID, &sessionIDNull, &log.RequestJSON, &responseJSONNull, &statusCodeNull, &durationMsNull, &tokensInputNull, &tokensOutputNull, &tokensTotalNull, &costNull, &log.CreatedAt); err != nil {
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
		if tokensInputNull.Valid {
			val := int(tokensInputNull.Int64)
			log.TokensInput = &val
		}
		if tokensOutputNull.Valid {
			val := int(tokensOutputNull.Int64)
			log.TokensOutput = &val
		}
		if tokensTotalNull.Valid {
			val := int(tokensTotalNull.Int64)
			log.TokensTotal = &val
		}
		if costNull.Valid {
			val := costNull.Float64
			log.Cost = &val
		}

		logs = append(logs, log)
	}

	return logs, rows.Err()
}

// Close закрывает соединение с БД
func (s *Storage) Close() error {
	return s.db.Close()
}
