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
          <div class="msg-col">
            <!-- eslint-disable-next-line vue/no-v-html -->
            <div class="msg-bubble" :class="{ streaming: msg.streaming }" v-html="msg.html" />
            <el-button
              v-if="msg.retryable"
              size="small"
              type="danger"
              plain
              class="retry-btn"
              :disabled="chatStore.sending"
              @click="chatStore.retry()"
            >🔄 重试</el-button>
          </div>
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

watch(() => chatStore.messages.map(m => m.html).join(''), () => {
  nextTick(() => {
    if (messagesEl.value) messagesEl.value.scrollTop = messagesEl.value.scrollHeight
  })
})

// Emit scriptSaved reliably when complete event fires (via store counter)
watch(() => chatStore.justCompleted, () => {
  emit('scriptSaved')
})

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

function clearScriptDetail() {
  scriptDetail.value = null
}

defineExpose({ showWelcome, showScriptDetail, clearScriptDetail })

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
.chat-panel { display: flex; flex-direction: column; height: 100%; background: #fdf4ff; }
.messages { flex: 1; overflow-y: auto; padding: 16px; display: flex; flex-direction: column; gap: 8px; }
.welcome { text-align: center; margin: auto; color: #9ca3af; max-width: 480px; }
.welcome h2 { font-size: 24px; background: linear-gradient(135deg, #c026d3, #7c3aed); -webkit-background-clip: text; -webkit-text-fill-color: transparent; background-clip: text; margin-bottom: 12px; }
.welcome p { margin-bottom: 16px; line-height: 1.6; color: #6b7280; }
.hint-box { background: #ffffff; border: 1px solid #f0abfc; border-radius: 12px; padding: 14px 16px; font-size: 13px; line-height: 1.8; text-align: left; color: #4b5563; box-shadow: 0 2px 8px rgba(192,38,211,0.06); }
.msg-row { display: flex; gap: 8px; align-items: flex-start; }
.msg-row.user { flex-direction: row-reverse; }
.msg-avatar { width: 32px; height: 32px; border-radius: 50%; display: flex; align-items: center; justify-content: center; font-size: 16px; flex-shrink: 0; background: #f3e8ff; }
.user-av { background: linear-gradient(135deg, #c026d3, #7c3aed); }
.ai { background: #fce7f3; }
.msg-bubble { background: #ffffff; border: 1px solid #f0abfc; border-radius: 12px; padding: 10px 14px; max-width: 720px; font-size: 14px; line-height: 1.6; color: #1e1b4b; word-break: break-word; box-shadow: 0 1px 4px rgba(192,38,211,0.06); }
.msg-row.user .msg-bubble { background: linear-gradient(135deg, #c026d3, #7c3aed); border-color: transparent; color: #ffffff; box-shadow: 0 2px 8px rgba(192,38,211,0.25); }
.streaming::after { content: '▊'; animation: blink .7s step-end infinite; color: #c026d3; }
@keyframes blink { 50% { opacity: 0; } }
.outline-card { background: #ffffff; border: 1px solid #f0abfc; border-radius: 12px; padding: 16px; max-width: 680px; box-shadow: 0 2px 8px rgba(192,38,211,0.08); }
.outline-card h4 { color: #c026d3; margin: 0 0 12px; font-size: 14px; }
.elements { margin-bottom: 10px; font-size: 12px; color: #9ca3af; }
.outline-row { display: flex; gap: 8px; align-items: flex-start; margin: 6px 0; font-size: 13px; }
.part-content { color: #374151; flex: 1; }
.duration { color: #9ca3af; font-size: 12px; }
.action-btns { display: flex; flex-wrap: wrap; gap: 8px; padding: 4px 0; }
.sim-card { display: flex; flex-wrap: wrap; gap: 12px; background: #ffffff; border: 1px solid #f0abfc; border-radius: 12px; padding: 14px; box-shadow: 0 2px 8px rgba(192,38,211,0.08); }
.sim-item { text-align: center; min-width: 60px; }
.label { font-size: 11px; color: #9ca3af; margin-bottom: 4px; }
.value { font-size: 16px; font-weight: 700; color: #1e1b4b; }
.value.ok { color: #059669; }
.value.warn { color: #d97706; }
.value.bad { color: #e11d48; }
.msg-col { display: flex; flex-direction: column; gap: 6px; max-width: 720px; }
.retry-btn { align-self: flex-start; }
.msg-row.user .msg-col { align-items: flex-end; }
.script-detail h3 { color: #c026d3; margin: 0 0 12px; font-size: 16px; }
.script-detail pre { white-space: pre-wrap; color: #374151; font-size: 14px; line-height: 1.6; margin: 0; }
.input-area { display: flex; gap: 8px; padding: 12px 16px; border-top: 1px solid #f0abfc; background: #ffffff; flex-shrink: 0; box-shadow: 0 -2px 8px rgba(192,38,211,0.06); }
.input-area :deep(.el-textarea__inner) { background: #fdf4ff; color: #1e1b4b; border-color: #f0abfc; border-radius: 10px; }
.input-area :deep(.el-textarea__inner:focus) { border-color: #c026d3; box-shadow: 0 0 0 2px rgba(192,38,211,0.1); }
.input-area :deep(.el-button--primary) { background: linear-gradient(135deg, #c026d3, #7c3aed); border: none; border-radius: 10px; padding: 0 16px; }
:deep(.el-button--primary) { background: linear-gradient(135deg, #c026d3, #7c3aed); border: none; }
:deep(.el-tag) { background: #fce7f3; border-color: #f9a8d4; color: #9d174d; }
:deep(.el-tag--info) { background: #f3e8ff; border-color: #d8b4fe; color: #6d28d9; }
:deep(.step-badge) { background: #fce7f3; border-radius: 8px; padding: 6px 12px; font-size: 13px; color: #9d174d; display: inline-block; border: 1px solid #f9a8d4; }
:deep(.info-badge) { font-size: 12px; color: #9ca3af; padding: 2px 8px; display: inline-block; }
:deep(.err-text) { color: #e11d48; }
:deep(.ok-text) { color: #059669; }
:deep(.hint-text) { color: #9ca3af; font-size: 13px; }
:deep(h3) { color: #c026d3; margin: 8px 0 4px; }
:deep(strong) { color: #7c3aed; }
:deep(code) { background: #fce7f3; padding: 1px 5px; border-radius: 4px; font-size: 13px; color: #9d174d; }
:deep(table) { border-collapse: collapse; width: 100%; margin: 8px 0; font-size: 13px; }
:deep(td), :deep(th) { border: 1px solid #f0abfc; padding: 6px 10px; }
:deep(th) { background: #fce7f3; color: #7c3aed; }
:deep(ul) { padding-left: 20px; margin: 4px 0; }
:deep(li) { margin: 2px 0; color: #374151; }
</style>
