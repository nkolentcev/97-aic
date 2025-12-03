<script lang="ts">
  import type { CollectConfig } from './api';

  interface Props {
    enabled: boolean;
    config: CollectConfig;
  }

  let { enabled = $bindable(), config = $bindable() }: Props = $props();

  let questionsText: string = $state('');

  // Синхронизация текстового поля с массивом вопросов
  $effect(() => {
    if (config.required_questions) {
      questionsText = config.required_questions.join('\n');
    }
  });

  function updateQuestions() {
    config.required_questions = questionsText
      .split('\n')
      .map(q => q.trim())
      .filter(q => q.length > 0);
  }

  // Предустановленные шаблоны
  const templates = {
    tz: {
      role: 'технический аналитик',
      goal: 'техническое задание на разработку приложения',
      required_questions: [
        'Как называется проект/приложение?',
        'Какую проблему должно решать приложение?',
        'Кто целевая аудитория?',
        'Какие основные функции должны быть?',
        'На каких платформах должно работать (веб, мобильное, десктоп)?',
        'Какие технологии предпочтительны?',
        'Есть ли требования по интеграциям?',
        'Какие сроки реализации?',
      ],
      output_format: 'структурированное техническое задание с разделами: Цели, Аудитория, Функциональные требования, Технические требования, Ограничения'
    },
    menu: {
      role: 'шеф-повар ресторана',
      goal: 'меню для мероприятия',
      required_questions: [
        'Какой тип мероприятия (свадьба, корпоратив, день рождения)?',
        'Сколько гостей ожидается?',
        'Есть ли особые диетические требования?',
        'Какой бюджет на человека?',
        'Предпочтения по кухне (русская, итальянская, азиатская)?',
        'Нужны ли алкогольные напитки?',
      ],
      output_format: 'меню с разделами: Закуски, Основные блюда, Десерты, Напитки с указанием количества порций'
    },
    interview: {
      role: 'HR-специалист',
      goal: 'профиль кандидата на вакансию',
      required_questions: [
        'На какую позицию ищете кандидата?',
        'Какой уровень (junior, middle, senior)?',
        'Какие ключевые навыки обязательны?',
        'Какой опыт работы требуется?',
        'Формат работы (офис, удаленка, гибрид)?',
        'Какая вилка зарплаты?',
      ],
      output_format: 'профиль вакансии с требованиями, обязанностями и условиями'
    }
  };

  function applyTemplate(templateName: keyof typeof templates) {
    const template = templates[templateName];
    config = {
      enabled: true,
      ...template
    };
  }
</script>

<div class="collect-config">
  <div class="config-header">
    <label class="checkbox-label">
      <input type="checkbox" bind:checked={enabled} />
      <span>Режим сбора требований</span>
    </label>
  </div>

  {#if enabled}
    <div class="config-body">
      <!-- Шаблоны -->
      <div class="templates">
        <span class="templates-label">Шаблоны:</span>
        <button class="template-btn" onclick={() => applyTemplate('tz')}>ТЗ</button>
        <button class="template-btn" onclick={() => applyTemplate('menu')}>Меню</button>
        <button class="template-btn" onclick={() => applyTemplate('interview')}>Вакансия</button>
      </div>

      <!-- Роль -->
      <div class="field">
        <label for="role">Роль модели:</label>
        <input 
          type="text" 
          id="role"
          bind:value={config.role}
          placeholder="технический аналитик"
        />
      </div>

      <!-- Цель -->
      <div class="field">
        <label for="goal">Цель сбора:</label>
        <input 
          type="text" 
          id="goal"
          bind:value={config.goal}
          placeholder="техническое задание"
        />
      </div>

      <!-- Обязательные вопросы -->
      <div class="field">
        <label for="questions">Обязательные вопросы (по одному на строку):</label>
        <textarea 
          id="questions"
          bind:value={questionsText}
          oninput={updateQuestions}
          placeholder="Как называется проект?&#10;Какие функции нужны?&#10;На каких платформах?"
          rows="6"
        ></textarea>
      </div>

      <!-- Формат вывода -->
      <div class="field">
        <label for="output">Формат результата:</label>
        <input 
          type="text" 
          id="output"
          bind:value={config.output_format}
          placeholder="структурированный документ"
        />
      </div>
    </div>
  {/if}
</div>

<style>
  .collect-config {
    padding: 16px;
    background-color: var(--card);
    border-radius: 8px;
  }

  .config-header {
    margin-bottom: 12px;
  }

  .checkbox-label {
    display: flex;
    align-items: center;
    gap: 8px;
    cursor: pointer;
    font-weight: 500;
    color: var(--foreground);
  }

  .checkbox-label input {
    width: 16px;
    height: 16px;
    cursor: pointer;
  }

  .config-body {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .templates {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
  }

  .templates-label {
    color: var(--muted-foreground);
    font-size: 13px;
  }

  .template-btn {
    padding: 4px 12px;
    background-color: var(--secondary);
    color: var(--secondary-foreground);
    border: 1px solid var(--border);
    border-radius: 4px;
    font-size: 12px;
    cursor: pointer;
    transition: background-color 0.2s;
  }

  .template-btn:hover {
    background-color: var(--accent);
  }

  .field {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .field label {
    font-size: 13px;
    color: var(--muted-foreground);
  }

  .field input,
  .field textarea {
    padding: 8px 12px;
    background-color: var(--background);
    border: 1px solid var(--border);
    border-radius: 6px;
    color: var(--foreground);
    font-size: 14px;
    font-family: inherit;
  }

  .field input:focus,
  .field textarea:focus {
    outline: none;
    border-color: var(--ring);
  }

  .field textarea {
    resize: vertical;
    min-height: 100px;
  }
</style>

