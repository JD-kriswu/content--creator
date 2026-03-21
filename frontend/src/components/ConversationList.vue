<template>
  <div class="conv-list">
    <div v-if="loading" class="hint">加载中...</div>
    <div v-else-if="!conversations.length" class="hint">暂无历史会话</div>
    <div
      v-for="conv in conversations"
      :key="conv.id"
      class="conv-item"
      :class="{ active: activeConvId === conv.id }"
      @click="$emit('select', conv)"
    >
      <div class="conv-title">{{ conv.title || '未命名会话' }}</div>
      <div class="conv-meta">
        <span>{{ formatDate(conv.created_at) }}</span>
        <span class="badge" :class="conv.state === 1 ? 'done' : 'progress'">
          {{ conv.state === 1 ? '已完成' : '进行中' }}
        </span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { listConversations, type Conversation } from '@/api/conversations'

defineEmits<{ select: [conv: Conversation] }>()

const props = defineProps<{ activeId?: number }>()
const activeConvId = computed(() => props.activeId)

const conversations = ref<Conversation[]>([])
const loading = ref(true)

async function reload() {
  loading.value = true
  try {
    const { data } = await listConversations()
    conversations.value = data.conversations ?? []
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
.conv-list { flex: 1; overflow-y: auto; }
.hint { padding: 16px; color: #9ca3af; font-size: 13px; text-align: center; }
.conv-item {
  padding: 12px 16px; cursor: pointer;
  border-bottom: 1px solid #f3e8ff; transition: background 0.15s;
}
.conv-item:hover { background: #fce7f3; }
.conv-item.active { background: #f0e6ff; border-left: 3px solid #c026d3; }
.conv-title { font-size: 13px; color: #1e1b4b; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.conv-meta { display: flex; align-items: center; justify-content: space-between; margin-top: 4px; font-size: 11px; color: #a78bfa; }
.badge { padding: 1px 6px; border-radius: 10px; font-size: 10px; }
.badge.done { background: #dcfce7; color: #16a34a; }
.badge.progress { background: #fef9c3; color: #ca8a04; }
</style>
