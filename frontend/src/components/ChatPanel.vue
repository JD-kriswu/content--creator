<template>
  <div class="chat-panel">
    <div ref="messagesEl" class="messages">
      <!-- Welcome screen -->
      <div v-if="!chatStore.messages.length && !scriptDetail" class="welcome">
        <h2>🎙 口播稿助手</h2>
        <p>粘贴短视频链接，或直接粘贴口播文案<br>AI 5角色分析，生成原创改写稿</p>
        <div class="hint-box">
          <strong>使用方式：</strong><br>
          1. 粘贴抖音/小红书/B站链接，或直接粘贴文案<br>
          2. AI 自动完成5角色分析 + 辩论决策<br>
          3. 确认大纲后，AI 流式生成终稿<br>
          4. 相似度检测通过后自动保存
        </div>
      </div>

      <!-- Script detail view -->
      <div v-if="scriptDetail" class="script-detail">
        <h3>{{ scriptDetail.title }}</h3>
        <pre>{{ scriptDetail.content }}</pre>
      </div>

      <!-- Message list -->
      <template v-for="msg in chatStore.messages" :key="msg.id">
        <!-- Outline card -->
        <div v-if="msg.outlineData" class="msg-row assistant">
          <div class="msg-avatar ai">📋</div>
          <div class="outline-card">
            <h4>📋 大纲方案（确认后开始撰写）</h4>
            <div v-if="(msg.outlineData as OutlineData).elements?.length" class="elements">
              <strong>保留要素：</strong>
              <el-tag
                v-for="e in (msg.outlineData as OutlineData).elements"
                :key="String(e)"
                size="small"
                style="margin:2px"
              >{{ e }}</el-tag>
            </div>
            <div
              v-for="p in (msg.outlineData as OutlineData).outline"
              :key="String(p.part)"
              class="outline-row"
            >
              <el-tag type="info" size="small">{{ p.part }}</el-tag>
              <span class="part-content">{{ p.content }} <span class="duration">{{ p.duration }}</span></span>
            </div>
          </div>
        </div>

        <!-- Action buttons -->
        <div v-else-if="msg.actionOptions" class="msg-row assistant">
          <div class="msg-avatar ai">💬</div>
          <div class="action-btns">
            <el-button
              v-for="(opt, i) in (msg.actionOptions as string[])"
              :key="i"
              :type="i === 0 ? 'primary' : 'default'"
              size="small"
              @click="quickSend(i + 1)"
            >{{ opt }}</el-button>
          </div>
        </div>

        <!-- Similarity card -->
        <div v-else-if="msg.simData" class="msg-row assistant">
          <div class="msg-avatar ai">📊</div>
          <div class="sim-card">
            <div v-for="(val, key) in simDisplay(msg.simData as SimilarityData)" :key="key" class="sim-item">
              <div class="label">{{ val.label }}</div>
              <div :class="['value', val.cls]">{{ val.text }}</div>
            </div>
          </div>
        </div>

        <!-- Regular message -->
        <div v-else class="msg-row" :class="msg.role">
          <div class="msg-avatar" :class="msg.role === 'user' ? 'user-av' : 'ai'">
            {{ msg.role === 'user' ? '👤' : '🤖' }}
          </div>
          <!-- eslint-disable-next-line vue/no-v-html -->
          <div class="msg-bubble" :class="{ streaming: msg.streaming }" v-html="msg.html" />
        </div>
      </template>
    </div>

    <!-- Input area -->
    <div class="input-area">
      <el-input
        v-model="inputText"
        type="textarea"
        :autosize="{ minRows: 1, maxRows: 6 }"
        placeholder="粘贴视频链接 或 直接粘贴口播文案... (Enter 发送，Shift+Enter 换行)"
        :disabled="chatStore.sending"
        @keydown.enter.exact.prevent="doSend"
      />
      <el-button
        type="primary"
        :loading="chatStore.sending"
        :disabled="!inputText.trim()"
        @click="doSend"
      >▶</el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, nextTick } from 'vue'
import { useChatStore, type SimilarityData, type OutlineData } from '@/stores/chat'

const emit = defineEmits<{ scriptSaved: [] }>()

const chatStore = useChatStore()
const inputText = ref('')
const messagesEl = ref<HTMLElement | null>(null)
const scriptDetail = ref<{ title: string; content: string } | null>(null)

watch(() => chatStore.messages.length, () => {
  nextTick(() => {
    if (messagesEl.value) messagesEl.value.scrollTop = messagesEl.value.scrollHeight
  })
})

// Also watch for streaming updates (token events update existing message, length doesn't change)
watch(() => chatStore.messages.map(m => m.html).join(''), () => {
  nextTick(() => {
    if (messagesEl.value) messagesEl.value.scrollTop = messagesEl.value.scrollHeight
  })
})

// Emit scriptSaved when complete event arrives
watch(() => chatStore.messages, (msgs) => {
  const last = msgs[msgs.length - 1]
  if (last?.html?.includes('稿件已保存')) {
    emit('scriptSaved')
  }
}, { deep: true })

function doSend() {
  const text = inputText.value.trim()
  if (!text || chatStore.sending) return
  inputText.value = ''
  scriptDetail.value = null
  chatStore.send(text)
}

function quickSend(num: number) {
  inputText.value = String(num)
  doSend()
}

function showWelcome() {
  chatStore.messages.length = 0
  scriptDetail.value = null
}

