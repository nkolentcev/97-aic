<script lang="ts">
  import { onMount } from 'svelte';
  import ChatMessage from './lib/ChatMessage.svelte';
  import ChatInput from './lib/ChatInput.svelte';
  import LogsTable from './lib/LogsTable.svelte';
  import ThemeDropdown from './lib/ThemeDropdown.svelte';
  import JSONFormatConfig from './lib/JSONFormatConfig.svelte';
  import CollectModeConfig from './lib/CollectModeConfig.svelte';
  import ProviderConfig from './lib/ProviderConfig.svelte';
  import TemperatureTest from './lib/TemperatureTest.svelte';
  import { sendMessage, sendCollectMessage, fetchLogs, fetchProviders, sendMessageV2 } from './lib/api';
  import { theme } from './lib/theme';
  import type { ChatMessage as ChatMessageType, RequestLog, JSONResponseConfig, CollectConfig, CollectResponse, ProviderInfo, ReasoningModeInfo, ReasoningMode } from './lib/api';

  let messages: ChatMessageType[] = $state([]);
  let logs: RequestLog[] = $state([]);
  let loading: boolean = $state(false);
  let error: string | null = $state(null);
  let currentAssistantMessage: string = $state('');
  let jsonFormatEnabled: boolean = $state(false);
  let jsonSchema: string = $state('');

  // Вкладки левой панели
  type LeftPanelTab = 'config' | 'logs' | 'temperature';
  let leftPanelTab: LeftPanelTab = $state('config');

  // Режим сбора требований
  let collectModeEnabled: boolean = $state(false);
  let collectConfig: CollectConfig = $state({
    enabled: false,
    role: '',
    goal: '',
    required_questions: [],
    output_format: ''
  });
  let collectSessionId: string | null = $state(null);
  let collectStatus: string = $state('');
  let collectResult: string | null = $state(null);

  // Провайдеры и настройки (API v2)
  let providers: ProviderInfo[] = $state([]);
  let reasoningModes: ReasoningModeInfo[] = $state([
    { id: 'direct', name: 'Прямой ответ', description: 'Краткий ответ без рассуждений' },
    { id: 'step_by_step', name: 'Пошаговое решение', description: 'Разбивает задачу на шаги' },
    { id: 'experts', name: 'Группа экспертов', description: 'Несколько экспертов дают мнения' }
  ]);
  let selectedProvider: string = $state('');
  let selectedModel: string = $state('');
  let selectedReasoningMode: ReasoningMode = $state('direct');
  let systemPrompt: string = $state('');
  let temperature: number = $state(0.7);
  let useApiV2: boolean = $state(true); // Использовать новый API

  onMount(async () => {
    // Инициализируем тему
    theme.subscribe(() => {});
    loadLogs();

    // Загружаем провайдеры
    try {
      const data = await fetchProviders();
      providers = data.providers;
      reasoningModes = data.reasoning_modes;
      selectedProvider = data.default_provider;
      // Устанавливаем модель по умолчанию
      const defaultP = providers.find(p => p.name === selectedProvider);
      if (defaultP) {
        selectedModel = defaultP.current_model || defaultP.models[0] || '';
      }
      useApiV2 = true;
    } catch (e) {
      console.warn('API v2 недоступен, используем legacy API:', e);
      useApiV2 = false;
    }
  });

  async function loadLogs() {
    try {
      logs = await fetchLogs(50);
    } catch (e) {
      console.error('Ошибка загрузки логов:', e);
    }
  }

  function startNewCollectSession() {
    collectSessionId = null;
    collectStatus = '';
    collectResult = null;
    messages = [];
    error = null;
  }

  async function handleSend(userMessage: string) {
    messages = [...messages, { role: 'user', content: userMessage }];
    loading = true;
    error = null;
    currentAssistantMessage = '';

    try {
      // Режим сбора требований
      if (collectModeEnabled) {
        const startNew = collectSessionId === null;
        const response: CollectResponse = await sendCollectMessage(
          userMessage,
          collectSessionId || undefined,
          { ...collectConfig, enabled: true },
          startNew
        );

        collectSessionId = response.session_id;
        collectStatus = response.status;

        let assistantContent = '';
        if (response.status === 'collecting' && response.question) {
          assistantContent = response.question;
        } else if (response.status === 'ready' && response.result) {
          assistantContent = '✅ **Результат готов!**\n\n' + response.result;
          collectResult = response.result;
        } else if (response.status === 'error') {
          throw new Error(response.error || 'Неизвестная ошибка');
        } else if (response.status === 'raw') {
          assistantContent = response.raw_response || response.result || 'Получен ответ';
        }

        if (assistantContent) {
          messages = [...messages, { role: 'assistant', content: assistantContent }];
        }
      } else if (useApiV2 && providers.length > 0) {
        // Новый API v2 с поддержкой провайдеров
        const request = {
          message: userMessage,
          provider: selectedProvider,
          model: selectedModel,
          system_prompt: systemPrompt || undefined,
          reasoning_mode: selectedReasoningMode !== 'direct' ? selectedReasoningMode : undefined,
          json_format: jsonFormatEnabled && jsonSchema.trim() ? true : undefined,
          json_schema: jsonFormatEnabled && jsonSchema.trim() ? jsonSchema.trim() : undefined,
          temperature: temperature >= 0 ? temperature : undefined,
        };

        for await (const chunk of sendMessageV2(request)) {
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
      } else {
        // Legacy API (обычный режим чата)
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
    <h1>AI Chat</h1>
    <ThemeDropdown />
  </div>

  <div class="main-content">
    <!-- Левая панель -->
    <div class="left-panel">
      <!-- Табы панели -->
      <div class="panel-tabs">
        <button
          class="panel-tab"
          class:active={leftPanelTab === 'config'}
          onclick={() => leftPanelTab = 'config'}
        >
          Настройки
        </button>
        <button
          class="panel-tab"
          class:active={leftPanelTab === 'temperature'}
          onclick={() => leftPanelTab = 'temperature'}
        >
          Тест температуры
        </button>
        <button
          class="panel-tab"
          class:active={leftPanelTab === 'logs'}
          onclick={() => { leftPanelTab = 'logs'; loadLogs(); }}
        >
          Логи
        </button>
      </div>

      <!-- Содержимое вкладок -->
      <div class="panel-content">
        {#if leftPanelTab === 'config'}
          <!-- Вкладка настроек -->
          <div class="config-section">
            <!-- Переключатель режимов чата -->
            <div class="mode-tabs">
              <button
                class="mode-tab"
                class:active={!collectModeEnabled}
                onclick={() => { collectModeEnabled = false; }}
              >
                Обычный чат
              </button>
              <button
                class="mode-tab"
                class:active={collectModeEnabled}
                onclick={() => { collectModeEnabled = true; }}
              >
                Сбор требований
              </button>
            </div>

            {#if collectModeEnabled}
              <CollectModeConfig bind:enabled={collectConfig.enabled} bind:config={collectConfig} />

              {#if collectSessionId}
                <div class="collect-status">
                  <div class="status-row">
                    <span class="status-label">Сессия:</span>
                    <code class="session-id">{collectSessionId}</code>
                  </div>
                  <div class="status-row">
                    <span class="status-label">Статус:</span>
                    <span class="status-value" class:ready={collectStatus === 'ready'}>
                      {collectStatus === 'collecting' ? '🔄 Сбор данных...' :
                       collectStatus === 'ready' ? '✅ Готово' : collectStatus}
                    </span>
                  </div>
                  <button class="new-session-btn" onclick={startNewCollectSession}>
                    Начать новую сессию
                  </button>
                </div>
              {/if}
            {:else}
              <!-- Конфигурация провайдера (API v2) -->
              {#if useApiV2 && providers.length > 0}
                <ProviderConfig
                  {providers}
                  {reasoningModes}
                  {selectedProvider}
                  {selectedModel}
                  {selectedReasoningMode}
                  {systemPrompt}
                  {temperature}
                  onProviderChange={(p) => selectedProvider = p}
                  onModelChange={(m) => selectedModel = m}
                  onReasoningModeChange={(r) => selectedReasoningMode = r}
                  onSystemPromptChange={(s) => systemPrompt = s}
                  onTemperatureChange={(t) => temperature = t}
                />
                <div class="config-divider"></div>
              {/if}

              <JSONFormatConfig bind:enabled={jsonFormatEnabled} bind:jsonSchema={jsonSchema} />
            {/if}
          </div>
        {:else if leftPanelTab === 'temperature'}
          <!-- Вкладка теста температуры -->
          <div class="temperature-section">
            <TemperatureTest
              selectedProvider={selectedProvider}
              selectedModel={selectedModel}
              systemPrompt={systemPrompt}
            />
          </div>
        {:else}
          <!-- Вкладка логов -->
          <div class="logs-section">
            <LogsTable {logs} />
          </div>
        {/if}
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

  .left-panel {
    width: 400px;
    border-right: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .panel-tabs {
    display: flex;
    border-bottom: 1px solid var(--border);
    flex-shrink: 0;
  }

  .panel-tab {
    flex: 1;
    padding: 12px 16px;
    border: none;
    background: transparent;
    color: var(--muted-foreground);
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s;
    border-bottom: 2px solid transparent;
  }

  .panel-tab:hover {
    color: var(--foreground);
    background-color: var(--muted);
  }

  .panel-tab.active {
    color: var(--foreground);
    border-bottom-color: var(--primary);
  }

  .panel-content {
    flex: 1;
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }

  .config-section {
    flex: 1;
    overflow-y: auto;
    padding: 16px;
  }

  .mode-tabs {
    display: flex;
    gap: 4px;
    margin-bottom: 16px;
    background-color: var(--muted);
    padding: 4px;
    border-radius: 8px;
  }

  .mode-tab {
    flex: 1;
    padding: 8px 12px;
    border: none;
    background: transparent;
    color: var(--muted-foreground);
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    border-radius: 6px;
    transition: all 0.2s;
  }

  .mode-tab:hover {
    color: var(--foreground);
  }

  .mode-tab.active {
    background-color: var(--background);
    color: var(--foreground);
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  }

  .collect-status {
    margin-top: 12px;
    padding: 12px;
    background-color: var(--muted);
    border-radius: 8px;
  }

  .status-row {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 8px;
  }

  .status-label {
    font-size: 12px;
    color: var(--muted-foreground);
    min-width: 60px;
  }

  .session-id {
    font-size: 11px;
    background-color: var(--background);
    padding: 2px 6px;
    border-radius: 4px;
    color: var(--foreground);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    max-width: 200px;
  }

  .status-value {
    font-size: 13px;
    color: var(--foreground);
  }

  .status-value.ready {
    color: var(--primary);
    font-weight: 500;
  }

  .new-session-btn {
    width: 100%;
    margin-top: 8px;
    padding: 8px 12px;
    background-color: var(--secondary);
    color: var(--secondary-foreground);
    border: 1px solid var(--border);
    border-radius: 6px;
    font-size: 13px;
    cursor: pointer;
    transition: background-color 0.2s;
  }

  .new-session-btn:hover {
    background-color: var(--accent);
  }

  .config-divider {
    height: 1px;
    background-color: var(--border);
    margin: 16px 0;
  }

  .logs-section {
    flex: 1;
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }

  .temperature-section {
    flex: 1;
    overflow-y: auto;
  }

  .chat-panel {
    flex: 1;
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

    .left-panel {
      width: 100%;
      height: 45%;
      border-right: none;
      border-bottom: 1px solid var(--border);
    }

    .chat-panel {
      width: 100%;
      height: 55%;
    }
  }
</style>
