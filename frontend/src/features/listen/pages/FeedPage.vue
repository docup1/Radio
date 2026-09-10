<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useFeed } from '../composables/useFeed'
import StreamCard from '@/features/streams/components/StreamCard.vue'
import { t } from '@/shared/i18n'

const router = useRouter()
const { streams, loading, hasMore, load, setQuery } = useFeed()
const localQuery = ref('')
let debounce: ReturnType<typeof setTimeout> | null = null

onMounted(() => load(true))

function onPlay(id: string) {
  router.push({ name: 'listen', params: { id } })
}

function onSearchInput(e: Event) {
  const v = (e.target as HTMLInputElement).value
  localQuery.value = v
  if (debounce) clearTimeout(debounce)
  debounce = setTimeout(() => setQuery(v), 350)
}

function loadMore() {
  if (!loading.value && hasMore.value) load()
}
</script>

<template>
  <div class="feed-page">
    <section class="hero">
      <h1 class="hero__title">{{ t('feed.title') }}</h1>
      <p class="hero__sub">{{ t('feed.subtitle') }}</p>
      <div class="search">
        <span class="search__icon">⌕</span>
        <input
          :value="localQuery"
          class="search__input"
          :placeholder="t('feed.search')"
          @input="onSearchInput"
        />
      </div>
    </section>

    <div v-if="loading && streams.length === 0" class="state">{{ t('feed.loading') }}</div>

    <div v-else-if="streams.length === 0" class="state">{{ t('feed.empty') }}</div>

    <div v-else class="grid">
      <StreamCard
        v-for="s in streams"
        :key="s.id"
        :stream="s"
        :show-play="true"
        @play="onPlay"
      />
    </div>

    <div v-if="hasMore && streams.length > 0" class="load-more">
      <button class="load-more__btn" :disabled="loading" @click="loadMore">
        {{ loading ? t('feed.loading') : t('common.loading') }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.feed-page {
  min-height: 100vh;
  padding: 104px 24px 96px;
  max-width: 1180px;
  margin: 0 auto;
}
.hero {
  text-align: center;
  padding: 12px 0 40px;
}
.hero__title {
  font-size: 44px;
  font-weight: 800;
  color: #fff;
  margin: 0 0 8px;
  letter-spacing: -0.02em;
}
.hero__sub {
  color: var(--muted);
  font-size: 16px;
  margin: 0 0 28px;
}
.search {
  max-width: 520px;
  margin: 0 auto;
  display: flex;
  align-items: center;
  gap: 10px;
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.14);
  border-radius: 999px;
  padding: 10px 18px;
  backdrop-filter: blur(8px);
}
.search__icon {
  color: var(--muted);
  font-size: 20px;
}
.search__input {
  flex: 1;
  background: transparent;
  border: none;
  outline: none;
  color: #fff;
  font-size: 15px;
}
.search__input::placeholder {
  color: var(--muted);
}
.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 20px;
}
.state {
  color: var(--muted);
  text-align: center;
  padding: 64px 0;
}
.load-more {
  display: flex;
  justify-content: center;
  padding: 28px 0 0;
}
.load-more__btn {
  background: transparent;
  border: 1px solid #333;
  color: var(--muted);
  padding: 10px 26px;
  border-radius: 999px;
  cursor: pointer;
  font-size: 14px;
}
.load-more__btn:hover {
  border-color: var(--primary);
  color: var(--text);
}
</style>