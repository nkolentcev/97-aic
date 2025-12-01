# API Документация

## Endpoints

### POST /api/chat

Отправляет сообщение в GigaChat API и возвращает streaming ответ.

#### Запрос

```json
{
  "message": "Ваш вопрос здесь"
}
```

#### Ответ

Ответ приходит в формате Server-Sent Events (SSE) с типом `text/event-stream`.

Каждое событие имеет формат:
```
data: {"content": "часть ответа"}
```

Когда ответ завершен, отправляется:
```
data: [DONE]
```

#### Пример использования

```javascript
const response = await fetch('/api/chat', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
  },
  body: JSON.stringify({ message: 'Привет!' }),
});

const reader = response.body.getReader();
const decoder = new TextDecoder();

while (true) {
  const { done, value } = await reader.read();
  if (done) break;
  
  const text = decoder.decode(value);
  // Обработка streaming данных
}
```

#### Коды ошибок

- `400 Bad Request` - неверный формат запроса или отсутствует поле `message`
- `405 Method Not Allowed` - используется неверный HTTP метод
- `500 Internal Server Error` - ошибка при обращении к GigaChat API

#### Ошибки в потоке

Если происходит ошибка во время обработки, в поток отправляется:
```json
{
  "error": "описание ошибки"
}
```

