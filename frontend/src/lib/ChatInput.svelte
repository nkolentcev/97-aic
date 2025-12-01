<script lang="ts">
  import { createEventDispatcher } from 'svelte';

  export let disabled = false;
  export let loading = false;

  const dispatch = createEventDispatcher();
  let input: HTMLTextAreaElement;
  let message = '';

  function handleSubmit() {
    if (message.trim() && !disabled && !loading) {
      dispatch('send', message.trim());
      message = '';
      if (input) {
        input.focus();
      }
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSubmit();
    }
  }

  function adjustHeight() {
    if (input) {
      input.style.height = 'auto';
      input.style.height = input.scrollHeight + 'px';
    }
  }
</script>

<div class="input-container">
  <textarea
    bind:this={input}
    bind:value={message}
    on:keydown={handleKeydown}
    on:input={adjustHeight}
    placeholder="Введите сообщение..."
    disabled={disabled || loading}
    rows="1"
  />
  <button
    on:click={handleSubmit}
    disabled={disabled || loading || !message.trim()}
    class="send-button"
  >
    {#if loading}
      <span class="spinner"></span>
    {:else}
      <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
        <path
          d="M2.01 21L23 12 2.01 3 2 10l15 2-15 2z"
          fill="currentColor"
        />
      </svg>
    {/if}
  </button>
</div>

<style>
  .input-container {
    display: flex;
    align-items: flex-end;
    gap: 8px;
    padding: 16px 24px;
    background-color: var(--bg-primary);
    border-top: 1px solid var(--border-color);
    max-width: 100%;
  }

  textarea {
    flex: 1;
    min-height: 24px;
    max-height: 200px;
    padding: 12px 16px;
    border: 1px solid var(--border-color);
    border-radius: 24px;
    font-size: 16px;
    font-family: inherit;
    resize: none;
    outline: none;
    background-color: var(--bg-secondary);
    color: var(--text-primary);
  }

  textarea:focus {
    border-color: var(--accent-color);
  }

  textarea:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .send-button {
    width: 48px;
    height: 48px;
    border-radius: 50%;
    border: none;
    background-color: var(--accent-color);
    color: white;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: background-color 0.2s;
    flex-shrink: 0;
  }

  .send-button:hover:not(:disabled) {
    background-color: var(--accent-hover);
  }

  .send-button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .spinner {
    width: 20px;
    height: 20px;
    border: 2px solid rgba(255, 255, 255, 0.3);
    border-top-color: white;
    border-radius: 50%;
    animation: spin 0.6s linear infinite;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }
</style>

