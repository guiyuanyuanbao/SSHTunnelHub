<template>
  <div class="hosts-page">
    <!-- 顶部操作与筛选栏 -->
    <el-card shadow="never" class="toolbar-card">
      <div class="toolbar-content">
        <div class="toolbar-left">
          <div class="page-title-box">
            <span class="page-title">SSH 主机管理</span>
            <el-tag type="info" round size="small" class="count-tag">
              共 {{ store.hosts.length }} 台主机
            </el-tag>
          </div>
          <el-input
            v-model="searchQuery"
            placeholder="搜索主机名称 / 地址 / 用户名"
            prefix-icon="Search"
            clearable
            class="search-input"
          />
          <el-radio-group v-model="authFilter" class="filter-group">
            <el-radio-button value="all">全部认证</el-radio-button>
            <el-radio-button value="password">密码</el-radio-button>
            <el-radio-button value="private_key">私钥</el-radio-button>
          </el-radio-group>
        </div>
        <div class="toolbar-right">
          <el-button icon="Refresh" @click="handleRefresh" :loading="loading">刷新</el-button>
          <el-button type="primary" icon="Plus" @click="handleAddHost">添加主机</el-button>
        </div>
      </div>
    </el-card>

    <!-- 卡片网格展示 -->
    <div v-loading="loading" class="card-grid-container">
      <el-row :gutter="20">
        <!-- 循环渲染主机卡片 -->
        <el-col
          v-for="host in filteredHosts"
          :key="host.id"
          :xs="24"
          :sm="12"
          :md="12"
          :lg="8"
          :xl="6"
          class="col-card-wrapper"
        >
          <el-card shadow="hover" class="host-item-card">
            <!-- 卡片头部 -->
            <div class="host-card-header">
              <div class="header-main">
                <div class="host-avatar">
                  <el-icon><Platform /></el-icon>
                </div>
                <div class="host-header-info">
                  <div class="host-name-row">
                    <span class="host-name" :title="host.name">{{ host.name }}</span>
                    <span class="host-id-badge">#{{ host.id }}</span>
                  </div>
                  <div class="host-auth-tag">
                    <el-tag
                      :type="host.auth_type === 'password' ? 'success' : 'warning'"
                      size="small"
                      effect="light"
                    >
                      {{ host.auth_type === 'password' ? '密码认证' : '私钥认证' }}
                    </el-tag>
                  </div>
                </div>
              </div>
            </div>

            <!-- 卡片主体内容 -->
            <div class="host-card-body">
              <!-- SSH 连接端点 -->
              <div class="endpoint-box" @click="copyEndpoint(host)" title="点击复制连接串">
                <div class="endpoint-text">
                  <span class="user">{{ host.username }}</span>
                  <span class="at">@</span>
                  <span class="ip">{{ host.host }}</span>
                  <span class="colon">:</span>
                  <span class="port">{{ host.port }}</span>
                </div>
                <el-icon class="copy-icon"><DocumentCopy /></el-icon>
              </div>

              <!-- 凭据状态 -->
              <div class="info-row">
                <span class="label">凭据状态:</span>
                <span v-if="host.auth_type === 'password'" class="value success">
                  <el-icon><Check /></el-icon> 密码已加密保存
                </span>
                <span v-else class="value warning">
                  <el-icon><Key /></el-icon> 私钥已加密保存
                  <small v-if="host.has_passphrase">(带口令)</small>
                </span>
              </div>

              <!-- 备注说明 -->
              <div class="info-row">
                <span class="label">备注用途:</span>
                <span class="value remark" :title="host.remark || '无备注'">
                  {{ host.remark || '暂无备注' }}
                </span>
              </div>

              <!-- 创建时间 -->
              <div class="info-row">
                <span class="label">添加时间:</span>
                <span class="value time">{{ formatDate(host.created_at) }}</span>
              </div>

              <!-- 连通性测试结果提示条 (若有) -->
              <div v-if="testResultMap[host.id]" class="test-inline-result">
                <el-tag
                  v-if="testResultMap[host.id]?.success"
                  type="success"
                  size="small"
                  effect="dark"
                  class="test-tag"
                >
                  🟢 连通正常 ({{ testResultMap[host.id]?.latency }}ms)
                </el-tag>
                <el-tag
                  v-else
                  type="danger"
                  size="small"
                  effect="dark"
                  class="test-tag"
                  :title="testResultMap[host.id]?.error"
                >
                  🔴 连接失败
                </el-tag>
              </div>
            </div>

            <!-- 卡片底栏操作按钮 -->
            <div class="host-card-footer">
              <el-button
                size="small"
                type="primary"
                plain
                :loading="testingMap[host.id]"
                @click="handleTestSaved(host)"
              >
                测试连通
              </el-button>
              <div class="action-btn-group">
                <el-button size="small" @click="handleEditHost(host)">编辑</el-button>
                <el-button size="small" type="danger" plain @click="handleDeleteHost(host)">
                  删除
                </el-button>
              </div>
            </div>
          </el-card>
        </el-col>

        <!-- 添加主机快捷空白卡片 -->
        <el-col :xs="24" :sm="12" :md="12" :lg="8" :xl="6" class="col-card-wrapper">
          <div class="add-host-card" @click="handleAddHost">
            <el-icon class="add-icon"><Plus /></el-icon>
            <span class="add-text">添加新 SSH 主机</span>
            <small class="add-sub">支持密码与私钥认证</small>
          </div>
        </el-col>
      </el-row>

      <!-- 搜索空状态 -->
      <div v-if="filteredHosts.length === 0 && store.hosts.length > 0" class="empty-box">
        <el-empty description="没有找到匹配的 SSH 主机">
          <el-button @click="searchQuery = ''; authFilter = 'all'">清除筛选条件</el-button>
        </el-empty>
      </div>
    </div>

    <!-- 测试结果详细弹窗 -->
    <el-dialog
      v-model="testDialogVisible"
      title="SSH 连接测试结果"
      width="480px"
      :close-on-click-modal="false"
    >
      <div v-if="activeTestResult" class="test-dialog-body">
        <div v-if="activeTestResult.success" class="test-success">
          <el-icon class="test-icon success"><CircleCheckFilled /></el-icon>
          <h3>连接成功！</h3>
          <p><strong>网络往返耗时 (RTT):</strong> {{ activeTestResult.latency }} ms</p>
          <p v-if="activeTestResult.banner"><strong>服务端标识:</strong> <code>{{ activeTestResult.banner }}</code></p>
        </div>
        <div v-else class="test-failed">
          <el-icon class="test-icon error"><CircleCloseFilled /></el-icon>
          <h3>连接失败</h3>
          <p class="error-msg">{{ activeTestResult.error }}</p>
        </div>
      </div>
      <template #footer>
        <el-button @click="testDialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- 编辑/添加弹窗 -->
    <HostDialog ref="hostDialogRef" @success="handleRefresh" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, reactive, onMounted } from 'vue'
