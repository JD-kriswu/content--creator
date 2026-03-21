<template>
  <div class="app-shell">
    <header class="app-header">
      <div class="logo">🎙 口播稿助手</div>
      <div class="user-info">
        <span>{{ userStore.user?.username }}</span>
        <el-button size="small" text type="info" @click="logout">退出</el-button>
      </div>
    </header>

    <div class="main-content">
      <aside class="sidebar">
        <div class="sidebar-header">
          <el-button size="small" @click="newChat">+ 新建</el-button>
        </div>
        <el-tabs v-model="activeTab" class="sidebar-tabs">
          <el-tab-pane label="会话" name="conversations">
            <ConversationList
              ref="convListRef"
              :active-id="activeConvId"
              @select="loadConversation"
            />
          </el-tab-pane>
          <el-tab-pane label="稿件" name="scripts">
            <ScriptList ref="scriptListRef" @select="viewScript" />
          </el-tab-pane>
        </el-tabs>
      </aside>

      <main class="chat-area">
        <ChatPanel ref="chatPanelRef" @script-saved="onScriptSaved" />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { useChatStore } from '@/stores/chat'
import { ElMessage } from 'element-plus'
import ChatPanel from '@/components/ChatPanel.vue'
import ScriptList from '@/components/ScriptList.vue'
import ConversationList from '@/components/ConversationList.vue'
import { getScript } from '@/api/scripts'
import { getConversation, type Conversation } from '@/api/conversations'

const router = useRouter()
const userStore = useUserStore()
const chatStore = useChatStore()
const chatPanelRef = ref<InstanceType<typeof ChatPanel> | null>(null)
const scriptListRef = ref<InstanceType<typeof ScriptList> | null>(null)
const convListRef = ref<InstanceType<typeof ConversationList> | null>(null)
const activeTab = ref('conversations')
const activeConvId = ref<number | undefined>(undefined)

// Refresh conversation list after every send completes (so new conversations appear immediately)
watch(() => chatStore.messagesUpdated, () => {
  convListRef.value?.reload()
})

function logout() {
  userStore.logout()
  router.push('/login')
}

async function newChat() {
  await chatStore.reset()
  activeConvId.value = undefined
  chatPanelRef.value?.showWelcome()
  await convListRef.value?.reload()
  ElMessage.success('新会话已开始')
}

async function loadConversation(conv: Conversation) {
  // If streaming is in progress for this exact conversation, don't interrupt
  if (chatStore.sending && conv.id === chatStore.currentConvId) {
    activeConvId.value = conv.id
    chatPanelRef.value?.clearScriptDetail()
    return
  }
  activeConvId.value = conv.id
  try {
    const { data } = await getConversation(conv.id)
    chatStore.restoreMessages(data.messages ?? '')
    chatPanelRef.value?.clearScriptDetail()
  } catch {
    ElMessage.error('加载会话失败')
  }
}

async function viewScript(id: number) {
  try {
    const { data } = await getScript(id)
    chatPanelRef.value?.showScriptDetail(data.script.title, data.content)
    activeConvId.value = undefined
  } catch {
    // ignore
  }
}

async function onScriptSaved() {
  await scriptListRef.value?.reload()
  await convListRef.value?.reload()
}
</script>

<style scoped>
.app-shell { display: flex; flex-direction: column; height: 100%; background: #fdf4ff; color: #1e1b4b; }
.app-header {
  display: flex; align-items: center; justify-content: space-between;
  padding: 0 20px; height: 56px; flex-shrink: 0;
  background: linear-gradient(135deg, #c026d3, #7c3aed);
  box-shadow: 0 2px 12px rgba(192, 38, 211, 0.2);
}
.logo { font-size: 16px; font-weight: 700; color: #ffffff; letter-spacing: 0.5px; }
.user-info { display: flex; align-items: center; gap: 12px; font-size: 14px; color: rgba(255,255,255,0.85); }
.user-info :deep(.el-button) { color: rgba(255,255,255,0.75) !important; }
.user-info :deep(.el-button:hover) { color: #ffffff !important; }
.main-content { display: flex; flex: 1; overflow: hidden; }
.sidebar {
  width: 240px; border-right: 1px solid #f0abfc;
  display: flex; flex-direction: column; overflow: hidden;
  background: #faf5ff;
}
.sidebar-header {
  display: flex; align-items: center; justify-content: flex-end;
  padding: 10px 16px; border-bottom: 1px solid #f0abfc;
}
.sidebar-header :deep(.el-button) { color: #c026d3; border-color: #f0abfc; }
.sidebar-tabs { flex: 1; display: flex; flex-direction: column; overflow: hidden; }
.sidebar-tabs :deep(.el-tabs__header) { margin: 0; padding: 0 8px; background: #faf5ff; flex-shrink: 0; }
.sidebar-tabs :deep(.el-tabs__nav-wrap::after) { background: #f0abfc; height: 1px; }
.sidebar-tabs :deep(.el-tabs__item) { font-size: 13px; color: #9ca3af; padding: 0 12px; height: 36px; }
.sidebar-tabs :deep(.el-tabs__item.is-active) { color: #c026d3; font-weight: 600; }
.sidebar-tabs :deep(.el-tabs__active-bar) { background: #c026d3; }
.sidebar-tabs :deep(.el-tabs__content) { flex: 1; overflow: hidden; display: flex; flex-direction: column; }
.sidebar-tabs :deep(.el-tab-pane) { flex: 1; overflow: hidden; display: flex; flex-direction: column; }
.chat-area { flex: 1; display: flex; flex-direction: column; overflow: hidden; }
</style>
