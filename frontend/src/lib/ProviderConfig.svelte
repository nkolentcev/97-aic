<script lang="ts">
  import type { ProviderInfo, ReasoningModeInfo, ReasoningMode } from './api';

  interface Props {
    providers: ProviderInfo[];
    reasoningModes: ReasoningModeInfo[];
    selectedProvider: string;
    selectedModel: string;
    selectedReasoningMode: ReasoningMode;
    systemPrompt: string;
    onProviderChange: (provider: string) => void;
    onModelChange: (model: string) => void;
    onReasoningModeChange: (mode: ReasoningMode) => void;
    onSystemPromptChange: (prompt: string) => void;
  }

  let {
    providers,
    reasoningModes,
    selectedProvider,
    selectedModel,
    selectedReasoningMode,
    systemPrompt,
    onProviderChange,
    onModelChange,
    onReasoningModeChange,
    onSystemPromptChange
  }: Props = $props();

  let showSystemPrompt = $state(false);

  // Получаем текущий провайдер
  let currentProvider = $derived(providers.find(p => p.name === selectedProvider));
  let availableModels = $derived(currentProvider?.models || []);

  function handleProviderChange(e: Event) {
    const target = e.target as HTMLSelectElement;
    onProviderChange(target.value);
    // Сбрасываем модель на первую доступную
    const provider = providers.find(p => p.name === target.value);
    if (provider && provider.models.length > 0) {
      onModelChange(provider.current_model || provider.models[0]);
    }
  }

  function handleModelChange(e: Event) {
    const target = e.target as HTMLSelectElement;
    onModelChange(target.value);
  }

  function handleReasoningModeChange(e: Event) {
    const target = e.target as HTMLSelectElement;
    onReasoningModeChange(target.value as ReasoningMode);
  }

  function handleSystemPromptChange(e: Event) {
    const target = e.target as HTMLTextAreaElement;
    onSystemPromptChange(target.value);
  }
</script>

<div class="provider-config">
  <h3>Настройки AI</h3>

  <!-- Выбор провайдера -->
  <div class="config-row">
    <label for="provider-select">Провайдер:</label>
    <select id="provider-select" value={selectedProvider} onchange={handleProviderChange}>
      {#each providers as provider}
        <option value={provider.name}>
          {provider.name === 'gigachat' ? 'GigaChat (Сбер)' :
           provider.name === 'groq' ? 'Groq (бесплатно)' :
           provider.name === 'ollama' ? 'Ollama (локально)' : provider.name}
          {provider.is_default ? ' ★' : ''}
        </option>
      {/each}
    </select>
  </div>

  <!-- Выбор модели -->
  <div class="config-row">
    <label for="model-select">Модель:</label>
    <select id="model-select" value={selectedModel} onchange={handleModelChange}>
      {#each availableModels as model}
        <option value={model}>{model}</option>
      {/each}
    </select>
  </div>

  <!-- День 4: Режим рассуждения -->
  <div class="config-row">
    <label for="reasoning-select">Режим рассуждения:</label>
    <select id="reasoning-select" value={selectedReasoningMode} onchange={handleReasoningModeChange}>
      {#each reasoningModes as mode}
        <option value={mode.id} title={mode.description}>{mode.name}</option>
      {/each}
    </select>
  </div>

  <!-- Подсказка по режиму -->
  {#if selectedReasoningMode !== 'direct'}
    <div class="reasoning-hint">
      {#if selectedReasoningMode === 'step_by_step'}
        Модель будет разбивать задачу на шаги и объяснять каждый
      {:else if selectedReasoningMode === 'experts'}
        Несколько экспертов дадут свои мнения, затем синтезируется решение
      {/if}
    </div>
  {/if}

  <!-- День 5: System Prompt -->
  <div class="config-row">
    <button
      class="toggle-btn"
      onclick={() => showSystemPrompt = !showSystemPrompt}
    >
      {showSystemPrompt ? '▼' : '▶'} System Prompt
    </button>
  </div>

  {#if showSystemPrompt}
    <div class="system-prompt-section">
      <textarea
        class="system-prompt-input"
        placeholder="Введите system prompt для настройки поведения модели...

Примеры:
- Ты — эксперт по Python. Отвечай кратко и с примерами кода.
- Ты — переводчик. Переводи на английский.
- Ты — учитель математики. Объясняй простым языком."
        value={systemPrompt}
        oninput={handleSystemPromptChange}
        rows="5"
      ></textarea>
      <div class="prompt-hint">
        System prompt определяет роль и поведение модели в диалоге
      </div>
    </div>
  {/if}
</div>

<style>
  .provider-config {
    padding: 0;
  }

  h3 {
    font-size: 14px;
    font-weight: 600;
    color: var(--foreground);
    margin: 0 0 12px 0;
  }

  .config-row {
    margin-bottom: 12px;
  }

  .config-row label {
    display: block;
    font-size: 12px;
    color: var(--muted-foreground);
    margin-bottom: 4px;
  }

  select {
    width: 100%;
    padding: 8px 10px;
    border: 1px solid var(--border);
    border-radius: 6px;
    background-color: var(--background);
    color: var(--foreground);
    font-size: 13px;
    cursor: pointer;
  }

  select:focus {
    outline: none;
    border-color: var(--primary);
  }

  .reasoning-hint {
    font-size: 11px;
    color: var(--muted-foreground);
    padding: 8px;
    background-color: var(--muted);
    border-radius: 4px;
    margin-bottom: 12px;
  }

  .toggle-btn {
    background: none;
    border: none;
    color: var(--foreground);
    font-size: 13px;
    cursor: pointer;
    padding: 4px 0;
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .toggle-btn:hover {
    color: var(--primary);
  }

  .system-prompt-section {
    margin-top: 8px;
  }

  .system-prompt-input {
    width: 100%;
    padding: 10px;
    border: 1px solid var(--border);
    border-radius: 6px;
    background-color: var(--background);
    color: var(--foreground);
    font-size: 13px;
    font-family: inherit;
    resize: vertical;
    min-height: 100px;
  }

  .system-prompt-input:focus {
    outline: none;
    border-color: var(--primary);
  }

  .system-prompt-input::placeholder {
    color: var(--muted-foreground);
    opacity: 0.7;
  }

  .prompt-hint {
    font-size: 11px;
    color: var(--muted-foreground);
    margin-top: 4px;
  }
</style>