import {
  Platform,
  DocumentCopy,
  Check,
  Key,
  Plus,
  CircleCheckFilled,
  CircleCloseFilled,
} from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useTunnelStore } from '../store/useTunnelStore'
import { hostApi } from '../api'
import type { HostVO } from '../types'
import { formatDate } from '../utils/format'
import HostDialog from '../components/HostDialog.vue'

const store = useTunnelStore()
const loading = ref(false)
const searchQuery = ref('')
const authFilter = ref<'all' | 'password' | 'private_key'>('all')
const hostDialogRef = ref<InstanceType<typeof HostDialog>>()

const testingMap = reactive<Record<number, boolean>>({})
const testResultMap = reactive<Record<number, { success: boolean; latency?: number; banner?: string; error?: string }>>({})
const testDialogVisible = ref(false)
const activeTestResult = ref<{ success: boolean; latency?: number; banner?: string; error?: string } | null>(null)

const filteredHosts = computed(() => {
  return store.hosts.filter((h) => {
    if (authFilter.value !== 'all' && h.auth_type !== authFilter.value) {
      return false
    }
    if (!searchQuery.value) return true
    const q = searchQuery.value.toLowerCase()
    return (
      h.name.toLowerCase().includes(q) ||
      h.host.toLowerCase().includes(q) ||
      h.username.toLowerCase().includes(q) ||
      (h.remark && h.remark.toLowerCase().includes(q))
    )
  })
})

async function handleRefresh() {
  loading.value = true
  try {
    await store.fetchHosts()
  } finally {
    loading.value = false
  }
}

function handleAddHost() {
  hostDialogRef.value?.open()
}

function handleEditHost(host: HostVO) {
  hostDialogRef.value?.open(host)
}

function copyEndpoint(host: HostVO) {
  const text = `${host.username}@${host.host}:${host.port}`
  navigator.clipboard.writeText(text).then(() => {
    ElMessage.success(`已复制: ${text}`)
  })
}

