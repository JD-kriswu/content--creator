<template>
  <div class="login-page">
    <el-card class="login-card">
      <h1>🎙 口播稿助手</h1>
      <p class="subtitle">AI 驱动的爆款口播稿改写工具</p>

      <el-tabs v-model="activeTab">
        <el-tab-pane label="登录" name="login">
          <el-form :model="loginForm" @submit.prevent="doLogin">
            <el-form-item>
              <el-input v-model="loginForm.email" type="email" placeholder="邮箱" @keyup.enter="doLogin" />
            </el-form-item>
            <el-form-item>
              <el-input v-model="loginForm.password" type="password" placeholder="密码" show-password @keyup.enter="doLogin" />
            </el-form-item>
            <el-button type="primary" :loading="loading" style="width:100%" @click="doLogin">登录</el-button>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="注册" name="register">
          <el-form :model="regForm" @submit.prevent="doRegister">
            <el-form-item>
              <el-input v-model="regForm.username" placeholder="用户名（2-64字符）" />
            </el-form-item>
            <el-form-item>
              <el-input v-model="regForm.email" type="email" placeholder="邮箱" />
            </el-form-item>
            <el-form-item>
              <el-input v-model="regForm.password" type="password" placeholder="密码（至少6位）" show-password />
            </el-form-item>
            <el-button type="primary" :loading="loading" style="width:100%" @click="doRegister">注册</el-button>
          </el-form>
        </el-tab-pane>
      </el-tabs>

      <p v-if="errorMsg" class="error-msg">{{ errorMsg }}</p>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { ElMessage } from 'element-plus'

const router = useRouter()
const userStore = useUserStore()

const activeTab = ref('login')
const loading = ref(false)
const errorMsg = ref('')

const loginForm = ref({ email: '', password: '' })
const regForm = ref({ username: '', email: '', password: '' })

async function doLogin() {
  errorMsg.value = ''
  loading.value = true
  try {
    await userStore.login(loginForm.value.email, loginForm.value.password)
    router.push('/')
  } catch (e: unknown) {
    errorMsg.value = extractError(e) || '登录失败'
  } finally {
    loading.value = false
  }
}

async function doRegister() {
  errorMsg.value = ''
  loading.value = true
  try {
    await userStore.register(regForm.value.username, regForm.value.email, regForm.value.password)
    ElMessage.success('注册成功，请登录')
    activeTab.value = 'login'
    loginForm.value.email = regForm.value.email
  } catch (e: unknown) {
    errorMsg.value = extractError(e) || '注册失败'
  } finally {
    loading.value = false
  }
}

function extractError(e: unknown): string {
  if (e && typeof e === 'object' && 'response' in e) {
    const resp = (e as { response?: { data?: { error?: string } } }).response
    return resp?.data?.error ?? ''
  }
  return e instanceof Error ? e.message : ''
}
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #0f1117;
}
.login-card {
  width: 380px;
  --el-card-bg-color: #1a1d27;
  --el-border-color: #2a2d3e;
  border-radius: 16px;
}
h1 { text-align: center; color: #e2e8f0; font-size: 22px; margin: 0 0 8px; }
.subtitle { text-align: center; color: #64748b; font-size: 14px; margin-bottom: 24px; }
.error-msg { color: #f87171; font-size: 13px; text-align: center; margin-top: 12px; }
</style>
