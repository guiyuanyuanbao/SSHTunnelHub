<template>
  <div class="login-wrapper">
    <!-- 背景流光与科技网格 -->
    <div class="cyber-glow-bg">
      <div class="glow-circle cyan"></div>
      <div class="glow-circle purple"></div>
    </div>

    <!-- 登录居中卡片 -->
    <div class="login-box">
      <!-- 品牌标识与 Logo -->
      <div class="brand-header">
        <div class="logo-wrap">
          <img src="/logo.png" alt="SSHTunnelHub" class="hub-logo" />
        </div>
        <h1 class="brand-title">SSHTunnelHub</h1>
        <p class="brand-subtitle">
          <template v-if="!authStore.initialized">
            <span class="highlight-text">首次使用向导</span> · 请设定系统访问秘钥
          </template>
          <template v-else>
            现代化 SSH 隧道管理中心 · 输入秘钥解锁控制台
          </template>
        </p>
      </div>

      <!-- 托管提示 (若为环境变量配置) -->
      <div v-if="authStore.fromEnv" class="env-tip-bar">
        <el-icon class="env-icon"><Key /></el-icon>
        <span>系统访问秘钥已由环境变量 <code>HUB_AUTH_KEY</code> 托管保护</span>
      </div>

      <!-- 表单区域 -->
      <el-card shadow="hover" class="login-card">
        <!-- 模式 1: 日常解锁模式 (已初始化) -->
        <div v-if="authStore.initialized" class="form-container">
          <div class="input-row">
            <label class="input-label">访问秘钥</label>
            <el-input
              v-model="secretKey"
              type="password"
              show-password
              size="large"
              placeholder="请输入访问秘钥..."
              :prefix-icon="Lock"
              autofocus
              @keyup.enter="handleLogin"
            />
          </div>

          <div class="option-row">
            <el-checkbox v-model="rememberMe" label="7 天内免重复输入" />
          </div>

          <el-button
            type="primary"
            size="large"
            class="submit-btn"
            :loading="loading"
            @click="handleLogin"
          >
            解锁并进入控制台
            <el-icon class="el-icon--right"><Right /></el-icon>
          </el-button>
        </div>

        <!-- 模式 2: 首次使用初始化模式 (未初始化) -->
        <div v-else class="form-container">
          <div class="init-notice">
            <el-icon class="notice-icon"><InfoFilled /></el-icon>
            <span>系统尚未设置访问秘钥，请先设定用于保护控制台的全局访问口令（不少于 4 位）。</span>
          </div>

          <div class="input-row">
            <label class="input-label">设置新秘钥</label>
            <el-input
              v-model="newKey"
              type="password"
              show-password
              size="large"
              placeholder="请输入新的访问秘钥 (至少4位)..."
              :prefix-icon="Lock"
              autofocus
            />
          </div>

          <div class="input-row">
            <label class="input-label">确认新秘钥</label>
            <el-input
              v-model="confirmKey"
              type="password"
              show-password
              size="large"
              placeholder="请再次输入新访问秘钥..."
              :prefix-icon="Key"
              @keyup.enter="handleInit"
            />
          </div>

          <el-button
            type="success"
            size="large"
            class="submit-btn"
            :loading="loading"
            @click="handleInit"
          >
            完成设定并登录
            <el-icon class="el-icon--right"><Check /></el-icon>
          </el-button>
        </div>
      </el-card>

      <!-- 底部辅助信息 -->
      <div class="login-footer">
        <span>SSHTunnelHub &copy; 2026 · 高可用 SSH 端口转发调度平台</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { Lock, Key, Right, Check, InfoFilled } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '../store/useAuthStore'
import { useTunnelStore } from '../store/useTunnelStore'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const tunnelStore = useTunnelStore()

const secretKey = ref('')
const newKey = ref('')
const confirmKey = ref('')
const rememberMe = ref(true)
const loading = ref(false)

onMounted(async () => {
  loading.value = true
  try {
    await authStore.checkStatus()
  } finally {
    loading.value = false
  }
})

async function handleLogin() {
  if (!secretKey.value.trim()) {
    ElMessage.warning('请输入访问秘钥')
    return
  }

  loading.value = true
  try {
    await authStore.login(secretKey.value.trim())
    ElMessage.success('解锁成功，正在载入控制台')
    
    // 初始化 WebSocket 推送
    tunnelStore.initWebSocket()

    // 导向目标路径或首页
    const redirect = (route.query.redirect as string) || '/'
    router.replace(redirect)
  } catch (err: any) {
    ElMessage.error(err.message || '秘钥验证失败')
  } finally {
    loading.value = false
  }
}

