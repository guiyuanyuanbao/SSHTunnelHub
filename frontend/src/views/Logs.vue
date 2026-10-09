<template>
  <div class="logs-page">
    <el-card shadow="never" class="toolbar-card">
      <div class="toolbar-content">
        <div class="toolbar-left">
          <div class="page-title-box">
            <span class="page-title">运行与网络日志</span>
            <el-tag type="info" round size="small" class="count-tag">
              {{ filteredLogs.length }} 条记录
            </el-tag>
          </div>

          <!-- 级别筛选 -->
          <el-radio-group v-model="levelFilter" size="small" class="level-radio">
            <el-radio-button value="ALL">全部</el-radio-button>
            <el-radio-button value="INFO">INFO</el-radio-button>
            <el-radio-button value="SUCCESS">SUCCESS</el-radio-button>
            <el-radio-button value="WARN">WARN</el-radio-button>
            <el-radio-button value="ERROR">ERROR</el-radio-button>
          </el-radio-group>

          <!-- 隧道筛选 -->
          <el-select
            v-model="tunnelFilter"
            placeholder="全部隧道"
            clearable
            size="small"
            style="width: 180px"
          >
            <el-option label="全部隧道" :value="0" />
            <el-option
              v-for="t in store.tunnels"
              :key="t.id"
              :label="`#${t.id} ${t.name}`"
              :value="t.id"
            />
          </el-select>

          <!-- 关键词搜索 -->
          <el-input
            v-model="searchKeyword"
            placeholder="搜索日志内容..."
            prefix-icon="Search"
            clearable
            size="small"
            style="width: 200px"
          />
        </div>

        <div class="toolbar-right">
          <div class="autoscroll-box">
            <el-switch v-model="autoScroll" size="small" active-text="实时自动追踪" />
          </div>
          <el-button size="small" icon="DocumentCopy" @click="handleCopyLogs">复制</el-button>
          <el-button size="small" icon="Refresh" @click="handleRefresh" :loading="loading">刷新</el-button>
          <el-button size="small" type="danger" plain icon="Delete" @click="handleClear">清空</el-button>
        </div>
      </div>
    </el-card>

    <!-- 终端风格日志控制台 -->
    <el-card shadow="never" class="console-card">
      <div ref="logContainerRef" class="log-stream-container">
        <div v-if="filteredLogs.length === 0" class="empty-log-box">
          <el-empty description="暂无符合条件的运行日志" />
        </div>

        <div
          v-for="entry in filteredLogs"
          :key="entry.id"
          class="log-line"
          :class="`level-${entry.level.toLowerCase()}`"
        >
          <span class="log-time">{{ formatLogTime(entry.timestamp) }}</span>
          <span class="log-level-badge" :class="entry.level.toLowerCase()">
            {{ entry.level }}
          </span>
          <span class="log-tag-badge">{{ entry.tag }}</span>
          <span v-if="entry.tunnel_name" class="log-tunnel-badge">
            [{{ entry.tunnel_name }}]
          </span>
          <span class="log-msg">{{ entry.message }}</span>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, nextTick, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useTunnelStore } from '../store/useTunnelStore'
import { logApi } from '../api'
import type { LogEntry } from '../types'

const route = useRoute()
const store = useTunnelStore()
const loading = ref(false)
const levelFilter = ref('ALL')
const tunnelFilter = ref<number>(0)
const searchKeyword = ref('')
const autoScroll = ref(true)
const logContainerRef = ref<HTMLElement>()

const filteredLogs = computed(() => {
  return store.logs.filter((l) => {
    if (levelFilter.value !== 'ALL' && l.level !== levelFilter.value) {
      return false
    }
    if (tunnelFilter.value && tunnelFilter.value > 0 && l.tunnel_id !== tunnelFilter.value) {
      return false
    }
    if (!searchKeyword.value) return true
    const kw = searchKeyword.value.toLowerCase()
    return (
      l.message.toLowerCase().includes(kw) ||
      (l.tunnel_name && l.tunnel_name.toLowerCase().includes(kw)) ||
      l.tag.toLowerCase().includes(kw)
    )
  })
})

function formatLogTime(timestamp: string): string {
  try {
    const d = new Date(timestamp)
    const hh = String(d.getHours()).padStart(2, '0')
    const mm = String(d.getMinutes()).padStart(2, '0')
    const ss = String(d.getSeconds()).padStart(2, '0')
    const ms = String(d.getMilliseconds()).padStart(3, '0')
    return `${hh}:${mm}:${ss}.${ms}`
  } catch (e) {
    return timestamp
  }
}

