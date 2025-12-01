<script lang="ts">
  import ChatMessage from './lib/ChatMessage.svelte';
  import ChatInput from './lib/ChatInput.svelte';
  import { sendMessage } from './lib/api';
  import type { ChatMessage as ChatMessageType } from './lib/api';

  let messages: ChatMessageType[] = [];
  let loading = false;
  let error: string | null = null;
  let currentAssistantMessage = '';

  async function handleSend(event: CustomEvent<string>) {
    const userMessage = event.detail;
    messages = [...messages, { role: 'user', content: userMessage }];
    loading = true;
    error = null;
    currentAssistantMessage = '';

    try {
      for await (const chunk of sendMessage(userMessage)) {
        currentAssistantMessage += chunk;
        // Обновляем последнее сообщение ассистента или создаем новое
        const lastMessage = messages[messages.length - 1];
        if (lastMessage && lastMessage.role === 'assistant') {
          messages[messages.length - 1] = {
            role: 'assistant',
            content: currentAssistantMessage,
          };
        } else {
          messages = [...messages, { role: 'assistant', content: currentAssistantMessage }];
        }
        messages = [...messages]; // Триггер реактивности
      }

      // Финализируем сообщение
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
  </div>

  <div class="messages-container">
    {#if messages.length === 0}
      <div class="welcome">
        <p>Добро пожаловать! Задайте вопрос, и я постараюсь помочь.</p>
      </div>
    {/if}

    {#each messages as message}
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

  <ChatInput on:send={handleSend} {loading} disabled={loading} />
</div>

<style>
  .app {
    display: flex;
    flex-direction: column;
    height: 100vh;
    max-width: 1200px;
    margin: 0 auto;
    background-color: var(--bg-primary);
  }

  .header {
    padding: 16px 24px;
    border-bottom: 1px solid var(--border-color);
    background-color: var(--bg-primary);
  }

  .header h1 {
    font-size: 20px;
    font-weight: 400;
    color: var(--text-primary);
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
    color: var(--text-secondary);
    font-size: 16px;
  }

  .error {
    padding: 16px 24px;
    background-color: #fce8e6;
    color: var(--error-color);
    margin: 8px 24px;
    border-radius: 8px;
  }

  .error p {
    margin: 0;
  }
</style>