async function handleInit() {
  const k = newKey.value.trim()
  const ck = confirmKey.value.trim()

  if (!k) {
    ElMessage.warning('请输入新的访问秘钥')
    return
  }
  if (k.length < 4) {
    ElMessage.warning('访问秘钥长度至少为 4 个字符')
    return
  }
  if (k !== ck) {
    ElMessage.warning('两次输入的访问秘钥不一致')
    return
  }

  loading.value = true
  try {
    await authStore.initKey(k)
    ElMessage.success('访问秘钥设定成功！已自动解锁')

    tunnelStore.initWebSocket()

    const redirect = (route.query.redirect as string) || '/'
    router.replace(redirect)
  } catch (err: any) {
    ElMessage.error(err.message || '初始化秘钥失败')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-wrapper {
  position: fixed;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: #0b0f19;
  overflow: hidden;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif;
  z-index: 1000;
}

/* 霓虹背光 */
.cyber-glow-bg {
  position: absolute;
  inset: 0;
  pointer-events: none;
}

.glow-circle {
  position: absolute;
  width: 500px;
  height: 500px;
  border-radius: 50%;
  filter: blur(140px);
  opacity: 0.18;
}

.glow-circle.cyan {
  top: -100px;
  left: -100px;
  background: #06b6d4;
}

.glow-circle.purple {
  bottom: -100px;
  right: -100px;
  background: #8b5cf6;
}

/* 居中面板 */
.login-box {
  position: relative;
  width: 100%;
  max-width: 440px;
  padding: 20px;
  display: flex;
  flex-direction: column;
  align-items: center;
  z-index: 2;
}

/* 品牌头部 */
.brand-header {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-bottom: 24px;
}

.logo-wrap {
  width: 90px;
  height: 90px;
  margin-bottom: 16px;
  border-radius: 20px;
  padding: 4px;
  background: linear-gradient(135deg, rgba(56, 189, 248, 0.4), rgba(168, 85, 247, 0.4));
  box-shadow: 0 0 35px rgba(56, 189, 248, 0.25);
  display: flex;
  align-items: center;
  justify-content: center;
}

.hub-logo {
  width: 100%;
  height: 100%;
  border-radius: 18px;
  object-fit: cover;
}

.brand-title {
  margin: 0;
  font-size: 26px;
  font-weight: 800;
  letter-spacing: -0.5px;
  background: linear-gradient(135deg, #f8fafc 0%, #94a3b8 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

.brand-subtitle {
  margin: 8px 0 0;
  font-size: 13px;
  color: #64748b;
}

.highlight-text {
  color: #38bdf8;
  font-weight: 600;
}

/* 环境变量提示 */
.env-tip-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: #38bdf8;
  background: rgba(14, 165, 233, 0.1);
  border: 1px solid rgba(14, 165, 233, 0.25);
  padding: 8px 14px;
  border-radius: 8px;
  margin-bottom: 16px;
  width: 100%;
  box-sizing: border-box;
}

.env-icon {
  font-size: 15px;
  flex-shrink: 0;
}

.env-tip-bar code {
  background: rgba(0, 0, 0, 0.3);
  padding: 2px 6px;
  border-radius: 4px;
  font-family: monospace;
}

/* 卡片 */
.login-card {
  width: 100%;
  background: rgba(15, 23, 42, 0.75) !important;
  backdrop-filter: blur(20px);
  border: 1px solid rgba(255, 255, 255, 0.1) !important;
  border-radius: 16px !important;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.4) !important;
}

:deep(.login-card .el-card__body) {
  padding: 28px 24px;
}

.form-container {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.init-notice {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  background: rgba(16, 185, 129, 0.1);
  border: 1px solid rgba(16, 185, 129, 0.25);
  color: #34d399;
  font-size: 13px;
  padding: 10px 12px;
  border-radius: 8px;
  line-height: 1.5;
}

.notice-icon {
  font-size: 16px;
  margin-top: 2px;
  flex-shrink: 0;
}

.input-row {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.input-label {
  font-size: 13px;
  font-weight: 600;
  color: #94a3b8;
}

:deep(.el-input__wrapper) {
  background-color: rgba(30, 41, 59, 0.7) !important;
  box-shadow: 0 0 0 1px rgba(255, 255, 255, 0.1) inset !important;
  border-radius: 10px !important;
  padding: 6px 14px !important;
  transition: all 0.2s;
}

:deep(.el-input__wrapper:hover) {
  box-shadow: 0 0 0 1px rgba(56, 189, 248, 0.4) inset !important;
}

:deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 2px #0ea5e9 inset !important;
  background-color: rgba(30, 41, 59, 0.95) !important;
}

:deep(.el-input__inner) {
  color: #f1f5f9 !important;
  font-size: 14px;
}

.option-row {
  display: flex;
  justify-content: flex-start;
}

:deep(.el-checkbox__label) {
  color: #64748b !important;
  font-size: 13px;
}

:deep(.el-checkbox__input.is-checked + .el-checkbox__label) {
  color: #38bdf8 !important;
}

.submit-btn {
  width: 100%;
  height: 44px;
  border-radius: 10px !important;
  font-weight: 600;
  font-size: 15px;
  letter-spacing: 0.3px;
  box-shadow: 0 4px 15px rgba(14, 165, 233, 0.3);
  transition: all 0.2s;
}

.submit-btn:hover {
  transform: translateY(-1px);
  box-shadow: 0 6px 20px rgba(14, 165, 233, 0.45);
}

.login-footer {
  margin-top: 24px;
  font-size: 12px;
  color: #475569;
  text-align: center;
}
</style>
