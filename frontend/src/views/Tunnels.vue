<template>
  <div class="tunnels-page">
    <!-- 顶部操作与筛选栏 -->
    <el-card shadow="never" class="toolbar-card">
      <div class="toolbar-content">
        <div class="toolbar-left">
          <div class="page-title-box">
            <span class="page-title">SSH 隧道管理</span>
            <el-tag type="info" round size="small" class="count-tag">
              共 {{ store.tunnels.length }} 条隧道
            </el-tag>
            <el-tag v-if="runningCount > 0" type="success" round size="small" class="count-tag">
              {{ runningCount }} 运行中
            </el-tag>
            <el-tag v-if="reconnectingCount > 0" type="warning" round size="small" class="count-tag">
              {{ reconnectingCount }} 重连中
            </el-tag>
          </div>
          <el-input
            v-model="searchQuery"
            placeholder="搜索隧道名 / 端口 / 目标地址 / 主机"
            prefix-icon="Search"
            clearable
            class="search-input"
          />
          <el-radio-group v-model="typeFilter" class="filter-group">
            <el-radio-button value="all">全部模式</el-radio-button>
            <el-radio-button value="forward">正向 (-L)</el-radio-button>
            <el-radio-button value="reverse">反向 (-R)</el-radio-button>
            <el-radio-button value="running">运行中</el-radio-button>
          </el-radio-group>
        </div>
        <div class="toolbar-right">
          <el-button icon="Refresh" @click="handleRefresh" :loading="loading">刷新</el-button>
          <el-button type="primary" icon="Plus" @click="handleCreateTunnel">创建隧道</el-button>
        </div>
      </div>
    </el-card>

    <!-- 卡片网格展示 -->
    <div v-loading="loading" class="card-grid-container">
      <el-row :gutter="20">
        <!-- 循环渲染隧道卡片 -->
        <el-col
          v-for="tunnel in filteredTunnels"
          :key="tunnel.id"
          :xs="24"
          :sm="24"
          :md="12"
          :lg="12"
          :xl="8"
          class="col-card-wrapper"
        >
          <el-card
            shadow="hover"
            class="tunnel-item-card"
            :class="`status-${tunnel.runtime.status}`"
          >
            <!-- 顶部装饰色条 -->
            <div class="card-status-bar" :class="tunnel.runtime.status"></div>

            <!-- 卡片头部 -->
            <div class="tunnel-card-header">
              <div class="header-left-meta">
                <el-tag
                  :type="tunnel.type === 'forward' ? 'primary' : 'warning'"
                  size="small"
                  effect="dark"
                  class="type-tag"
                >
                  {{ tunnel.type === 'forward' ? '正向 (-L)' : '反向 (-R)' }}
                </el-tag>
                <span class="tunnel-name" :title="tunnel.name">{{ tunnel.name }}</span>
                <span class="tunnel-id-badge">#{{ tunnel.id }}</span>
              </div>
              <div class="header-right-meta">
                <el-tooltip
                  v-if="tunnel.auto_start"
                  content="系统启动时自动建立并保持隧道"
                  placement="top"
                >
                  <el-tag size="small" type="success" effect="plain" class="autostart-tag">
                    自启
                  </el-tag>
                </el-tooltip>
                <el-tag
                  v-if="tunnel.runtime.status === 'running' && tunnel.runtime.health === 'unhealthy'"
                  size="small"
                  type="danger"
                  effect="light"
                  :title="tunnel.runtime.health_message"
                >
                  目标异常
                </el-tag>
                <el-tag
                  v-else-if="tunnel.runtime.status === 'running' && tunnel.runtime.health === 'healthy'"
                  size="small"
                  type="success"
                  effect="plain"
                >
                  目标正常
                </el-tag>
                <StatusBadge :status="tunnel.runtime.status" />
              </div>
            </div>

            <!-- 卡片主体内容 -->
            <div class="tunnel-card-body">
              <!-- 关联主机信息 -->
              <div class="host-info-bar">
                <el-icon class="host-icon"><Platform /></el-icon>
                <span class="host-text">
                  <strong v-if="tunnel.host">{{ tunnel.host.name }}</strong>
                  <span v-else>主机 #{{ tunnel.host_id }}</span>
                  <small class="host-sub" v-if="tunnel.host">
                    ({{ tunnel.host.username }}@{{ tunnel.host.host }}:{{ tunnel.host.port }})
                  </small>
                </span>
              </div>

              <!-- 端口转发图解路由盒 (3列严格对称居中) -->
              <div class="route-box-container">
                <div class="route-node source">
                  <div class="node-label">
                    {{ tunnel.type === 'forward' ? 'Hub 本地监听' : '远程主机监听' }}
                  </div>
                  <div class="node-addr" :title="`${tunnel.listen_host}:${tunnel.listen_port}`">
                    {{ tunnel.listen_host }}:{{ tunnel.listen_port }}
                  </div>
                </div>

                <div class="route-arrow-flow">
                  <span class="flow-pill">SSH 通道</span>
                  <div class="flow-arrow-wrap">
                    <span class="flow-dash"></span>
                    <el-icon class="flow-arrow"><Right /></el-icon>
                    <span class="flow-dash"></span>
                  </div>
                </div>

                <div class="route-node target">
                  <div class="node-label">
                    {{ tunnel.type === 'forward' ? '远程目标服务' : '本地/内网目标' }}
                  </div>
                  <div class="node-addr" :title="`${tunnel.target_host}:${tunnel.target_port}`">
                    {{ tunnel.target_host }}:{{ tunnel.target_port }}
                  </div>
                </div>
              </div>

              <!-- 4格实时性能指标 -->
              <div class="metrics-grid">
                <div class="metric-cell">
                  <span class="m-label">活跃连接</span>
                  <span class="m-val highlight">{{ tunnel.runtime.active_conns }}</span>
                </div>
                <div class="metric-cell">
                  <span class="m-label">运行时长</span>
                  <span class="m-val">{{ formatDuration(tunnel.runtime.uptime) }}</span>
                </div>
                <div class="metric-cell">
                  <span class="m-label">下行流量 (In)</span>
                  <span class="m-val">{{ formatBytes(tunnel.runtime.bytes_in) }}</span>
                </div>
                <div class="metric-cell">
                  <span class="m-label">上行流量 (Out)</span>
                  <span class="m-val">{{ formatBytes(tunnel.runtime.bytes_out) }}</span>
                </div>
              </div>

              <!-- 目标端健康异常告警 -->
              <div v-if="tunnel.runtime.status === 'running' && tunnel.runtime.health === 'unhealthy'" class="health-alert-box">
                <el-icon class="health-icon"><WarningFilled /></el-icon>
                <div class="health-text" :title="tunnel.runtime.health_message">
                  目标端探测异常: {{ tunnel.runtime.health_message }}
                </div>
              </div>

              <!-- 错误告警区 (若存在报错或重连原因) -->
              <div v-if="tunnel.runtime.last_error" class="error-alert-box">
                <el-icon class="err-icon"><WarningFilled /></el-icon>
                <div class="err-text" :title="tunnel.runtime.last_error">
                  {{ tunnel.runtime.last_error }}
                </div>
              </div>

              <!-- 备注说明 (可选) -->
              <div v-if="tunnel.remark" class="remark-row">
                <span class="remark-label">备注:</span>
                <span class="remark-val">{{ tunnel.remark }}</span>
              </div>
            </div>

            <!-- 卡片底栏操作按钮 -->
            <div class="tunnel-card-footer">
              <div class="control-actions">
                <!-- 启动 / 停止 按钮 -->
                <el-button
                  v-if="tunnel.runtime.status === 'stopped' || tunnel.runtime.status === 'error'"
                  type="success"
                  size="small"
                  icon="VideoPlay"
                  :loading="actionLoading[tunnel.id]"
                  @click="handleStart(tunnel)"
                >
                  启动
                </el-button>
                <el-button
                  v-else
                  type="danger"
                  size="small"
                  icon="VideoPause"
                  :loading="actionLoading[tunnel.id]"
                  @click="handleStop(tunnel)"
                >
                  停止
                </el-button>

                <!-- 重启按钮 -->
                <el-button
                  size="small"
                  icon="RefreshRight"
                  :loading="actionLoading[tunnel.id]"
                  @click="handleRestart(tunnel)"
                >
                  重启
                </el-button>
              </div>

              <div class="manage-actions">
                <el-button size="small" icon="Tickets" @click="$router.push(`/logs?tunnel_id=${tunnel.id}`)">
                  日志
                </el-button>
                <el-button size="small" @click="handleEdit(tunnel)">编辑</el-button>
                <el-button size="small" type="danger" plain @click="handleDelete(tunnel)">
                  删除
                </el-button>
              </div>
            </div>
          </el-card>
        </el-col>

        <!-- 创建隧道快捷空白卡片 -->
        <el-col :xs="24" :sm="24" :md="12" :lg="12" :xl="8" class="col-card-wrapper">
          <div class="add-tunnel-card" @click="handleCreateTunnel">
            <el-icon class="add-icon"><Plus /></el-icon>
            <span class="add-text">创建新 SSH 隧道</span>
            <small class="add-sub">支持正向 (-L) 与反向 (-R) 模式</small>
          </div>
        </el-col>
      </el-row>

      <!-- 搜索空状态 -->
      <div v-if="filteredTunnels.length === 0 && store.tunnels.length > 0" class="empty-box">
        <el-empty description="没有找到匹配的 SSH 隧道">
          <el-button @click="searchQuery = ''; typeFilter = 'all'">清除筛选条件</el-button>
        </el-empty>
      </div>
    </div>

    <!-- 弹窗 -->
    <TunnelDialog
      ref="tunnelDialogRef"
      :hosts="store.hosts"
      @success="handleRefresh"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import {
  Platform,
  Right,
  Plus,
  VideoPlay,
  VideoPause,
  RefreshRight,
  WarningFilled,
  Tickets,
} from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useTunnelStore } from '../store/useTunnelStore'
import { tunnelApi } from '../api'
import type { TunnelVO } from '../types'
import { formatBytes, formatDuration } from '../utils/format'
import StatusBadge from '../components/StatusBadge.vue'
import TunnelDialog from '../components/TunnelDialog.vue'

