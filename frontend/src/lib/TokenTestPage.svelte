<script lang="ts">
  import { onMount } from 'svelte';
  import { fetchProviders, testTokens } from './api';
  import type { ProviderInfo, TokenTestResponse, TokenTestResult, TokenTestType } from './api';

  let providers: ProviderInfo[] = $state([]);
  let selectedProvider: string = $state('');
  let selectedModel: string = $state('');
  let isRunning: boolean = $state(false);
  let results: TokenTestResponse | null = $state(null);
  let error: string | null = $state(null);

  onMount(async () => {
    try {
      const data = await fetchProviders();
      providers = data.providers;
      if (data.default_provider) {
        selectedProvider = data.default_provider;
        const defaultP = providers.find(p => p.name === selectedProvider);
        if (defaultP) {
          selectedModel = defaultP.current_model || defaultP.models[0] || '';
        }
      }
    } catch (e) {
      error = `Ошибка загрузки провайдеров: ${e instanceof Error ? e.message : String(e)}`;
    }
  });

  function onProviderChange() {
    const provider = providers.find(p => p.name === selectedProvider);
    if (provider) {
      selectedModel = provider.current_model || provider.models[0] || '';
    }
  }

  async function runTest(testType: TokenTestType) {
    if (!selectedProvider) {
      error = 'Выберите провайдера';
      return;
    }

    isRunning = true;
    error = null;
    results = null;

    try {
      const response = await testTokens({
        provider: selectedProvider,
        model: selectedModel || undefined,
        test_type: testType,
      });
      results = response;
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      isRunning = false;
    }
  }

  async function runAllTests() {
    await runTest('all');
  }

  function clearResults() {
    results = null;
    error = null;
  }

  function formatCost(cost: number): string {
    if (cost === 0) {
      return 'Бесплатно';
    }
    return `$${cost.toFixed(6)}`;
  }

  function formatDuration(ms: number): string {
    if (ms < 1000) {
      return `${ms} мс`;
    }
    return `${(ms / 1000).toFixed(2)} с`;
  }
</script>

