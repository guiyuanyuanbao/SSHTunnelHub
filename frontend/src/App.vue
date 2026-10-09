<template>
  <el-container class="app-container">
    <!-- 侧边栏 -->
    <el-aside width="230px" class="app-sidebar">
      <div class="logo-box">
        <el-icon class="logo-icon"><Connection /></el-icon>
        <div class="logo-text">
          <span class="brand">SSHTunnelHub</span>
          <span class="subtitle">隧道管理中心</span>
        </div>
      </div>

      <el-menu
        :default-active="$route.path"
        router
        class="sidebar-menu"
        background-color="#1f2937"
        text-color="#9ca3af"
        active-text-color="#ffffff"
      >
        <el-menu-item index="/">
          <el-icon><Odometer /></el-icon>
          <span>概览看板</span>
        </el-menu-item>
        <el-menu-item index="/hosts">
          <el-icon><Platform /></el-icon>
          <span>主机管理</span>
        </el-menu-item>
        <el-menu-item index="/tunnels">
          <el-icon><Connection /></el-icon>
          <span>隧道管理</span>
        </el-menu-item>
        <el-menu-item index="/logs">
          <el-icon><Tickets /></el-icon>
          <span>运行日志</span>
        </el-menu-item>
      </el-menu>

      <div class="sidebar-footer">
        <div class="version-tag">v1.0.0 · Go & Vue</div>
      </div>
    </el-aside>

    <!-- 主体区域 -->
    <el-container class="main-container">
      <!-- 顶栏 -->
      <el-header height="60px" class="app-header">
        <div class="header-breadcrumb">
          <span class="current-page-title">{{ $route.meta.title || 'SSHTunnelHub' }}</span>
        </div>

        <div class="header-status">
          <el-tag
            :type="store.wsConnected ? 'success' : 'warning'"
            effect="plain"
            round
            class="ws-tag"
          >
            <span class="pulse-dot" :class="{ active: store.wsConnected }"></span>
            {{ store.wsConnected ? '实时监控已连接' : '通信重连中...' }}
          </el-tag>
          <el-divider direction="vertical" />
          <div class="quick-stat">
            活动隧道: <strong>{{ store.stats.running_tunnels }}</strong> / {{ store.stats.total_tunnels }}
          </div>
        </div>
      </el-header>

      <!-- 内容区 -->
      <el-main class="app-content">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { Odometer, Platform, Connection, Tickets } from '@element-plus/icons-vue'
import { useTunnelStore } from './store/useTunnelStore'

const store = useTunnelStore()

onMounted(() => {
  store.initWebSocket()
})
</script>

<style>
/* Global resets */
body, html {
  margin: 0;
  padding: 0;
  height: 100%;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
  background-color: #f3f4f6;
}

#app {
  height: 100%;
}
</style>

<style scoped>
.app-container {
  height: 100vh;
  overflow: hidden;
}

.app-sidebar {
  background-color: #1f2937;
  display: flex;
  flex-direction: column;
  box-shadow: 2px 0 8px rgba(0, 0, 0, 0.05);
}

.logo-box {
  height: 60px;
  display: flex;
  align-items: center;
  padding: 0 18px;
  background-color: #111827;
  gap: 12px;
}

.logo-icon {
  font-size: 26px;
  color: #3b82f6;
}

.logo-text {
  display: flex;
  flex-direction: column;
}

.logo-text .brand {
  font-size: 16px;
  font-weight: 700;
  color: #ffffff;
  letter-spacing: 0.5px;
}

.logo-text .subtitle {
  font-size: 11px;
  color: #9ca3af;
}

.sidebar-menu {
  border-right: none;
  flex: 1;
  padding-top: 10px;
}

.sidebar-menu :deep(.el-menu-item) {
  height: 48px;
  margin: 4px 10px;
  border-radius: 6px;
}

.sidebar-menu :deep(.el-menu-item.is-active) {
  background-color: #2563eb !important;
  color: #ffffff !important;
  font-weight: 600;
}

.sidebar-footer {
  padding: 16px;
  text-align: center;
}

.version-tag {
  font-size: 11px;
  color: #6b7280;
}

.main-container {
  background-color: #f8fafc;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.app-header {
  background: #ffffff;
  border-bottom: 1px solid #e5e7eb;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
}

.current-page-title {
  font-size: 17px;
  font-weight: 600;
  color: #111827;
}

.header-status {
  display: flex;
  align-items: center;
  gap: 12px;
}

.ws-tag {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
}

.pulse-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background-color: #e6a23c;
  display: inline-block;
}

.pulse-dot.active {
  background-color: #67c23a;
  box-shadow: 0 0 6px #67c23a;
  animation: pulse 2s infinite;
}

.quick-stat {
  font-size: 13px;
  color: #4b5563;
}

.app-content {
  padding: 20px 24px;
  overflow-y: auto;
  box-sizing: border-box;
}

:deep(.el-card) {
  --el-card-padding: 20px;
}
</style>
