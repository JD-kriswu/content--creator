<template>
  <div class="script-list">
    <div v-if="loading" class="hint">加载中...</div>
    <div v-else-if="!scripts.length" class="hint">暂无历史稿件</div>
    <div
      v-for="s in scripts"
      :key="s.id"
      class="script-item"
      @click="$emit('select', s.id)"
    >
      <div class="title">{{ s.title || '未命名' }}</div>
      <div class="meta">{{ formatDate(s.created_at) }} · 相似度 {{ ((s.similarity_score || 0) * 100).toFixed(0) }}%</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getScripts, type Script } from '@/api/scripts'

defineEmits<{ select: [id: number] }>()

const scripts = ref<Script[]>([])
const loading = ref(true)

async function reload() {
  loading.value = true
  try {
    const { data } = await getScripts()
    scripts.value = data.scripts ?? []
  } finally {
    loading.value = false
  }
}

onMounted(reload)
defineExpose({ reload })

function formatDate(dateStr: string): string {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  const diff = Date.now() - d.getTime()
  if (diff < 60000) return '刚刚'
  if (diff < 3600000) return Math.floor(diff / 60000) + '分钟前'
  if (diff < 86400000) return Math.floor(diff / 3600000) + '小时前'
  return `${d.getMonth() + 1}/${d.getDate()}`
}
</script>

<style scoped>
.script-list { flex: 1; overflow-y: auto; }
.hint { padding: 16px; color: #9ca3af; font-size: 13px; text-align: center; }
.script-item { padding: 12px 16px; cursor: pointer; border-bottom: 1px solid #f3e8ff; transition: background 0.15s; }
.script-item:hover { background: #fce7f3; }
.title { font-size: 13px; color: #1e1b4b; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.meta { font-size: 11px; color: #a78bfa; margin-top: 4px; }
</style>