const store = useTunnelStore()
const loading = ref(false)
const searchQuery = ref('')
const typeFilter = ref<'all' | 'forward' | 'reverse' | 'running'>('all')
const tunnelDialogRef = ref<InstanceType<typeof TunnelDialog>>()
const actionLoading = reactive<Record<number, boolean>>({})

const runningCount = computed(() => {
  return store.tunnels.filter((t) => t.runtime.status === 'running').length
})

const reconnectingCount = computed(() => {
  return store.tunnels.filter((t) => t.runtime.status === 'reconnecting').length
})

const filteredTunnels = computed(() => {
  return store.tunnels.filter((t) => {
    if (typeFilter.value === 'forward' && t.type !== 'forward') return false
    if (typeFilter.value === 'reverse' && t.type !== 'reverse') return false
    if (typeFilter.value === 'running' && t.runtime.status !== 'running' && t.runtime.status !== 'reconnecting') {
      return false
    }

    if (!searchQuery.value) return true
    const q = searchQuery.value.toLowerCase()
    return (
      t.name.toLowerCase().includes(q) ||
      String(t.listen_port).includes(q) ||
      String(t.target_port).includes(q) ||
      t.target_host.toLowerCase().includes(q) ||
      t.listen_host.toLowerCase().includes(q) ||
      (t.host && t.host.name.toLowerCase().includes(q))
    )
  })
})

