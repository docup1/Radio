import { computed, reactive } from 'vue'
import ru, { type Messages } from './locales/ru'

export type Locale = 'ru'

const STORAGE_KEY = 'radio:lang'

const messages: Record<Locale, Messages> = { ru }

const state = reactive<{ locale: Locale }>({
  locale: loadInitial(),
})

function loadInitial(): Locale {
  try {
    const saved = localStorage.getItem(STORAGE_KEY)
    if (saved && saved in messages) return saved as Locale
  } catch {
    // ignore storage errors
  }
  return 'ru'
}

export const locale = computed(() => state.locale)

export const t = (key: keyof Messages | string, params?: Record<string, string | number>): string => {
  const dict = messages[state.locale]
  let tmpl: string = (dict as Record<string, string>)[key] ?? String(key)
  if (params) {
    for (const [k, v] of Object.entries(params)) {
      tmpl = tmpl.replaceAll(`{${k}}`, String(v))
    }
  }
  return tmpl
}

export function setLocale(l: Locale) {
  state.locale = l
  try {
    localStorage.setItem(STORAGE_KEY, l)
  } catch {
    // ignore storage errors
  }
}

export function useI18n() {
  return { t, locale, setLocale }
}