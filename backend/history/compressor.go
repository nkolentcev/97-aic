package history

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nnk/97-aic/backend/logger"
	"github.com/nnk/97-aic/backend/provider"
	"github.com/nnk/97-aic/backend/storage"
)

// Config настройки компрессии истории.
type Config struct {
	Enabled          bool
	EveryMessages    int
	KeepLastMessages int
	MaxTokens        int
	Temperature      float64
}

const (
	defaultEveryMessages    = 10
	defaultKeepLastMessages = 4
	defaultMaxTokens        = 256
	defaultTemperature      = 0.2
)

func (c Config) withDefaults() Config {
	if c.EveryMessages <= 0 {
		c.EveryMessages = defaultEveryMessages
	}
	if c.KeepLastMessages < 0 {
		c.KeepLastMessages = defaultKeepLastMessages
	}
	if c.MaxTokens <= 0 {
		c.MaxTokens = defaultMaxTokens
	}
	// Temperature: если не задана, используем дефолт
	if c.Temperature < 0 {
		c.Temperature = defaultTemperature
	}
	if c.Temperature == 0 {
		c.Temperature = defaultTemperature
	}
	return c
}

// CompressSessionIfNeeded сворачивает историю в summary (батчами) и удаляет оригиналы.
// Возвращает true, если была выполнена компрессия хотя бы один раз.
func CompressSessionIfNeeded(ctx context.Context, p provider.Provider, store *storage.Storage, sessionID string, cfg Config) (bool, error) {
	if store == nil || p == nil || sessionID == "" {
		return false, nil
	}
	cfg = cfg.withDefaults()
	if !cfg.Enabled {
		return false, nil
	}

	did := false

	for {
		select {
		case <-ctx.Done():
			return did, ctx.Err()
		default:
		}

		cnt, err := store.CountNonSummaryMessages(sessionID)
		if err != nil {
			return did, err
		}

		// Делаем summary только когда есть "голова" минимум на EveryMessages,
		// и при этом сохраняем KeepLastMessages последних сообщений как «хвост».
		if cnt <= cfg.KeepLastMessages+cfg.EveryMessages {
			return did, nil
		}

		batch, err := store.GetOldestNonSummaryMessages(sessionID, cfg.EveryMessages, cfg.KeepLastMessages)
		if err != nil {
			return did, err
		}
		if len(batch) < cfg.EveryMessages {
			return did, nil
		}

		prevSummary, err := store.GetLatestSummary(sessionID)
		if err != nil {
			return did, err
		}

		prompt := buildSummarizePrompt(prevSummary, batch)
		summary, err := summarize(ctx, p, prompt, cfg.MaxTokens, cfg.Temperature)
		if err != nil {
			return did, err
		}

		if _, err := store.UpsertSummary(sessionID, summary); err != nil {
			return did, err
		}

		ids := make([]int64, 0, len(batch))
		for _, m := range batch {
			ids = append(ids, m.ID)
		}
		if err := store.DeleteMessagesByIDs(sessionID, ids); err != nil {
			return did, err
		}

		did = true
		logger.Info("история сжата", "session_id", sessionID, "compressed_messages", len(batch), "summary_len", len(summary))
	}
}

func buildSummarizePrompt(prevSummary *storage.Message, batch []storage.Message) string {
	var b strings.Builder

	b.WriteString("Задача: обновить краткое резюме диалога.\n")
	b.WriteString("Требования:\n")
	b.WriteString("- Сохрани факты, требования, ограничения, принятые решения, текущий статус и открытые вопросы.\n")
	b.WriteString("- Сохрани важные значения/идентификаторы/пути/команды, если они упоминались.\n")
	b.WriteString("- Не придумывай детали.\n")
	b.WriteString("- Результат: компактный текст на русском, без markdown.\n")
	b.WriteString("- Объем: по возможности кратко, но не теряй критичную информацию.\n\n")

	if prevSummary != nil && strings.TrimSpace(prevSummary.Content) != "" {
		b.WriteString("Текущее резюме (обнови его):\n")
		b.WriteString(prevSummary.Content)
		b.WriteString("\n\n")
	}

	b.WriteString("Новые сообщения для включения в резюме:\n")
	for _, m := range batch {
		switch m.Role {
		case storage.RoleUser:
			b.WriteString("USER: ")
		case storage.RoleAssistant:
			b.WriteString("ASSISTANT: ")
		default:
			b.WriteString(strings.ToUpper(m.Role) + ": ")
		}
		b.WriteString(m.Content)
		b.WriteString("\n")
	}

	return b.String()
}

func summarize(ctx context.Context, p provider.Provider, prompt string, maxTokens int, temperature float64) (string, error) {
	systemPrompt := "Ты — модуль компрессии истории диалога. Отвечай только резюме."
	opts := &provider.ChatOptions{
		SystemPrompt: systemPrompt,
		MaxTokens:    maxTokens,
		Temperature:  temperature,
	}

	var out strings.Builder
	err := p.Chat(ctx, prompt, opts, func(chunk string) error {
		out.WriteString(chunk)
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("ошибка генерации summary: %w", err)
	}

	s := strings.TrimSpace(out.String())
	// Минимальная нормализация — убираем пустой результат.
	if s == "" {
		// Фоллбек: не должно случаться, но лучше не удалять сообщения без summary.
		return "", fmt.Errorf("получено пустое summary")
	}
	// Ограничиваем без фанатизма (на случай ухода модели в простыню).
	if len([]rune(s)) > 4000 {
		rs := []rune(s)
		s = strings.TrimSpace(string(rs[:4000]))
	}

	// Пауза, чтобы не захлебнуться при цепочке батчей на слабых провайдерах.
	time.Sleep(50 * time.Millisecond)

	return s, nil
}