async function handleRefresh() {
  loading.value = true
  try {
    await store.fetchTunnels()
    await store.fetchHosts()
    await store.fetchStats()
  } finally {
    loading.value = false
  }
}

function handleCreateTunnel() {
  if (store.hosts.length === 0) {
    ElMessage.warning('请先在「主机管理」中添加至少一台 SSH 主机')
    return
  }
  tunnelDialogRef.value?.open()
}

function handleEdit(tunnel: TunnelVO) {
  tunnelDialogRef.value?.open(tunnel)
}

async function handleStart(tunnel: TunnelVO) {
  actionLoading[tunnel.id] = true
  try {
    await tunnelApi.start(tunnel.id)
    ElMessage.success(`隧道 [${tunnel.name}] 启动指令已发出`)
    handleRefresh()
  } catch (err: any) {
    ElMessage.error(err.message || '启动失败')
  } finally {
    actionLoading[tunnel.id] = false
  }
}

async function handleStop(tunnel: TunnelVO) {
  actionLoading[tunnel.id] = true
  try {
    await tunnelApi.stop(tunnel.id)
    ElMessage.success(`隧道 [${tunnel.name}] 已停止`)
    handleRefresh()
  } catch (err: any) {
    ElMessage.error(err.message || '停止失败')
  } finally {
    actionLoading[tunnel.id] = false
  }
}