<div class="token-test-page">
  <h2>Тестирование токенов</h2>
  <p class="description">
    Проверьте поведение модели при работе с короткими, длинными запросами и запросами, превышающими лимит токенов
  </p>

  <div class="controls">
    <div class="control-group">
      <label for="provider">Провайдер:</label>
      <select
        id="provider"
        bind:value={selectedProvider}
        onchange={onProviderChange}
        disabled={isRunning}
      >
        <option value="">Выберите провайдера</option>
        {#each providers as provider}
          <option value={provider.name}>{provider.name} {provider.is_default ? '(по умолчанию)' : ''}</option>
        {/each}
      </select>
    </div>

    <div class="control-group">
      <label for="model">Модель:</label>
      <select
        id="model"
        bind:value={selectedModel}
        disabled={isRunning || !selectedProvider}
      >
        {#if selectedProvider}
          {#each providers.find(p => p.name === selectedProvider)?.models || [] as model}
            <option value={model}>{model}</option>
          {/each}
        {/if}
      </select>
    </div>
  </div>

  <div class="test-buttons">
    <button
      class="test-btn"
      onclick={() => runTest('short')}
      disabled={isRunning || !selectedProvider}
    >
      Короткий запрос (~50 токенов)
    </button>
    <button
      class="test-btn"
      onclick={() => runTest('long')}
      disabled={isRunning || !selectedProvider}
    >
      Длинный запрос (~2000 токенов)
    </button>
    <button
      class="test-btn"
      onclick={() => runTest('exceed_limit')}
      disabled={isRunning || !selectedProvider}
    >
      Превышение лимита
    </button>
    <button
      class="test-btn primary"
      onclick={runAllTests}
      disabled={isRunning || !selectedProvider}
    >
      Запустить все тесты
    </button>
  </div>

  {#if error}
    <div class="error-message">
      Ошибка: {error}
    </div>
  {/if}

  {#if isRunning}
    <div class="loading">
      <p>Выполняется тестирование...</p>
    </div>
  {/if}

  {#if results}
    <div class="results-section">
      <h3>Результаты тестирования</h3>
      
      <div class="summary">
        <h4>Сводка</h4>
        <div class="summary-grid">
          <div class="summary-item">
            <span class="label">Всего тестов:</span>
            <span class="value">{results.summary.total_tests}</span>
          </div>
          <div class="summary-item">
            <span class="label">Успешных:</span>
            <span class="value success">{results.summary.success_count}</span>
          </div>
          <div class="summary-item">
            <span class="label">Ошибок:</span>
            <span class="value error">{results.summary.error_count}</span>
          </div>
          <div class="summary-item">
            <span class="label">Всего токенов:</span>
            <span class="value">{results.summary.total_tokens}</span>
          </div>
          <div class="summary-item">
            <span class="label">Общая стоимость:</span>
            <span class="value">{formatCost(results.summary.total_cost)}</span>
          </div>
          <div class="summary-item">
            <span class="label">Среднее время:</span>
            <span class="value">{formatDuration(results.summary.avg_duration_ms)}</span>
          </div>
        </div>
      </div>

      <div class="test-results">
        <h4>Детали тестов</h4>
        {#each results.results as result (result.test_type)}
          <div class="test-result-card" class:success={result.success} class:error={!result.success}>
            <div class="result-header">
              <h5>
                {result.test_type === 'short' ? 'Короткий запрос' :
                 result.test_type === 'long' ? 'Длинный запрос' :
                 'Превышение лимита'}
              </h5>
              <span class="status-badge" class:success={result.success} class:error={!result.success}>
                {result.success ? 'Успех' : 'Ошибка'}
              </span>
            </div>

            <div class="result-stats">
              <div class="stat-item">
                <span class="stat-label">Токены запроса:</span>
                <span class="stat-value">{result.tokens_input}</span>
              </div>
              <div class="stat-item">
                <span class="stat-label">Токены ответа:</span>
                <span class="stat-value">{result.tokens_output}</span>
              </div>
              <div class="stat-item">
                <span class="stat-label">Всего токенов:</span>
                <span class="stat-value">{result.tokens_total} / {result.max_tokens}</span>
              </div>
              <div class="stat-item">
                <span class="stat-label">Стоимость:</span>
                <span class="stat-value">{formatCost(result.cost)}</span>
              </div>
              <div class="stat-item">
                <span class="stat-label">Время:</span>
                <span class="stat-value">{formatDuration(result.duration_ms)}</span>
              </div>
            </div>

            {#if result.error}
              <div class="error-details">
                <strong>Ошибка:</strong> {result.error}
              </div>
            {/if}

            <div class="result-content">
              <details>
                <summary>Запрос ({result.message.length} символов)</summary>
                <pre class="message-content">{result.message}</pre>
              </details>
              {#if result.response}
                <details>
                  <summary>Ответ ({result.response.length} символов)</summary>
                  <pre class="message-content">{result.response}</pre>
                </details>
              {/if}
            </div>
          </div>
        {/each}
      </div>

      <button class="clear-btn" onclick={clearResults}>Очистить результаты</button>
    </div>
  {/if}
</div>

<style>
  .token-test-page {
    padding: 24px;
    max-width: 1200px;
    margin: 0 auto;
  }

  h2 {
    font-size: 24px;
    font-weight: 600;
    color: var(--foreground);
    margin: 0 0 8px 0;
  }

  .description {
    color: var(--muted-foreground);
    margin: 0 0 24px 0;
    line-height: 1.5;
  }

  .controls {
    display: flex;
    gap: 16px;
    margin-bottom: 24px;
    flex-wrap: wrap;
  }

  .control-group {
    display: flex;
    flex-direction: column;
    gap: 8px;
    flex: 1;
    min-width: 200px;
  }

  label {
    font-size: 14px;
    font-weight: 500;
    color: var(--foreground);
  }

  select {
    padding: 8px 12px;
    border: 1px solid var(--border);
    border-radius: 6px;
    background-color: var(--background);
    color: var(--foreground);
    font-size: 14px;
  }

  select:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .test-buttons {
    display: flex;
    gap: 12px;
    margin-bottom: 24px;
    flex-wrap: wrap;
  }

  .test-btn {
    padding: 10px 20px;
    border: 1px solid var(--border);
    border-radius: 6px;
    background-color: var(--background);
    color: var(--foreground);
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s;
  }

  .test-btn:hover:not(:disabled) {
    background-color: var(--muted);
    border-color: var(--border);
  }

  .test-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .test-btn.primary {
    background-color: var(--primary);
    color: var(--primary-foreground);
    border-color: var(--primary);
  }

  .test-btn.primary:hover:not(:disabled) {
    background-color: var(--primary);
    opacity: 0.9;
  }

  .error-message {
    padding: 12px;
    background-color: var(--destructive);
    color: var(--destructive-foreground);
    border-radius: 6px;
    margin-bottom: 24px;
  }

  .loading {
    padding: 24px;
    text-align: center;
    color: var(--muted-foreground);
  }

  .results-section {
    margin-top: 32px;
  }

  h3 {
    font-size: 20px;
    font-weight: 600;
    color: var(--foreground);
    margin: 0 0 16px 0;
  }

  h4 {
    font-size: 16px;
    font-weight: 600;
    color: var(--foreground);
    margin: 0 0 12px 0;
  }

  .summary {
    background-color: var(--muted);
    padding: 16px;
    border-radius: 8px;
    margin-bottom: 24px;
  }

  .summary-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 16px;
  }

  .summary-item {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .summary-item .label {
    font-size: 12px;
    color: var(--muted-foreground);
  }

  .summary-item .value {
    font-size: 18px;
    font-weight: 600;
    color: var(--foreground);
  }

  .summary-item .value.success {
    color: var(--success, #10b981);
  }

  .summary-item .value.error {
    color: var(--destructive);
  }

  .test-results {
    display: flex;
    flex-direction: column;
    gap: 16px;
    margin-bottom: 24px;
  }

  .test-result-card {
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 16px;
    background-color: var(--background);
  }

  .test-result-card.success {
    border-color: var(--success, #10b981);
  }

  .test-result-card.error {
    border-color: var(--destructive);
  }

  .result-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 16px;
  }

  .result-header h5 {
    font-size: 16px;
    font-weight: 600;
    color: var(--foreground);
    margin: 0;
  }

  .status-badge {
    padding: 4px 12px;
    border-radius: 12px;
    font-size: 12px;
    font-weight: 500;
  }

  .status-badge.success {
    background-color: var(--success, #10b981);
    color: white;
  }

  .status-badge.error {
    background-color: var(--destructive);
    color: var(--destructive-foreground);
  }

  .result-stats {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
    gap: 12px;
    margin-bottom: 16px;
  }

  .stat-item {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .stat-label {
    font-size: 12px;
    color: var(--muted-foreground);
  }

  .stat-value {
    font-size: 14px;
    font-weight: 600;
    color: var(--foreground);
  }

  .error-details {
    padding: 12px;
    background-color: var(--destructive);
    color: var(--destructive-foreground);
    border-radius: 6px;
    margin-bottom: 16px;
  }

  .result-content {
    margin-top: 16px;
  }

  details {
    margin-bottom: 12px;
  }

  summary {
    cursor: pointer;
    padding: 8px;
    background-color: var(--muted);
    border-radius: 4px;
    font-weight: 500;
    user-select: none;
  }

  summary:hover {
    background-color: var(--muted);
    opacity: 0.8;
  }

  .message-content {
    padding: 12px;
    background-color: var(--muted);
    border-radius: 6px;
    overflow-x: auto;
    font-size: 12px;
    white-space: pre-wrap;
    word-wrap: break-word;
    max-height: 300px;
    overflow-y: auto;
  }

  .clear-btn {
    padding: 10px 20px;
    border: 1px solid var(--border);
    border-radius: 6px;
    background-color: var(--background);
    color: var(--foreground);
    font-size: 14px;
    cursor: pointer;
    transition: all 0.2s;
  }

  .clear-btn:hover {
    background-color: var(--muted);
  }
</style>
