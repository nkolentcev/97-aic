import { writable } from 'svelte/store';

export type Theme = 'light' | 'dark' | 'system';

const getSystemTheme = (): 'light' | 'dark' => {
  if (typeof window === 'undefined') return 'light';
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
};

const getStoredTheme = (): Theme => {
  if (typeof window === 'undefined') return 'system';
  const stored = localStorage.getItem('theme') as Theme | null;
  return stored || 'system';
};

const applyTheme = (theme: 'light' | 'dark') => {
  if (typeof window === 'undefined') return;
  const root = document.documentElement;
  if (theme === 'dark') {
    root.classList.add('dark');
  } else {
    root.classList.remove('dark');
  }
};

const getEffectiveTheme = (theme: Theme): 'light' | 'dark' => {
  return theme === 'system' ? getSystemTheme() : theme;
};

function createThemeStore() {
  const { subscribe, set, update } = writable<Theme>(getStoredTheme());

  // Применяем тему при инициализации
  if (typeof window !== 'undefined') {
    const effectiveTheme = getEffectiveTheme(getStoredTheme());
    applyTheme(effectiveTheme);

    // Слушаем изменения системной темы
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    const handleChange = () => {
      const currentTheme = getStoredTheme();
      if (currentTheme === 'system') {
        applyTheme(getSystemTheme());
      }
    };
    mediaQuery.addEventListener('change', handleChange);
  }

  return {
    subscribe,
    set: (theme: Theme) => {
      if (typeof window !== 'undefined') {
        localStorage.setItem('theme', theme);
      }
      set(theme);
      applyTheme(getEffectiveTheme(theme));
    },
    update,
    toggle: () => {
      update((current) => {
        const effective = getEffectiveTheme(current);
        const newTheme: Theme = effective === 'dark' ? 'light' : 'dark';
        if (typeof window !== 'undefined') {
          localStorage.setItem('theme', newTheme);
        }
        applyTheme(newTheme);
        return newTheme;
      });
    },
  };
}

export const theme = createThemeStore();

