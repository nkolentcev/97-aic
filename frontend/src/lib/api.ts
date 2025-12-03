export interface ChatMessage {
  role: 'user' | 'assistant';
  content: string;
}

export interface ChatResponse {
  content?: string;
  error?: string;
}

export interface JSONResponseConfig {
  enabled: boolean;
  schema_text?: string;  // Текст JSON-структуры из текстового поля
}

// Конфигурация режима сбора требований
export interface CollectConfig {
  enabled: boolean;
  role?: string;              // Роль модели (например, "технический аналитик")
  goal?: string;              // Цель сбора (например, "ТЗ на мобильное приложение")
  required_questions?: string[]; // Список обязательных вопросов
  output_format?: string;     // Формат финального результата
}

// Ответ режима сбора требований
export interface CollectResponse {
  session_id: string;
  status: 'collecting' | 'ready' | 'error' | 'raw';
  question?: string;          // Следующий вопрос
  collected?: string[];       // Собранные данные
  result?: string;            // Финальный результат
  error?: string;             // Ошибка
  raw_response?: string;      // Сырой ответ модели
}

// Расширенные опции чата
export interface ChatOptions {
  system_prompt?: string;
  json_config?: JSONResponseConfig;
  collect_config?: CollectConfig;
  max_tokens?: number;
  temperature?: number;
}

export interface RequestLog {
  id: number;
  session_id: string;
  request_json: string;
  response_json: string;
  status_code: number;
  duration_ms: number;
  created_at: string;
}

export interface SendMessageOptions {
  jsonConfig?: JSONResponseConfig;
  sessionId?: string;
  useHistory?: boolean;
  options?: ChatOptions;
}

export async function* sendMessage(
  message: string,
  optionsOrJsonConfig?: JSONResponseConfig | SendMessageOptions
): AsyncGenerator<string, void, unknown> {
  const body: any = { message };
  
  // Поддержка старого API (только JSONResponseConfig) и нового (SendMessageOptions)
  if (optionsOrJsonConfig) {
    if ('enabled' in optionsOrJsonConfig) {
      // Это JSONResponseConfig (старый формат)
      body.response_json = optionsOrJsonConfig;
    } else {
      // Это SendMessageOptions (новый формат)
      const opts = optionsOrJsonConfig as SendMessageOptions;
      if (opts.jsonConfig) {
        body.response_json = opts.jsonConfig;
      }
      if (opts.sessionId) {
        body.session_id = opts.sessionId;
      }
      if (opts.useHistory) {
        body.use_history = opts.useHistory;
      }
      if (opts.options) {
        body.options = opts.options;
      }
    }
  }

  const response = await fetch('/api/chat', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(body),
  });

  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status}`);
  }

  const reader = response.body?.getReader();
  if (!reader) {
    throw new Error('Response body is not readable');
  }

  const decoder = new TextDecoder();
  let buffer = '';

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;

    buffer += decoder.decode(value, { stream: true });
    const lines = buffer.split('\n');
    buffer = lines.pop() || '';

    for (const line of lines) {
      if (line.startsWith('data: ')) {
        const data = line.slice(6).trim();
        if (data === '[DONE]') {
          return;
        }

        try {
          const parsed: ChatResponse = JSON.parse(data);
          if (parsed.content) {
            yield parsed.content;
          } else if (parsed.error) {
            throw new Error(parsed.error);
          }
        } catch (e) {
          // Игнорируем ошибки парсинга отдельных чанков
        }
      }
    }
  }
}

// Отправка сообщения в режиме сбора требований
export async function sendCollectMessage(
  message: string,
  sessionId?: string,
  collectConfig?: CollectConfig,
  startNewSession: boolean = false
): Promise<CollectResponse> {
  const body: any = { 
    message,
    start_new_session: startNewSession,
  };
  
  if (sessionId) {
    body.session_id = sessionId;
  }
  
  if (collectConfig) {
    body.collect_config = collectConfig;
  }

  const response = await fetch('/api/chat/collect', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(body),
  });

  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(`HTTP error! status: ${response.status}, body: ${errorText}`);
  }

  return await response.json();
}

export async function fetchLogs(limit: number = 50): Promise<RequestLog[]> {
  const response = await fetch(`/api/logs?limit=${limit}`);
  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status}`);
  }
  const data = await response.json();
  return data || [];
}

export async function fetchHistory(sessionId: string, limit: number = 100): Promise<ChatMessage[]> {
  const response = await fetch(`/api/history?session_id=${sessionId}&limit=${limit}`);
  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status}`);
  }
  const data = await response.json();
  return data || [];
}

