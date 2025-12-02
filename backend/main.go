package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/nnk/97-aic/backend/api"
	"github.com/nnk/97-aic/backend/config"
	"github.com/nnk/97-aic/backend/gigachat"
	"github.com/nnk/97-aic/backend/logger"
	"github.com/nnk/97-aic/backend/storage"
)

func main() {
	// Загрузка конфигурации
	configPath := "config.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		// Инициализируем логгер с дефолтными настройками для вывода ошибки
		logger.Init(config.DefaultLogLevel, false)
		logger.Error("ошибка загрузки конфига", "error", err, "path", configPath)
		os.Exit(1)
	}

	// Инициализация логгера с настройками из конфига
	logger.Init(cfg.LogLevel, cfg.LogFormat == "json")
	logger.Info("конфигурация загружена",
		"port", cfg.Port,
		"log_level", cfg.LogLevel,
		"database_path", cfg.DatabasePath,
	)

	// Инициализация хранилища
	store, err := storage.New(cfg.DatabasePath)
	if err != nil {
		logger.Error("ошибка инициализации хранилища", "error", err)
		os.Exit(1)
	}
	logger.Info("хранилище инициализировано", "path", cfg.DatabasePath)

	// Создание клиента GigaChat (с автообновлением токена)
	var gigachatClient *gigachat.Client
	if cfg.GigaChatAccessToken != "" {
		gigachatClient = gigachat.NewClientWithToken(cfg.GigaChatAccessToken, cfg.GigaChatAPIURL)
		logger.Info("GigaChat клиент создан с готовым токеном")
	} else {
		gigachatClient = gigachat.NewClient(cfg.GigaChatAuthKey, cfg.GigaChatAPIURL, cfg.GigaChatAuthURL)
		logger.Info("GigaChat клиент создан с автообновлением токена")
	}

	// Настройка handlers
	chatHandler := api.NewChatHandler(gigachatClient, store)
	historyHandler := api.NewHistoryHandler(store, cfg)
	logsHandler := api.NewLogsHandler(store, cfg)
	healthHandler := api.NewHealthHandler(store)

	// Раздача статики
	staticDir := filepath.Join(".", "static")
	if _, err := os.Stat(staticDir); os.IsNotExist(err) {
		logger.Warn("директория static не найдена, создаю пустую", "path", staticDir)
		os.MkdirAll(staticDir, 0755)
	}
	fs := http.FileServer(http.Dir(staticDir))

	// Маршрутизатор
	mux := http.NewServeMux()
	mux.Handle("/api/chat", chatHandler)
	mux.Handle("/api/history", historyHandler)
	mux.Handle("/api/logs", logsHandler)
	mux.Handle("/health", healthHandler)
	mux.Handle("/", fs)

	// Применяем middleware
	var handler http.Handler = mux
	handler = api.LimitBodyMiddleware(cfg.MaxRequestBodySize, handler)
	handler = api.CORSMiddleware(cfg, handler)

	// Создание сервера
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second, // Больше для streaming
		IdleTimeout:  60 * time.Second,
	}

	// Канал для graceful shutdown
	done := make(chan bool, 1)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		logger.Info("получен сигнал завершения, начинаю graceful shutdown...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		server.SetKeepAlivesEnabled(false)
		if err := server.Shutdown(ctx); err != nil {
			logger.Error("ошибка graceful shutdown", "error", err)
		}

		// Закрываем хранилище
		if err := store.Close(); err != nil {
			logger.Error("ошибка закрытия хранилища", "error", err)
		}

		close(done)
	}()

	// Запуск сервера
	logger.Info("сервер запущен", "addr", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("ошибка запуска сервера", "error", err)
		os.Exit(1)
	}

	<-done
	logger.Info("сервер остановлен")
}
