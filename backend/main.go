package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/nnk/97-aic/backend/api"
	"github.com/nnk/97-aic/backend/config"
	"github.com/nnk/97-aic/backend/gigachat"
)

func main() {
	// Загрузка конфигурации
	configPath := "config.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Ошибка загрузки конфига: %v", err)
	}

	// Получение Access Token
	accessToken := cfg.GigaChatAccessToken
	if accessToken == "" && cfg.GigaChatAuthKey != "" {
		log.Println("Получение Access Token из Authorization Key...")
		accessToken, err = gigachat.GetAccessToken(cfg.GigaChatAuthKey, cfg.GigaChatAuthURL)
		if err != nil {
			log.Fatalf("Ошибка получения токена: %v", err)
		}
		log.Println("Access Token успешно получен")
	}

	// Создание клиента GigaChat
	gigachatClient := gigachat.NewClient(accessToken, cfg.GigaChatAPIURL)

	// Настройка маршрутов
	// API endpoint для чата
	chatHandler := api.NewChatHandler(gigachatClient)

	// Раздача статики
	staticDir := filepath.Join(".", "static")
	if _, err := os.Stat(staticDir); os.IsNotExist(err) {
		log.Printf("Предупреждение: директория static не найдена, создаю пустую")
		os.MkdirAll(staticDir, 0755)
	}
	fs := http.FileServer(http.Dir(staticDir))

	// Главный обработчик - проверяет путь и направляет в нужный handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// API запросы обрабатываем отдельно
		if r.URL.Path == "/api/chat" {
			chatHandler.ServeHTTP(w, r)
			return
		}
		
		// Для остальных - статика
		fs.ServeHTTP(w, r)
	})

	// Запуск сервера
	addr := ":" + cfg.Port
	log.Printf("Сервер запущен на %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}

