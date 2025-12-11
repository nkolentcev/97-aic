package provider

import (
	"strings"
	"unicode/utf8"
)

// CountTokens приблизительно подсчитывает количество токенов в тексте
// Использует эмпирическое правило: ~4 символа на токен для русского/английского текста
// Это приблизительная оценка, точное значение зависит от модели токенизатора
func CountTokens(text string) int {
	if text == "" {
		return 0
	}

	// Подсчитываем количество символов (включая пробелы)
	charCount := utf8.RuneCountInString(text)

	// Приблизительно: 4 символа = 1 токен
	// Для более точного подсчета можно использовать библиотеку tiktoken,
	// но для большинства случаев это достаточно
	tokens := charCount / 4

	// Минимум 1 токен для непустого текста
	if tokens < 1 && charCount > 0 {
		tokens = 1
	}

	return tokens
}

// CountTokensForMessages подсчитывает общее количество токенов для списка сообщений
// Включает системный промпт, историю и текущее сообщение
func CountTokensForMessages(systemPrompt string, history []Message, currentMessage string) int {
	total := 0

	// Системный промпт
	if systemPrompt != "" {
		total += CountTokens(systemPrompt)
	}

	// История сообщений
	for _, msg := range history {
		// Роль тоже считается (обычно ~1-2 токена)
		total += CountTokens(msg.Role)
		total += CountTokens(msg.Content)
		// Добавляем небольшой overhead для форматирования (~2 токена на сообщение)
		total += 2
	}

	// Текущее сообщение
	total += CountTokens(currentMessage)

	// Overhead для структуры запроса (~10 токенов)
	total += 10

	return total
}

// GenerateTextForTokens генерирует текст заданной длины в токенах
// Используется для создания тестовых запросов разной длины
func GenerateTextForTokens(targetTokens int, baseText string) string {
	// Базовый текст
	if baseText == "" {
		baseText = "Это тестовое сообщение для проверки обработки токенов. "
	}

	// Вычисляем, сколько раз нужно повторить базовый текст
	baseTokens := CountTokens(baseText)
	if baseTokens == 0 {
		return strings.Repeat("Слово ", targetTokens*4) // Fallback: простое повторение
	}

	// Количество повторений с учетом targetTokens
	repeatCount := (targetTokens / baseTokens) + 1

	// Генерируем текст и обрезаем до нужного количества токенов
	generated := strings.Repeat(baseText, repeatCount)

	// Обрезаем до нужного количества токенов
	currentTokens := CountTokens(generated)
	if currentTokens > targetTokens {
		// Упрощенное обрезание: удаляем по 4 символа за раз
		charsToRemove := (currentTokens - targetTokens) * 4
		if charsToRemove < utf8.RuneCountInString(generated) {
			runes := []rune(generated)
			generated = string(runes[:len(runes)-charsToRemove])
		}
	}

	return generated
}