function showScriptDetail(title: string, content: string) {
  scriptDetail.value = { title, content }
  chatStore.messages.length = 0
}

defineExpose({ showWelcome, showScriptDetail })

function simDisplay(data: SimilarityData) {
  const total = data.total ?? 0
  const cls = total < 25 ? 'ok' : total < 30 ? 'warn' : 'bad'
  return {
    total: { label: '综合相似度', text: `${total.toFixed(1)}%`, cls },
    vocab: { label: '词汇', text: `${(data.vocab ?? 0).toFixed(1)}%`, cls: '' },
    sentence: { label: '句式', text: `${(data.sentence ?? 0).toFixed(1)}%`, cls: '' },
    structure: { label: '结构', text: `${(data.structure ?? 0).toFixed(1)}%`, cls: '' },
    viewpoint: { label: '观点', text: `${(data.viewpoint ?? 0).toFixed(1)}%`, cls: '' },
    result: { label: '结论', text: total < 30 ? '✅ 通过' : '❌ 超标', cls }
  }
}
</script>

<style scoped>
.chat-panel { display: flex; flex-direction: column; height: 100%; background: #0f1117; }
.messages { flex: 1; overflow-y: auto; padding: 16px; display: flex; flex-direction: column; gap: 8px; }
.welcome { text-align: center; margin: auto; color: #64748b; max-width: 480px; }
.welcome h2 { font-size: 24px; color: #a78bfa; margin-bottom: 12px; }
.welcome p { margin-bottom: 16px; line-height: 1.6; }
.hint-box { background: #1a1d27; border: 1px solid #2a2d3e; border-radius: 10px; padding: 14px 16px; font-size: 13px; line-height: 1.8; text-align: left; }
.msg-row { display: flex; gap: 8px; align-items: flex-start; }
.msg-row.user { flex-direction: row-reverse; }
.msg-avatar { width: 32px; height: 32px; border-radius: 50%; display: flex; align-items: center; justify-content: center; font-size: 16px; flex-shrink: 0; background: #2a2d3e; }
.user-av { background: #7c3aed; }
.ai { background: #1e293b; }
.msg-bubble { background: #1a1d27; border: 1px solid #2a2d3e; border-radius: 12px; padding: 10px 14px; max-width: 720px; font-size: 14px; line-height: 1.6; color: #e2e8f0; word-break: break-word; }
.msg-row.user .msg-bubble { background: #4c1d95; border-color: #6d28d9; }
.streaming::after { content: '▊'; animation: blink .7s step-end infinite; color: #a78bfa; }
@keyframes blink { 50% { opacity: 0; } }
.outline-card { background: #1a1d27; border: 1px solid #2a2d3e; border-radius: 12px; padding: 16px; max-width: 680px; }
.outline-card h4 { color: #a78bfa; margin: 0 0 12px; font-size: 14px; }
.elements { margin-bottom: 10px; font-size: 12px; color: #94a3b8; }
.outline-row { display: flex; gap: 8px; align-items: flex-start; margin: 6px 0; font-size: 13px; }
.part-content { color: #e2e8f0; flex: 1; }
.duration { color: #64748b; font-size: 12px; }
.action-btns { display: flex; flex-wrap: wrap; gap: 8px; padding: 4px 0; }
.sim-card { display: flex; flex-wrap: wrap; gap: 12px; background: #1a1d27; border: 1px solid #2a2d3e; border-radius: 12px; padding: 14px; }
.sim-item { text-align: center; min-width: 60px; }
.label { font-size: 11px; color: #64748b; margin-bottom: 4px; }
.value { font-size: 16px; font-weight: 700; color: #e2e8f0; }
.value.ok { color: #34d399; }
.value.warn { color: #fbbf24; }
.value.bad { color: #f87171; }
.script-detail { background: #1a1d27; border: 1px solid #2a2d3e; border-radius: 12px; padding: 20px; }
.script-detail h3 { color: #a78bfa; margin: 0 0 12px; font-size: 16px; }
.script-detail pre { white-space: pre-wrap; color: #e2e8f0; font-size: 14px; line-height: 1.6; margin: 0; }
.input-area { display: flex; gap: 8px; padding: 12px 16px; border-top: 1px solid #2a2d3e; background: #1a1d27; flex-shrink: 0; }
.input-area :deep(.el-textarea__inner) { background: #0f1117; color: #e2e8f0; border-color: #2a2d3e; }
:deep(.step-badge) { background: #1e2133; border-radius: 8px; padding: 6px 12px; font-size: 13px; color: #94a3b8; display: inline-block; }
:deep(.info-badge) { font-size: 12px; color: #64748b; padding: 2px 8px; display: inline-block; }
:deep(.err-text) { color: #f87171; }
:deep(.ok-text) { color: #34d399; }
:deep(.hint-text) { color: #64748b; font-size: 13px; }
:deep(h3) { color: #a78bfa; margin: 8px 0 4px; }
:deep(strong) { color: #c4b5fd; }
:deep(code) { background: #1e2133; padding: 1px 5px; border-radius: 4px; font-size: 13px; }
:deep(table) { border-collapse: collapse; width: 100%; margin: 8px 0; font-size: 13px; }
:deep(td), :deep(th) { border: 1px solid #2a2d3e; padding: 6px 10px; }
:deep(th) { background: #1e2133; color: #94a3b8; }
:deep(ul) { padding-left: 20px; margin: 4px 0; }
:deep(li) { margin: 2px 0; }
</style>
