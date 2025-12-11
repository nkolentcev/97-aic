<script lang="ts">
  interface RequestLog {
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

  function formatCost(cost?: number): string {
    if (cost === undefined || cost === null) return '-';
    if (cost === 0) return 'Бесплатно';
    return `$${cost.toFixed(6)}`;
  }

  interface Props {
    logs: RequestLog[];
  }

  let { logs }: Props = $props();

  let selectedLog = $state<RequestLog | null>(null);
  let showModal = $state(false);

  function formatDate(dateStr: string): string {
    const date = new Date(dateStr);
    return date.toLocaleString('ru-RU', {
      day: '2-digit',
      month: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    });
  }

  function truncate(str: string, maxLen: number): string {
    if (!str) return '';
    try {
      const parsed = JSON.parse(str);
      const content = parsed.message || parsed.content || str;
      if (content.length <= maxLen) return content;
      return content.slice(0, maxLen) + '...';
    } catch {
      if (str.length <= maxLen) return str;
      return str.slice(0, maxLen) + '...';
    }
  }

  function openDetails(log: RequestLog) {
    selectedLog = log;
    showModal = true;
  }

  function closeModal() {
    showModal = false;
    selectedLog = null;
  }

  function formatJson(str: string): string {
    try {
      const obj = JSON.parse(str);
      return formatObject(obj, 0);
    } catch {
      return str;
    }
  }

  function formatObject(obj: any, indent: number): string {
    const spaces = '  '.repeat(indent);
    const innerSpaces = '  '.repeat(indent + 1);

    if (obj === null) return 'null';
    if (obj === undefined) return 'undefined';

    if (typeof obj === 'string') {
      // Раскрываем строку - показываем реальные переносы
      return obj;
    }

    if (typeof obj === 'number' || typeof obj === 'boolean') {
      return String(obj);
    }

    if (Array.isArray(obj)) {
      if (obj.length === 0) return '[]';
      const items = obj.map(item => innerSpaces + formatValue(item, indent + 1));
      return '[\n' + items.join(',\n') + '\n' + spaces + ']';
    }

    if (typeof obj === 'object') {
      const keys = Object.keys(obj);
      if (keys.length === 0) return '{}';
      const pairs = keys.map(key => {
        const value = formatValue(obj[key], indent + 1);
        return `${innerSpaces}${key}: ${value}`;
      });
      return '{\n' + pairs.join(',\n') + '\n' + spaces + '}';
    }

    return String(obj);
  }

  function formatValue(value: any, indent: number): string {
    if (value === null) return 'null';
    if (typeof value === 'string') {
      // Для коротких строк без переносов - в одну строку
      if (value.length < 50 && !value.includes('\n')) {
        return `"${value}"`;
      }
      // Для длинных строк или с переносами - показываем как есть
      return `"\n${'  '.repeat(indent + 1)}${value.split('\n').join('\n' + '  '.repeat(indent + 1))}\n${'  '.repeat(indent)}"`;
    }
    if (typeof value === 'number' || typeof value === 'boolean') {
      return String(value);
    }
    return formatObject(value, indent);
  }
</script>

<div class="logs-container">
  <h2>Логи запросов</h2>

  {#if logs.length === 0}
    <p class="empty">Нет логов</p>
  {:else}
    <div class="table-wrapper">
      <table>
        <thead>
          <tr>
            <th>Время</th>
            <th>Request</th>
            <th>Response</th>
            <th>Статус</th>
            <th>Время (мс)</th>
            <th>Токены (вход)</th>
            <th>Токены (выход)</th>
            <th>Всего</th>
            <th>Стоимость</th>
          </tr>
        </thead>
        <tbody>
          {#each logs as log}
            <tr onclick={() => openDetails(log)} class="clickable">
              <td class="time">{formatDate(log.created_at)}</td>
              <td class="content">{truncate(log.request_json, 30)}</td>
              <td class="content">{truncate(log.response_json, 30)}</td>
              <td class="status" class:success={log.status_code === 200}>{log.status_code}</td>
              <td class="duration">{log.duration_ms}</td>
              <td class="tokens">{log.tokens_input ?? '-'}</td>
              <td class="tokens">{log.tokens_output ?? '-'}</td>
              <td class="tokens">{log.tokens_total ?? '-'}</td>
              <td class="cost">{formatCost(log.cost)}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>

{#if showModal && selectedLog}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <div class="modal-overlay" onclick={closeModal} role="button" tabindex="0" onkeydown={(e) => e.key === 'Enter' && closeModal()}>
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <div class="modal" onclick={(e) => e.stopPropagation()} role="dialog" aria-modal="true" aria-labelledby="modal-title" tabindex="-1">
      <div class="modal-header">
        <h3 id="modal-title">Детали запроса #{selectedLog.id}</h3>
        <button class="close-btn" onclick={closeModal}>&times;</button>
      </div>
      <div class="modal-body">
        <div class="columns">
          <div class="column">
            <h4>Request</h4>
            <textarea class="json-textarea" readonly>{formatJson(selectedLog.request_json)}</textarea>
          </div>
          <div class="column">
            <h4>Response</h4>
            <textarea class="json-textarea" readonly>{formatJson(selectedLog.response_json)}</textarea>
          </div>
        </div>
        <div class="meta-info">
          <span>Статус: {selectedLog.status_code}</span>
          <span>Время: {selectedLog.duration_ms} мс</span>
          <span>Сессия: {selectedLog.session_id}</span>
          {#if selectedLog.tokens_input !== undefined}
            <span>Токены (вход): {selectedLog.tokens_input}</span>
          {/if}
          {#if selectedLog.tokens_output !== undefined}
            <span>Токены (выход): {selectedLog.tokens_output}</span>
          {/if}
          {#if selectedLog.tokens_total !== undefined}
            <span>Всего токенов: {selectedLog.tokens_total}</span>
          {/if}
          {#if selectedLog.cost !== undefined}
            <span>Стоимость: {formatCost(selectedLog.cost)}</span>
          {/if}
        </div>
      </div>
    </div>
  </div>
{/if}

<style>
  .logs-container {
    height: 100%;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  h2 {
    font-size: 16px;
    font-weight: 500;
    padding: 12px 16px;
    margin: 0;
    border-bottom: 1px solid var(--border);
    background-color: var(--muted);
    color: var(--foreground);
  }

  .empty {
    padding: 24px;
    text-align: center;
    color: var(--muted-foreground);
  }

  .table-wrapper {
    flex: 1;
    overflow-y: auto;
  }

  table {
    width: 100%;
    border-collapse: collapse;
    font-size: 13px;
  }

  th, td {
    padding: 8px 12px;
    text-align: left;
    border-bottom: 1px solid var(--border);
  }

  th {
    background-color: var(--muted);
    font-weight: 500;
    position: sticky;
    top: 0;
    color: var(--foreground);
  }

  .clickable {
    cursor: pointer;
    transition: background-color 0.15s;
  }

  .clickable:hover {
    background-color: var(--muted);
  }

  .time {
    white-space: nowrap;
    font-size: 12px;
    color: var(--muted-foreground);
  }

  .content {
    max-width: 150px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .status {
    font-weight: 500;
  }

  .status.success {
    color: #4caf50;
  }

  .duration {
    text-align: right;
    font-family: monospace;
  }

  .tokens {
    text-align: right;
    font-family: monospace;
    font-size: 12px;
    color: var(--muted-foreground);
  }

  .cost {
    text-align: right;
    font-family: monospace;
    font-size: 12px;
    color: var(--muted-foreground);
  }

  /* Modal */
  .modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background-color: rgba(0, 0, 0, 0.7);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
    backdrop-filter: blur(2px);
  }

  .modal {
    background-color: var(--background);
    border-radius: 8px;
    width: 90%;
    max-width: 1000px;
    height: 70vh;
    display: flex;
    flex-direction: column;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
    border: 1px solid var(--border);
  }

  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px;
    border-bottom: 1px solid var(--border);
  }

  .modal-header h3 {
    margin: 0;
    font-size: 18px;
  }

  .close-btn {
    background: none;
    border: none;
    font-size: 24px;
    cursor: pointer;
    color: var(--muted-foreground);
    padding: 0;
    line-height: 1;
  }

  .close-btn:hover {
    color: var(--foreground);
  }

  .modal-body {
    padding: 16px;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 16px;
    flex: 1;
    min-height: 0;
  }

  .columns {
    display: flex;
    gap: 16px;
    flex: 1;
    min-height: 0;
  }

  .column {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-width: 0;
  }

  .column h4 {
    margin: 0 0 8px 0;
    font-size: 14px;
    color: var(--muted-foreground);
    flex-shrink: 0;
  }

  .json-textarea {
    flex: 1;
    min-height: 200px;
    padding: 12px;
    border: 1px solid var(--border);
    border-radius: 6px;
    background-color: var(--muted);
    color: var(--foreground);
    font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
    font-size: 12px;
    line-height: 1.5;
    resize: none;
    white-space: pre-wrap;
    word-break: break-word;
  }

  .json-textarea:focus {
    outline: none;
    border-color: var(--ring);
  }

  .meta-info {
    display: flex;
    gap: 16px;
    font-size: 12px;
    color: var(--muted-foreground);
    padding-top: 12px;
    border-top: 1px solid var(--border);
    flex-shrink: 0;
  }
</style>