async function handleRestart(tunnel: TunnelVO) {
  actionLoading[tunnel.id] = true
  try {
    await tunnelApi.restart(tunnel.id)
    ElMessage.success(`隧道 [${tunnel.name}] 重启指令已发出`)
    handleRefresh()
  } catch (err: any) {
    ElMessage.error(err.message || '重启失败')
  } finally {
    actionLoading[tunnel.id] = false
  }
}

async function handleDelete(tunnel: TunnelVO) {
  try {
    await ElMessageBox.confirm(
      `确定要删除隧道「${tunnel.name}」吗？若该隧道正在运行，将被同步停止。`,
      '删除确认',
      {
        confirmButtonText: '确定删除',
        cancelButtonText: '取消',
        type: 'warning',
      }
    )

    await tunnelApi.delete(tunnel.id)
    ElMessage.success('隧道已成功删除')
    handleRefresh()
  } catch (err: any) {
    if (err !== 'cancel') {
      ElMessage.error(err.message || '删除失败')
    }
  }
}

onMounted(() => {
  handleRefresh()
})
</script>

<style scoped>
.tunnels-page {
  padding: 8px 12px;
}

.toolbar-card {
  border-radius: 8px;
  margin-bottom: 20px;
}

:deep(.toolbar-card .el-card__body) {
  padding: 16px 20px;
}

.toolbar-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 16px;
}

.toolbar-left {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 16px;
}

.page-title-box {
  display: flex;
  align-items: center;
  gap: 8px;
}

.page-title {
  font-size: 16px;
  font-weight: 700;
  color: #1f2937;
}

.count-tag {
  font-weight: 500;
}

.search-input {
  width: 260px;
}

.card-grid-container {
  min-height: 200px;
}

.card-grid-container :deep(.el-row) {
  display: flex;
  flex-wrap: wrap;
}

.col-card-wrapper {
  margin-bottom: 20px;
  display: flex;
  flex-direction: column;
}

.tunnel-item-card {
  border-radius: 10px;
  border: 1px solid #e5e7eb;
  transition: all 0.25s ease;
  position: relative;
  overflow: hidden;
  width: 100%;
  height: 100%;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
}

.tunnel-item-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.08), 0 4px 6px -2px rgba(0, 0, 0, 0.04);
}

:deep(.tunnel-item-card .el-card__body) {
  padding: 18px 20px;
  display: flex;
  flex-direction: column;
  flex: 1;
  height: 100%;
  box-sizing: border-box;
  justify-content: space-between;
}

/* 顶部状态条 */
.card-status-bar {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 4px;
  background: #9ca3af;
}

.card-status-bar.running {
  background: #10b981;
}

.card-status-bar.reconnecting {
  background: #f59e0b;
}

.card-status-bar.error {
  background: #ef4444;
}

.tunnel-card-header {
  border-bottom: 1px solid #f3f4f6;
  padding-bottom: 14px;
  margin-bottom: 14px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.header-left-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  min-width: 0;
}