async function handleTestSaved(host: HostVO) {
  testingMap[host.id] = true
  try {
    const res = await hostApi.testSaved(host.id)
    testResultMap[host.id] = res
    activeTestResult.value = res
    testDialogVisible.value = true
  } catch (err: any) {
    const failRes = { success: false, error: err.message }
    testResultMap[host.id] = failRes
    activeTestResult.value = failRes
    testDialogVisible.value = true
  } finally {
    testingMap[host.id] = false
  }
}

async function handleDeleteHost(host: HostVO) {
  try {
    await ElMessageBox.confirm(
      `确定要删除主机「${host.name}」吗？如果已有隧道绑定至此主机，需要先移除对应隧道。`,
      '删除确认',
      {
        confirmButtonText: '确定删除',
        cancelButtonText: '取消',
        type: 'warning',
      }
    )

    await hostApi.delete(host.id)
    ElMessage.success('主机已成功删除')
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
.hosts-page {
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
  gap: 10px;
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

.host-item-card {
  border-radius: 10px;
  border: 1px solid #e5e7eb;
  transition: all 0.25s ease;
  display: flex;
  flex-direction: column;
  width: 100%;
  height: 100%;
  box-sizing: border-box;
}

.host-item-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.08), 0 4px 6px -2px rgba(0, 0, 0, 0.04);
}

:deep(.host-item-card .el-card__body) {
  padding: 18px 20px;
  display: flex;
  flex-direction: column;
  flex: 1;
  height: 100%;
  box-sizing: border-box;
  justify-content: space-between;
}

.host-card-header {
  border-bottom: 1px solid #f3f4f6;
  padding-bottom: 14px;
  margin-bottom: 14px;
}

.header-main {
  display: flex;
  align-items: center;
  gap: 12px;
}

.host-avatar {
  width: 42px;
  height: 42px;
  border-radius: 8px;
  background: #eff6ff;
  color: #2563eb;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 22px;
  flex-shrink: 0;
}

.host-header-info {
  flex: 1;
  min-width: 0;
}

.host-name-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.host-name {
  font-size: 15px;
  font-weight: 600;
  color: #111827;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.host-id-badge {
  font-size: 11px;
  color: #9ca3af;
  font-family: monospace;
}

.host-card-body {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.endpoint-box {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  padding: 8px 12px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  cursor: pointer;
  transition: all 0.2s;
}

.endpoint-box:hover {
  background: #f1f5f9;
  border-color: #cbd5e1;
}

.endpoint-text {
  font-family: monospace;
  font-size: 13px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.endpoint-text .user {
  color: #0284c7;
  font-weight: 600;
}

.endpoint-text .at,
.endpoint-text .colon {
  color: #94a3b8;
  margin: 0 1px;
}

.endpoint-text .ip {
  color: #334155;
  font-weight: 600;
}

.endpoint-text .port {
  color: #d97706;
}

.copy-icon {
  color: #94a3b8;
  font-size: 14px;
  margin-left: 6px;
}

.endpoint-box:hover .copy-icon {
  color: #2563eb;
}

.info-row {
  display: flex;
  align-items: center;
  font-size: 13px;
  gap: 8px;
}

.info-row .label {
  color: #64748b;
  width: 68px;
  flex-shrink: 0;
}

.info-row .value {
  color: #1e293b;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
}

.info-row .value.success {
  color: #16a34a;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.info-row .value.warning {
  color: #d97706;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.info-row .value.remark {
  color: #64748b;
  font-style: italic;
}

.info-row .value.time {
  color: #94a3b8;
  font-size: 12px;
}

.test-inline-result {
  margin-top: 4px;
}

.host-card-footer {
  border-top: 1px solid #f3f4f6;
  padding-top: 14px;
  margin-top: auto;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.action-btn-group {
  display: flex;
  gap: 8px;
}

/* 虚线新建卡片 */
.add-host-card {
  border: 2px dashed #cbd5e1;
  border-radius: 10px;
  height: 100%;
  width: 100%;
  min-height: 250px;
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

.add-host-card:hover {
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

/* 测试弹窗样式 */
.test-dialog-body {
  text-align: center;
  padding: 10px 0;
}

.test-icon {
  font-size: 54px;
  margin-bottom: 8px;
}

.test-icon.success {
  color: #16a34a;
}

.test-icon.error {
  color: #dc2626;
}

.error-msg {
  color: #dc2626;
  background: #fef2f2;
  padding: 10px;
  border-radius: 6px;
  font-size: 13px;
  word-break: break-all;
  text-align: left;
}
</style>
