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

// ===== API v2 (новые типы) =====

// Режимы рассуждения (День 4)
export type ReasoningMode = 'direct' | 'step_by_step' | 'experts';

export interface ReasoningModeInfo {
  id: ReasoningMode;
  name: string;
  description: string;
}

// Информация о провайдере
export interface ProviderInfo {
  name: string;
  models: string[];
  current_model: string;
  is_default: boolean;
}

// Ответ API /api/v2/providers
export interface ProvidersResponse {
  providers: ProviderInfo[];
  default_provider: string;
  reasoning_modes: ReasoningModeInfo[];
}

// Запрос к API v2
export interface ChatRequestV2 {
  message: string;
  session_id?: string;
  use_history?: boolean;
  provider?: string;
  model?: string;
  system_prompt?: string;
  reasoning_mode?: ReasoningMode;
  json_format?: boolean;
  json_schema?: string;
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
  tokens_input?: number;
  tokens_output?: number;
  tokens_total?: number;
  cost?: number;
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

// ===== API v2 функции =====

// Получение списка провайдеров
export async function fetchProviders(): Promise<ProvidersResponse> {
  const response = await fetch('/api/v2/providers');
  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status}`);
  }
  return await response.json();
}

// Отправка сообщения через API v2 (streaming)
// ===== API для тестирования токенов =====

export type TokenTestType = 'short' | 'long' | 'exceed_limit' | 'all';

export interface TokenTestRequest {
  provider: string;
  model?: string;
  test_type: TokenTestType;
}

export interface TokenTestResult {
  test_type: string;
  message: string;
  response: string;
  tokens_input: number;
  tokens_output: number;
  tokens_total: number;
  cost: number;
  duration_ms: number;
  success: boolean;
  error?: string;
  max_tokens: number;
}

export interface TokenTestSummary {
  total_tests: number;
  success_count: number;
  error_count: number;
  total_tokens: number;
  total_cost: number;
  avg_duration_ms: number;
}

export interface TokenTestResponse {
  provider: string;
  model: string;
  results: TokenTestResult[];
  summary: TokenTestSummary;
}

export async function* sendMessageV2(
  request: ChatRequestV2
): AsyncGenerator<string, void, unknown> {
  const response = await fetch('/api/v2/chat', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(request),
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

// Тестирование токенов
export async function testTokens(request: TokenTestRequest): Promise<TokenTestResponse> {
  const response = await fetch('/api/v2/token-test', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(request),
  });

  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(`HTTP error! status: ${response.status}, body: ${errorText}`);
  }

  return await response.json();
}