.type-tag {
  font-weight: 600;
}

.tunnel-name {
  font-size: 15px;
  font-weight: 600;
  color: #111827;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tunnel-id-badge {
  font-size: 11px;
  color: #9ca3af;
  font-family: monospace;
}

.header-right-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.autostart-tag {
  font-size: 11px;
  padding: 0 6px;
}

.tunnel-card-body {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.host-info-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: #475569;
  background: #f8fafc;
  padding: 6px 10px;
  border-radius: 6px;
}

.host-icon {
  color: #3b82f6;
  font-size: 16px;
}

.host-sub {
  color: #94a3b8;
  margin-left: 4px;
}

/* 路由图解盒子 (3列严格对称网格布局) */
.route-box-container {
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  align-items: center;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 10px 12px;
  gap: 10px;
}

.route-node {
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  padding: 7px 8px;
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  min-width: 0;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.03);
}

.route-node.target {
  border-color: #bfdbfe;
  background: #eff6ff;
}

.route-node .node-label {
  font-size: 11px;
  color: #64748b;
  margin-bottom: 3px;
  white-space: nowrap;
}

.route-node .node-addr {
  font-family: monospace;
  font-size: 13px;
  font-weight: 700;
  color: #0f172a;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  width: 100%;
}

.route-node.target .node-addr {
  color: #2563eb;
}

.route-arrow-flow {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-width: 76px;
}

.flow-pill {
  font-size: 10px;
  color: #475569;
  background: #e2e8f0;
  border-radius: 10px;
  padding: 1px 7px;
  margin-bottom: 3px;
  white-space: nowrap;
  font-weight: 600;
}

.flow-arrow-wrap {
  display: flex;
  align-items: center;
  gap: 3px;
  color: #3b82f6;
}

.flow-dash {
  width: 10px;
  height: 1px;
  background: #93c5fd;
}

.flow-arrow {
  font-size: 15px;
}

/* 4格指标 */
.metrics-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 8px;
  background: #f8fafc;
  border-radius: 6px;
  padding: 8px 12px;
}

.metric-cell {
  display: flex;
  flex-direction: column;
}

.m-label {
  font-size: 11px;
  color: #94a3b8;
  margin-bottom: 2px;
}

.m-val {
  font-size: 13px;
  font-weight: 600;
  color: #334155;
  font-family: monospace;
}

.m-val.highlight {
  color: #10b981;
}

.health-alert-box {
  background: #fffbeb;
  border: 1px solid #fde68a;
  color: #b45309;
  border-radius: 6px;
  padding: 8px 10px;
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
}

.health-icon {
  font-size: 16px;
  flex-shrink: 0;
  color: #f59e0b;
}

.health-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.error-alert-box {
  background: #fef2f2;
  border: 1px solid #fecaca;
  color: #b91c1c;
  border-radius: 6px;
  padding: 8px 10px;
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
}

.err-icon {
  font-size: 16px;
  flex-shrink: 0;
}

.err-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.remark-row {
  font-size: 12px;
  color: #64748b;
  display: flex;
  gap: 4px;
}

.tunnel-card-footer {
  border-top: 1px solid #f3f4f6;
  padding-top: 14px;
  margin-top: auto;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.control-actions,
.manage-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

/* 虚线新建卡片 */
.add-tunnel-card {
  border: 2px dashed #cbd5e1;
  border-radius: 10px;
  height: 100%;
  width: 100%;
  min-height: 280px;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  background: rgba(255, 255, 255, 0.6);
  transition: all 0.25s ease;
  color: #64748b;
}

.add-tunnel-card:hover {
  border-color: #3b82f6;
  background: #eff6ff;
  color: #2563eb;
  transform: translateY(-2px);
}

.add-icon {
  font-size: 36px;
  margin-bottom: 10px;
}

.add-text {
  font-size: 15px;
  font-weight: 600;
  margin-bottom: 4px;
}

.add-sub {
  font-size: 12px;
  color: #94a3b8;
}

.empty-box {
  padding: 40px 0;
}
</style>