async function handleRefresh() {
  loading.value = true
  try {
    await store.fetchLogs({ limit: 300 })
    await store.fetchTunnels()
    scrollToBottom()
  } finally {
    loading.value = false
  }
}

async function handleClear() {
  try {
    await ElMessageBox.confirm('确定要清空所有当前运行日志吗？', '提示', {
      type: 'warning',
      confirmButtonText: '确定清空',
      cancelButtonText: '取消',
    })
    await logApi.clear()
    store.logs = []
    ElMessage.success('运行日志已清空')
  } catch (e) {}
}

function handleCopyLogs() {
  if (filteredLogs.value.length === 0) {
    ElMessage.warning('当前无可用日志')
    return
  }
  const text = filteredLogs.value
    .map(
      (l) =>
        `[${l.timestamp}] [${l.level}] [${l.tag}] ${l.tunnel_name ? '[' + l.tunnel_name + '] ' : ''}${l.message}`
    )
    .join('\n')

  navigator.clipboard.writeText(text).then(() => {
    ElMessage.success(`已复制 ${filteredLogs.value.length} 条日志`)
  })
}

function scrollToBottom() {
  if (!autoScroll.value) return
  nextTick(() => {
    if (logContainerRef.value) {
      logContainerRef.value.scrollTop = 0 // Since logs are stored newest first at top, top is most recent
    }
  })
}

watch(
  () => store.logs.length,
  () => {
    scrollToBottom()
  }
)

onMounted(() => {
  // If route has tunnel_id query param
  if (route.query.tunnel_id) {
    const tid = parseInt(route.query.tunnel_id as string, 10)
    if (!isNaN(tid)) {
      tunnelFilter.value = tid
    }
  }

  handleRefresh()
})
</script>

<style scoped>
.logs-page {
  padding: 8px 12px;
  display: flex;
  flex-direction: column;
  height: calc(100vh - 108px);
}

.toolbar-card {
  border-radius: 8px;
  margin-bottom: 16px;
  flex-shrink: 0;
}

:deep(.toolbar-card .el-card__body) {
  padding: 14px 20px;
}

.toolbar-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 14px;
}

.toolbar-left {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
}

.page-title-box {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-right: 8px;
}

.page-title {
  font-size: 16px;
  font-weight: 700;
  color: #1f2937;
}

.count-tag {
  font-weight: 500;
}

.toolbar-right {
  display: flex;
  align-items: center;
  gap: 10px;
}

.autoscroll-box {
  margin-right: 6px;
}

.console-card {
  border-radius: 10px;
  flex: 1;
  display: flex;
  flex-direction: column;
  background: #0f172a;
  border: 1px solid #1e293b;
  overflow: hidden;
}

:deep(.console-card .el-card__body) {
  padding: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  height: 100%;
}

.log-stream-container {
  flex: 1;
  overflow-y: auto;
  padding: 14px 16px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
  font-size: 13px;
  line-height: 1.6;
  color: #e2e8f0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.empty-log-box {
  padding: 60px 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

:deep(.empty-log-box .el-empty__description p) {
  color: #64748b;
}

.log-line {
  display: flex;
  align-items: baseline;
  gap: 8px;
  padding: 3px 6px;
  border-radius: 4px;
  transition: background 0.15s;
  word-break: break-all;
}

.log-line:hover {
  background: rgba(255, 255, 255, 0.05);
}

.log-time {
  color: #64748b;
  font-size: 12px;
  flex-shrink: 0;
}

.log-level-badge {
  font-size: 11px;
  font-weight: 700;
  padding: 0 5px;
  border-radius: 3px;
  flex-shrink: 0;
  line-height: 1.4;
}

.log-level-badge.info {
  background: #1e3a8a;
  color: #60a5fa;
}

.log-level-badge.success {
  background: #14532d;
  color: #4ade80;
}

.log-level-badge.warn {
  background: #78350f;
  color: #fbbf24;
}

.log-level-badge.error {
  background: #7f1d1d;
  color: #f87171;
}

.log-tag-badge {
  background: #334155;
  color: #94a3b8;
  font-size: 11px;
  padding: 0 5px;
  border-radius: 3px;
  flex-shrink: 0;
  line-height: 1.4;
}

.log-tunnel-badge {
  color: #c084fc;
  font-weight: 600;
  font-size: 12px;
  flex-shrink: 0;
}

.log-msg {
  color: #f1f5f9;
  flex: 1;
}

.log-line.level-error .log-msg {
  color: #fca5a5;
  font-weight: 600;
}

.log-line.level-warn .log-msg {
  color: #fde047;
}

.log-line.level-success .log-msg {
  color: #86efac;
}
</style>
