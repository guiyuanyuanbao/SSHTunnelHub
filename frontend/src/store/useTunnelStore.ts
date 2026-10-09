import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { HostVO, TunnelVO, DashboardStats, LogEntry } from '../types'
import { hostApi, tunnelApi, logApi } from '../api'

export const useTunnelStore = defineStore('tunnel', () => {
  const hosts = ref<HostVO[]>([])
  const tunnels = ref<TunnelVO[]>([])
  const logs = ref<LogEntry[]>([])
  const stats = ref<DashboardStats>({
    total_hosts: 0,
    total_tunnels: 0,
    running_tunnels: 0,
    total_conns: 0,
    total_bytes_in: 0,
    total_bytes_out: 0,
  })

  const wsConnected = ref(false)
  let ws: WebSocket | null = null
  let reconnectTimer: any = null

  async function fetchHosts() {
    try {
      const res = await hostApi.list()
      hosts.value = res.data || []
    } catch (err) {
      console.error('Failed to fetch hosts:', err)
    }
  }

  async function fetchTunnels() {
    try {
      const res = await tunnelApi.list()
      tunnels.value = res.data || []
    } catch (err) {
      console.error('Failed to fetch tunnels:', err)
    }
  }

  async function fetchStats() {
    try {
      const res = await tunnelApi.getStats()
      stats.value = res.data
    } catch (err) {
      console.error('Failed to fetch stats:', err)
    }
  }

  async function fetchLogs(params?: { level?: string; tag?: string; tunnel_id?: number; keyword?: string; limit?: number }) {
    try {
      const res = await logApi.list(params)
      logs.value = res.data || []
    } catch (err) {
      console.error('Failed to fetch logs:', err)
    }
  }

  function initWebSocket() {
    if (ws) {
      try {
        ws.close()
      } catch (e) {}
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.host
    const token = localStorage.getItem('ssh_hub_token')
    const wsUrl = token ? `${protocol}//${host}/ws?token=${encodeURIComponent(token)}` : `${protocol}//${host}/ws`

    try {
      ws = new WebSocket(wsUrl)
      ws.onopen = () => {
        wsConnected.value = true
        if (reconnectTimer) {
          clearTimeout(reconnectTimer)
          reconnectTimer = null
        }
      }

      ws.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data)
          if (msg.tunnels) {
            tunnels.value = msg.tunnels
          }
          if (msg.stats) {
            stats.value = msg.stats
          }
          if (msg.type === 'log' && msg.data) {
            // New log arrived via WebSocket
            logs.value.unshift(msg.data)
            if (logs.value.length > 500) {
              logs.value.pop()
            }
          }
        } catch (e) {
          console.error('Failed to parse WS message', e)
        }
      }

      ws.onclose = () => {
        wsConnected.value = false
        // Try reconnecting in 3 seconds
        if (!reconnectTimer) {
          reconnectTimer = setTimeout(() => {
            initWebSocket()
          }, 3000)
        }
      }

      ws.onerror = () => {
        wsConnected.value = false
      }
    } catch (err) {
      console.error('WebSocket connection error:', err)
      if (!reconnectTimer) {
        reconnectTimer = setTimeout(() => {
          initWebSocket()
        }, 3000)
      }
    }
  }

  return {
    hosts,
    tunnels,
    logs,
    stats,
    wsConnected,
    fetchHosts,
    fetchTunnels,
    fetchStats,
    fetchLogs,
    initWebSocket,
  }
})
