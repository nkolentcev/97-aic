<script lang="ts">
  interface Props {
    enabled?: boolean;
    jsonSchema?: string;
  }

  const defaultSchema = `{
  "session_id": 1,
  "status": 1,
  "title": "Заголовок ответа",
  "result": "Содержимое ответа"
}`;

  let {
    enabled = $bindable(false),
    jsonSchema = $bindable('')
  }: Props = $props();

  // Состояние валидации
  let validationError = $state<string | null>(null);
  let isValid = $state(true);

  // Валидация JSON при изменении текста
  function validateJson(text: string) {
    if (!text.trim()) {
      validationError = null;
      isValid = true;
      return;
    }

    try {
      JSON.parse(text);
      validationError = null;
      isValid = true;
    } catch (e) {
      if (e instanceof SyntaxError) {
        validationError = e.message;
      } else {
        validationError = 'Невалидный JSON';
      }
      isValid = false;
    }
  }

  // Реактивная валидация
  $effect(() => {
    if (enabled) {
      validateJson(jsonSchema);
    }
  });

  function handleToggle(e: Event) {
    const target = e.target as HTMLInputElement;
    enabled = target.checked;
    if (enabled && !jsonSchema.trim()) {
      jsonSchema = defaultSchema;
    }
  }

  function handleInput(e: Event) {
    const target = e.target as HTMLTextAreaElement;
    jsonSchema = target.value;
  }
</script>

<div class="json-config">
  <div class="config-header">
    <label class="toggle-label">
      <input type="checkbox" checked={enabled} onchange={handleToggle} />
      <span>Формат JSON-ответа</span>
    </label>
  </div>

  {#if enabled}
    <div class="schema-input">
      <div class="label-row">
        <label class="schema-label" for="json-schema">Структура ответа:</label>
        {#if jsonSchema.trim()}
          <span class="validation-status" class:valid={isValid} class:invalid={!isValid}>
            {isValid ? '✓ JSON валиден' : '✗ Ошибка'}
          </span>
        {/if}
      </div>
      <textarea
        id="json-schema"
        value={jsonSchema}
        oninput={handleInput}
        placeholder={defaultSchema}
        class="schema-textarea"
        class:error={!isValid}
        rows="8"
      ></textarea>
      {#if validationError}
        <p class="error-message">{validationError}</p>
      {:else}
        <p class="hint">Укажите JSON-структуру для ответа ассистента</p>
      {/if}
    </div>
  {/if}
</div>

<style>
  .json-config {
    display: flex;
    flex-direction: column;
    padding: 16px;
    background-color: var(--background);
  }

  .config-header {
    margin-bottom: 8px;
  }

  .toggle-label {
    display: flex;
    align-items: center;
    gap: 8px;
    cursor: pointer;
    font-size: 14px;
    color: var(--foreground);
  }

  .toggle-label input[type="checkbox"] {
    width: 18px;
    height: 18px;
    cursor: pointer;
  }

  .schema-input {
    margin-top: 8px;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .label-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .schema-label {
    font-size: 12px;
    color: var(--muted-foreground);
    font-weight: 500;
  }

  .validation-status {
    font-size: 11px;
    font-weight: 500;
    padding: 2px 8px;
    border-radius: 4px;
  }

  .validation-status.valid {
    color: #22c55e;
    background-color: rgba(34, 197, 94, 0.1);
  }

  .validation-status.invalid {
    color: #ef4444;
    background-color: rgba(239, 68, 68, 0.1);
  }

  .schema-textarea {
    width: 100%;
    padding: 12px;
    border: 1px solid var(--border);
    border-radius: 6px;
    font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
    font-size: 12px;
    line-height: 1.5;
    background-color: var(--background);
    color: var(--foreground);
    resize: vertical;
    min-height: 120px;
    box-sizing: border-box;
    transition: border-color 0.2s;
  }

  .schema-textarea:focus {
    outline: none;
    border-color: var(--ring);
  }

  .schema-textarea.error {
    border-color: #ef4444;
  }

  .schema-textarea.error:focus {
    border-color: #ef4444;
    box-shadow: 0 0 0 2px rgba(239, 68, 68, 0.2);
  }

  .schema-textarea::placeholder {
    color: var(--muted-foreground);
  }

  .hint {
    font-size: 11px;
    color: var(--muted-foreground);
    margin: 0;
    line-height: 1.4;
  }

  .error-message {
    font-size: 11px;
    color: #ef4444;
    margin: 0;
    line-height: 1.4;
  }
</style>
