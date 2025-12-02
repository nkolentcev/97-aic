<script lang="ts">
  import { onMount } from 'svelte';
  import ChatMessage from './lib/ChatMessage.svelte';
  import ChatInput from './lib/ChatInput.svelte';
  import LogsTable from './lib/LogsTable.svelte';
  import ThemeDropdown from './lib/ThemeDropdown.svelte';
  import JSONFormatConfig from './lib/JSONFormatConfig.svelte';
  import { sendMessage, fetchLogs } from './lib/api';
  import { theme } from './lib/theme';
  import type { ChatMessage as ChatMessageType, RequestLog, JSONResponseConfig } from './lib/api';

  let messages: ChatMessageType[] = $state([]);
  let logs: RequestLog[] = $state([]);
  let loading: boolean = $state(false);
  let error: string | null = $state(null);
  let currentAssistantMessage: string = $state('');
  let jsonFormatEnabled: boolean = $state(false);
  let jsonSchema: string = $state('');

  onMount(() => {
    // Инициализируем тему
    theme.subscribe(() => {});
    loadLogs();
  });

  async function loadLogs() {
    try {
      logs = await fetchLogs(50);
    } catch (e) {
      console.error('Ошибка загрузки логов:', e);
    }
  }

  async function handleSend(userMessage: string) {
    messages = [...messages, { role: 'user', content: userMessage }];
    loading = true;
    error = null;
    currentAssistantMessage = '';

    try {
      // Формируем конфиг JSON-ответа
      let jsonConfig: JSONResponseConfig | undefined;
      if (jsonFormatEnabled && jsonSchema.trim()) {
        jsonConfig = {
          enabled: true,
          schema_text: jsonSchema.trim(),
        };
      }

      for await (const chunk of sendMessage(userMessage, jsonConfig)) {
        currentAssistantMessage += chunk;
        const lastMessage = messages[messages.length - 1];
        if (lastMessage && lastMessage.role === 'assistant') {
          messages[messages.length - 1] = {
            role: 'assistant',
            content: currentAssistantMessage,
          };
        } else {
          messages = [...messages, { role: 'assistant', content: currentAssistantMessage }];
        }
        messages = [...messages];
      }

      if (currentAssistantMessage) {
        const lastMessage = messages[messages.length - 1];
        if (lastMessage && lastMessage.role === 'assistant') {
          messages[messages.length - 1] = {
            role: 'assistant',
            content: currentAssistantMessage,
          };
        } else {
          messages = [...messages, { role: 'assistant', content: currentAssistantMessage }];
        }
      }

      // Обновляем логи после успешного запроса
      await loadLogs();
    } catch (e) {
      error = e instanceof Error ? e.message : 'Произошла ошибка';
      console.error('Ошибка отправки сообщения:', e);
    } finally {
      loading = false;
      currentAssistantMessage = '';
    }
  }
</script>

<div class="app">
  <div class="header">
    <h1>GigaChat</h1>
    <ThemeDropdown />
  </div>

  <div class="main-content">
    <!-- Левая панель - конфигурация и логи -->
    <div class="logs-panel">
      <!-- Верхняя часть - конфигурация JSON -->
      <div class="config-section">
        <JSONFormatConfig bind:enabled={jsonFormatEnabled} bind:jsonSchema={jsonSchema} />
      </div>
      
      <!-- Нижняя часть - таблица логов -->
      <div class="logs-section">
        <LogsTable {logs} />
      </div>
    </div>

    <!-- Правая панель - чат -->
    <div class="chat-panel">
      <div class="messages-container">
        {#if messages.length === 0}
          <div class="welcome">
            <p>Добро пожаловать! Задайте вопрос, и я постараюсь помочь.</p>
          </div>
        {/if}

        {#each messages as message (message)}
          <ChatMessage role={message.role} content={message.content} />
        {/each}

        {#if loading && currentAssistantMessage}
          <ChatMessage role="assistant" content={currentAssistantMessage} />
        {/if}

        {#if error}
          <div class="error">
            <p>Ошибка: {error}</p>
          </div>
        {/if}
      </div>

      <ChatInput onsend={handleSend} {loading} disabled={loading} />
    </div>
  </div>
</div>

<style>
  .app {
    display: flex;
    flex-direction: column;
    height: 100vh;
    background-color: var(--background);
  }

  .header {
    padding: 12px 24px;
    border-bottom: 1px solid var(--border);
    background-color: var(--background);
    flex-shrink: 0;
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .header h1 {
    font-size: 20px;
    font-weight: 400;
    color: var(--foreground);
    margin: 0;
  }

  .main-content {
    display: flex;
    flex: 1;
    overflow: hidden;
  }

  .logs-panel {
    width: 50%;
    border-right: 1px solid var(--border);
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }

  .config-section {
    flex-shrink: 0;
    border-bottom: 1px solid var(--border);
    overflow-y: auto;
    max-height: 40%;
  }

  .logs-section {
    flex: 1;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    min-height: 0;
  }

  .chat-panel {
    width: 50%;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .messages-container {
    flex: 1;
    overflow-y: auto;
    padding: 16px 0;
  }

  .welcome {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    padding: 24px;
  }

  .welcome p {
    color: var(--muted-foreground);
    font-size: 16px;
  }

  .error {
    padding: 16px 24px;
    background-color: var(--destructive);
    color: var(--destructive-foreground);
    margin: 8px 24px;
    border-radius: 8px;
  }

  .error p {
    margin: 0;
  }

  /* Адаптивность для мобильных */
  @media (max-width: 768px) {
    .main-content {
      flex-direction: column;
    }

    .logs-panel {
      width: 100%;
      height: 40%;
      border-right: none;
      border-bottom: 1px solid var(--border);
    }

    .chat-panel {
      width: 100%;
      height: 60%;
    }
  }
</style>
