<script lang="ts">
  import { sendMessageV2 } from './api';
  import type { ChatRequestV2 } from './api';

  interface TemperatureResult {
    temperature: number;
    response: string;
    loading: boolean;
    error?: string;
  }

  interface Props {
    selectedProvider?: string;
    selectedModel?: string;
    systemPrompt?: string;
  }

  let {
    selectedProvider = '',
    selectedModel = '',
    systemPrompt = ''
  }: Props = $props();

  let testMessage: string = $state('');
  let results: TemperatureResult[] = $state([]);
  let isRunning: boolean = $state(false);

  const temperatures = [0, 0.7, 1.2];

  async function runTest() {
    if (!testMessage.trim()) {
      alert('Введите сообщение для тестирования');
      return;
    }

    isRunning = true;
    results = temperatures.map(temp => ({
      temperature: temp,
      response: '',
      loading: true,
    }));

    // Запускаем запросы параллельно
    const promises = temperatures.map(async (temp, index) => {
      try {
        const request: ChatRequestV2 = {
          message: testMessage,
          provider: selectedProvider || undefined,
          model: selectedModel || undefined,
          system_prompt: systemPrompt || undefined,
          temperature: temp,
        };

        let response = '';
        for await (const chunk of sendMessageV2(request)) {
          response += chunk;
          // Обновляем результат в реальном времени
          results[index] = {
            temperature: temp,
            response: response,
            loading: false,
          };
          results = [...results]; // Триггер реактивности
        }

        results[index] = {
          temperature: temp,
          response: response,
          loading: false,
        };
      } catch (error) {
        results[index] = {
          temperature: temp,
          response: '',
          loading: false,
          error: error instanceof Error ? error.message : 'Неизвестная ошибка',
        };
      }
      results = [...results]; // Триггер реактивности
    });

    await Promise.all(promises);
    isRunning = false;
  }

  function clearResults() {
    results = [];
    testMessage = '';
  }
</script>

