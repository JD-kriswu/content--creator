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
          <span>历史稿件</span>
          <el-button size="small" @click="newChat">+ 新建</el-button>
        </div>
        <ScriptList ref="scriptListRef" @select="viewScript" />
      </aside>

      <main class="chat-area">
        <ChatPanel ref="chatPanelRef" @script-saved="refreshScripts" />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { useChatStore } from '@/stores/chat'
import ChatPanel from '@/components/ChatPanel.vue'
import ScriptList from '@/components/ScriptList.vue'
import { getScript } from '@/api/scripts'

const router = useRouter()
const userStore = useUserStore()
const chatStore = useChatStore()
const chatPanelRef = ref<InstanceType<typeof ChatPanel> | null>(null)
const scriptListRef = ref<InstanceType<typeof ScriptList> | null>(null)

function logout() {
  userStore.logout()
  router.push('/login')
}

async function newChat() {
  await chatStore.reset()
  chatPanelRef.value?.showWelcome()
}

async function viewScript(id: number) {
  try {
    const { data } = await getScript(id)
    chatPanelRef.value?.showScriptDetail(data.script.title, data.content)
  } catch {
    // ignore
  }
}

function refreshScripts() {
  scriptListRef.value?.reload()
}
</script>

<style scoped>
.app-shell { display: flex; flex-direction: column; height: 100vh; background: #0f1117; color: #e2e8f0; }
.app-header { display: flex; align-items: center; justify-content: space-between; padding: 0 20px; height: 56px; background: #1a1d27; border-bottom: 1px solid #2a2d3e; flex-shrink: 0; }
.logo { font-size: 16px; font-weight: 600; color: #a78bfa; }
.user-info { display: flex; align-items: center; gap: 12px; font-size: 14px; color: #94a3b8; }
.main-content { display: flex; flex: 1; overflow: hidden; }
.sidebar { width: 240px; border-right: 1px solid #2a2d3e; display: flex; flex-direction: column; overflow: hidden; }
.sidebar-header { display: flex; align-items: center; justify-content: space-between; padding: 12px 16px; border-bottom: 1px solid #2a2d3e; font-size: 14px; font-weight: 600; color: #94a3b8; }
.chat-area { flex: 1; display: flex; flex-direction: column; overflow: hidden; }
</style>
