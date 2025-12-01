<script lang="ts">
  export let role: 'user' | 'assistant';
  export let content: string;
</script>

<div class="message" class:user={role === 'user'} class:assistant={role === 'assistant'}>
  <div class="content">
    {content}
  </div>
</div>

<style>
  .message {
    padding: 16px 24px;
    max-width: 100%;
    word-wrap: break-word;
  }

  .message.user {
    background-color: var(--bg-secondary);
    text-align: right;
  }

  .message.assistant {
    background-color: var(--bg-primary);
  }

  .content {
    display: inline-block;
    max-width: 80%;
    text-align: left;
    line-height: 1.6;
    white-space: pre-wrap;
  }

  .message.user .content {
    text-align: right;
  }
</style>

