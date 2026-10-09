import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { authApi } from '../api'
import type { AuthStatusVO } from '../types'

const TOKEN_KEY = 'ssh_hub_token'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string>(localStorage.getItem(TOKEN_KEY) || '')
  const initialized = ref<boolean>(true)
  const fromEnv = ref<boolean>(false)
  const checked = ref<boolean>(false)

  const isAuthenticated = computed(() => !!token.value)

  async function checkStatus(): Promise<AuthStatusVO> {
    try {
      const res = await authApi.status()
      initialized.value = res.initialized
      fromEnv.value = res.from_env
      checked.value = true
      return res
    } catch (e) {
      console.error('Failed to fetch auth status', e)
      return { initialized: true, from_env: false }
    }
  }

  async function login(secretKey: string): Promise<void> {
    const res = await authApi.login(secretKey)
    token.value = res.token
    localStorage.setItem(TOKEN_KEY, res.token)
  }

  async function initKey(secretKey: string): Promise<void> {
    const res = await authApi.init(secretKey)
    token.value = res.token
    initialized.value = true
    localStorage.setItem(TOKEN_KEY, res.token)
  }

  function logout(): void {
    token.value = ''
    localStorage.removeItem(TOKEN_KEY)
    try {
      authApi.logout()
    } catch {
      // ignore network errors on logout
    }
  }

  return {
    token,
    initialized,
    fromEnv,
    checked,
    isAuthenticated,
    checkStatus,
    login,
    initKey,
    logout,
  }
})
