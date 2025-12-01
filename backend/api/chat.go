package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/nnk/97-aic/backend/gigachat"
)

// ChatHandler обрабатывает запросы к /api/chat
type ChatHandler struct {
	GigaChatClient *gigachat.Client
}

// ChatRequest представляет входящий запрос
type ChatRequest struct {
	Message string `json:"message"`
}

// NewChatHandler создает новый обработчик чата
func NewChatHandler(client *gigachat.Client) *ChatHandler {
	return &ChatHandler{
		GigaChatClient: client,
	}
}

// ServeHTTP обрабатывает HTTP запросы
func (h *ChatHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Обработка CORS preflight
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		log.Printf("Неверный метод: %s", r.Method)
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	log.Printf("Получен запрос: Content-Type=%s, Content-Length=%s", r.Header.Get("Content-Type"), r.Header.Get("Content-Length"))

	// Читаем тело запроса для логирования
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Ошибка чтения тела запроса: %v", err)
		http.Error(w, "Ошибка чтения запроса", http.StatusBadRequest)
		return
	}
	log.Printf("Тело запроса: %s", string(bodyBytes))

	// Создаем новый reader для декодера
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	var req ChatRequest
	decoder := json.NewDecoder(bytes.NewBuffer(bodyBytes))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&req); err != nil {
		log.Printf("Ошибка парсинга JSON: %v", err)
		log.Printf("Тело запроса (raw): %s", string(bodyBytes))
		log.Printf("Тело запроса (hex): %x", bodyBytes)
		
		// Пробуем распарсить как есть, чтобы увидеть структуру
		var rawData map[string]interface{}
		if json.Unmarshal(bodyBytes, &rawData) == nil {
			log.Printf("Распарсенные данные: %+v", rawData)
		}
		
		http.Error(w, fmt.Sprintf("Ошибка парсинга запроса: %v", err), http.StatusBadRequest)
		return
	}

	log.Printf("Распарсенное сообщение: %q", req.Message)

	if req.Message == "" {
		log.Printf("Пустое сообщение")
		http.Error(w, "Поле message обязательно", http.StatusBadRequest)
		return
	}

	// Настройка для streaming ответа
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming не поддерживается", http.StatusInternalServerError)
		return
	}

	// Отправка сообщений через streaming
	err = h.GigaChatClient.Chat(req.Message, func(chunk string) error {
		data := map[string]string{"content": chunk}
		jsonData, marshalErr := json.Marshal(data)
		if marshalErr != nil {
			return marshalErr
		}

		if _, writeErr := fmt.Fprintf(w, "data: %s\n\n", jsonData); writeErr != nil {
			return writeErr
		}
		flusher.Flush()
		return nil
	})

	if err != nil {
		log.Printf("Ошибка при обработке запроса: %v", err)
		errorData := map[string]string{"error": err.Error()}
		jsonData, _ := json.Marshal(errorData)
		fmt.Fprintf(w, "data: %s\n\n", jsonData)
		flusher.Flush()
		return
	}

	// Отправка сигнала завершения
	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()
}

// HandleChat обрабатывает запросы к /api/chat (альтернативный вариант)
func HandleChat(client *gigachat.Client) http.HandlerFunc {
	handler := NewChatHandler(client)
	return handler.ServeHTTP
}

