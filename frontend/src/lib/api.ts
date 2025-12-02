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

export interface RequestLog {
  id: number;
  session_id: string;
  request_json: string;
  response_json: string;
  status_code: number;
  duration_ms: number;
  created_at: string;
}

export async function* sendMessage(
  message: string,
  jsonConfig?: JSONResponseConfig
): AsyncGenerator<string, void, unknown> {
  const body: any = { message };
  if (jsonConfig) {
    body.response_json = jsonConfig;
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