<div class="temperature-test">
  <h3>Тест температуры</h3>
  <p class="description">
    Запустите один запрос с разными значениями температуры для сравнения результатов
  </p>

  <div class="test-controls">
    <textarea
      class="test-input"
      placeholder="Введите сообщение для тестирования..."
      bind:value={testMessage}
      disabled={isRunning}
      rows="3"
    ></textarea>
    <div class="buttons">
      <button
        class="run-btn"
        onclick={runTest}
        disabled={isRunning || !testMessage.trim()}
      >
        {isRunning ? 'Запуск...' : 'Запустить тест'}
      </button>
      <button
        class="clear-btn"
        onclick={clearResults}
        disabled={isRunning}
      >
        Очистить
      </button>
    </div>
  </div>

  {#if results.length > 0}
    <div class="results-container">
      <h4>Результаты сравнения</h4>
      <div class="results-grid">
        {#each results as result (result.temperature)}
          <div class="result-card">
            <div class="result-header">
              <span class="temp-label">Температура: {result.temperature}</span>
              <span class="temp-badge" class:low={result.temperature === 0} class:medium={result.temperature === 0.7} class:high={result.temperature === 1.2}>
                {result.temperature === 0 ? 'Точность' :
                 result.temperature === 0.7 ? 'Баланс' : 'Креативность'}
              </span>
            </div>
            <div class="result-content">
              {#if result.loading}
                <div class="loading">Загрузка...</div>
              {:else if result.error}
                <div class="error">Ошибка: {result.error}</div>
              {:else}
                <div class="response-text">{result.response || '(пустой ответ)'}</div>
              {/if}
            </div>
            {#if !result.loading && !result.error && result.response}
              <div class="result-stats">
                Символов: {result.response.length}
              </div>
            {/if}
          </div>
        {/each}
      </div>

      <div class="comparison-section">
        <h4>Анализ результатов</h4>
        <div class="comparison-text">
          {#if results.every(r => !r.loading && !r.error && r.response)}
            <p><strong>Температура 0:</strong> Должна давать наиболее точные и детерминированные ответы, минимальная вариативность.</p>
            <p><strong>Температура 0.7:</strong> Баланс между точностью и креативностью, подходит для большинства задач.</p>
            <p><strong>Температура 1.2:</strong> Более креативные и разнообразные ответы, выше вариативность.</p>
          {/if}
        </div>
      </div>
    </div>
  {/if}
</div>

<style>
  .temperature-test {
    padding: 16px;
    background-color: var(--background);
  }

  h3 {
    font-size: 14px;
    font-weight: 600;
    color: var(--foreground);
    margin: 0 0 8px 0;
  }

  .description {
    font-size: 12px;
    color: var(--muted-foreground);
    margin: 0 0 16px 0;
    line-height: 1.4;
  }

  .test-controls {
    margin-bottom: 20px;
  }

  .test-input {
    width: 100%;
    padding: 10px;
    border: 1px solid var(--border);
    border-radius: 6px;
    background-color: var(--background);
    color: var(--foreground);
    font-size: 13px;
    font-family: inherit;
    resize: vertical;
    margin-bottom: 12px;
  }

  .test-input:focus {
    outline: none;
    border-color: var(--primary);
  }

  .test-input:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .buttons {
    display: flex;
    gap: 8px;
  }

  .run-btn,
  .clear-btn {
    padding: 8px 16px;
    border: 1px solid var(--border);
    border-radius: 6px;
    font-size: 13px;
    cursor: pointer;
    transition: all 0.2s;
  }

  .run-btn {
    background-color: var(--primary);
    color: var(--primary-foreground);
    flex: 1;
  }

  .run-btn:hover:not(:disabled) {
    opacity: 0.9;
  }

  .run-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .clear-btn {
    background-color: var(--secondary);
    color: var(--secondary-foreground);
  }

  .clear-btn:hover:not(:disabled) {
    background-color: var(--accent);
  }

  .clear-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .results-container {
    margin-top: 20px;
  }

  .results-container h4 {
    font-size: 13px;
    font-weight: 600;
    color: var(--foreground);
    margin: 0 0 12px 0;
  }

  .results-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
    gap: 16px;
    margin-bottom: 20px;
  }

  .result-card {
    border: 1px solid var(--border);
    border-radius: 8px;
    background-color: var(--muted);
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }

  .result-header {
    padding: 10px 12px;
    background-color: var(--background);
    border-bottom: 1px solid var(--border);
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .temp-label {
    font-size: 12px;
    font-weight: 600;
    color: var(--foreground);
  }

  .temp-badge {
    font-size: 10px;
    padding: 2px 8px;
    border-radius: 12px;
    font-weight: 500;
  }

  .temp-badge.low {
    background-color: #3b82f6;
    color: white;
  }

  .temp-badge.medium {
    background-color: #10b981;
    color: white;
  }

  .temp-badge.high {
    background-color: #f59e0b;
    color: white;
  }

  .result-content {
    padding: 12px;
    flex: 1;
    min-height: 150px;
    max-height: 400px;
    overflow-y: auto;
  }

  .loading {
    color: var(--muted-foreground);
    font-size: 12px;
    font-style: italic;
  }

  .error {
    color: var(--destructive);
    font-size: 12px;
  }

  .response-text {
    font-size: 13px;
    color: var(--foreground);
    line-height: 1.5;
    white-space: pre-wrap;
    word-wrap: break-word;
  }

  .result-stats {
    padding: 8px 12px;
    background-color: var(--background);
    border-top: 1px solid var(--border);
    font-size: 11px;
    color: var(--muted-foreground);
  }

  .comparison-section {
    margin-top: 20px;
    padding: 16px;
    background-color: var(--muted);
    border-radius: 8px;
  }

  .comparison-section h4 {
    margin-top: 0;
  }

  .comparison-text {
    font-size: 12px;
    color: var(--foreground);
    line-height: 1.6;
  }

  .comparison-text p {
    margin: 8px 0;
  }

  @media (max-width: 768px) {
    .results-grid {
      grid-template-columns: 1fr;
    }
  }
</style>

