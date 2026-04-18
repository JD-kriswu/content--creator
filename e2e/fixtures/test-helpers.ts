import { Page, BrowserContext } from '@playwright/test'

/**
 * E2E 测试辅助函数
 */

// 测试用户凭证
export const testUser = {
  email: 'test_e2e_new@example.com',
  password: 'Test1234!',
  username: 'test_e2e_new',
}

// 测试管理员用户（用于创建测试数据）
export const adminUser = {
  email: process.env.ADMIN_EMAIL || 'admin@example.com',
  password: process.env.ADMIN_PASSWORD || 'admin123',
}

/**
 * 登录辅助函数
 * 使用 API 直接设置 token，避免 UI 登录流程
 */
export async function loginViaAPI(page: Page, email: string, password: string) {
  // 先导航到页面确保有有效的 origin
  await page.goto('')

  // 使用完整 URL（因为 page.request 不使用 baseURL）
  const baseURL = process.env.E2E_BASE_URL || 'http://localhost/creator'
  const response = await page.request.post(`${baseURL}/api/auth/login`, {
    data: { email, password },
  })

  if (!response.ok()) {
    throw new Error(`Login failed: ${response.status()}`)
  }

  const data = await response.json()
  // 设置 localStorage（使用对象包装多个参数）
  await page.evaluate(({ token, user }) => {
    localStorage.setItem('token', token)
    localStorage.setItem('user', JSON.stringify(user))
  }, { token: data.token, user: data.user })

  // 刷新页面使 AuthContext 读取 localStorage
  await page.reload()
}

/**
 * 通过 UI 登录（用于测试登录流程本身）
 */
export async function loginViaUI(page: Page, email: string, password: string) {
  await page.goto('/auth')
  await page.getByRole('tab', { name: '登录' }).click()
  await page.getByLabel('邮箱').fill(email)
  await page.getByLabel('密码').fill(password)
  await page.getByRole('button', { name: '登录' }).click()
}

/**
 * 等待 SSE 流完成
 * 检测页面进入 complete 状态（script-editor 出现）
 */
export async function waitForSSEComplete(page: Page, timeout = 30000) {
  // 等待终稿编辑器出现（complete 状态标志）
  await page.locator('[data-testid="script-editor"]').waitFor({
    state: 'visible',
    timeout,
  })
}

/**
 * 等待大纲出现（awaiting 状态）
 */
export async function waitForOutline(page: Page, timeout = 20000) {
  await page.locator('[data-testid="outline-editor"]').waitFor({
    state: 'visible',
    timeout,
  })
}

/**
 * 发送带 mock 参数的 SSE 请求
 */
export async function sendMockSSEMessage(page: Page, message: string) {
  const baseURL = process.env.E2E_BASE_URL || 'http://localhost/creator'
  // 通过 API 直接发送 mock 请求
  const response = await page.request.post(`${baseURL}/api/chat/message`, {
    data: { message, mock: true },
    headers: {
      Authorization: `Bearer ${await getTokenFromPage(page)}`,
    },
  })
  return response
}

/**
 * 从页面获取当前 token
 */
export async function getTokenFromPage(page: Page): Promise<string> {
  return page.evaluate(() => localStorage.getItem('token') || '')
}

/**
 * 重置会话状态
 */
export async function resetSession(page: Page) {
  // 先导航到页面确保有有效的 origin 和 localStorage
  await page.goto('')

  const baseURL = process.env.E2E_BASE_URL || 'http://localhost/creator'
  const token = await getTokenFromPage(page)

  // 如果没有 token，说明未登录，无需重置
  if (!token) return

  try {
    await page.request.post(`${baseURL}/api/chat/reset`, {
      headers: {
        Authorization: `Bearer ${token}`,
      },
      timeout: 5000, // 缩短超时，快速失败
    })
  } catch {
    // 重置失败时忽略，可能后端未运行或会话不存在
    console.log('resetSession failed, continuing anyway')
  }
}

/**
 * 检查是否已登录
 */
export async function isLoggedIn(page: Page): Promise<boolean> {
  const token = await getTokenFromPage(page)
  return token !== ''
}

/**
 * 清理测试数据
 */
export async function cleanupTestData(page: Page) {
  // 先导航到页面确保有有效的 origin
  await page.goto('')
  // 清理 localStorage
  await page.evaluate(() => {
    localStorage.clear()
  })
}

export default {
  testUser,
  loginViaAPI,
  loginViaUI,
  waitForSSEComplete,
  waitForOutline,
  sendMockSSEMessage,
  getTokenFromPage,
  resetSession,
  isLoggedIn,
  cleanupTestData,
}