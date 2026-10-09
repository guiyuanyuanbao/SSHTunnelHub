<template>
  <div class="dashboard-container">
    <!-- 统计指标网格 (统一高度与等宽布局) -->
    <div class="metrics-grid-container">
      <!-- 主机总数 -->
      <el-card shadow="hover" class="metric-card">
        <div class="metric-icon host"><el-icon><Platform /></el-icon></div>
        <div class="metric-info">
          <div class="metric-title">主机总数</div>
          <div class="metric-value-wrap">
            <span class="metric-number">{{ store.stats.total_hosts }}</span>
            <span class="metric-unit">台</span>
          </div>
        </div>
      </el-card>

      <!-- 隧道总数 -->
      <el-card shadow="hover" class="metric-card">
        <div class="metric-icon tunnel"><el-icon><Connection /></el-icon></div>
        <div class="metric-info">
          <div class="metric-title">隧道总数</div>
          <div class="metric-value-wrap">
            <span class="metric-number">{{ store.stats.total_tunnels }}</span>
            <span class="metric-unit">条</span>
          </div>
        </div>
      </el-card>

      <!-- 运行中 / 保活 -->
      <el-card shadow="hover" class="metric-card">
        <div class="metric-icon running"><el-icon><VideoPlay /></el-icon></div>
        <div class="metric-info">
          <div class="metric-title">运行中 / 保活</div>
          <div class="metric-value-wrap">
            <span class="metric-number running-num">{{ store.stats.running_tunnels }}</span>
            <span class="metric-unit">活跃</span>
          </div>
        </div>
      </el-card>

      <!-- 当前活跃连接 -->
      <el-card shadow="hover" class="metric-card">
        <div class="metric-icon conn"><el-icon><Link /></el-icon></div>
        <div class="metric-info">
          <div class="metric-title">当前活跃连接</div>
          <div class="metric-value-wrap">
            <span class="metric-number">{{ store.stats.total_conns }}</span>
            <span class="metric-unit">conns</span>
          </div>
        </div>
      </el-card>

      <!-- 累计数据传输 -->
      <el-card shadow="hover" class="metric-card traffic-metric-card">
        <div class="metric-icon traffic"><el-icon><Sort /></el-icon></div>
        <div class="metric-info">
          <div class="metric-title">累计数据传输</div>
          <div class="traffic-dual-wrap">
            <div class="traffic-line">
              <span class="traffic-badge in">↓ 下行</span>
              <span class="traffic-val">{{ formatBytes(store.stats.total_bytes_in) }}</span>
            </div>
            <div class="traffic-line">
              <span class="traffic-badge out">↑ 上行</span>
              <span class="traffic-val">{{ formatBytes(store.stats.total_bytes_out) }}</span>
            </div>
          </div>
        </div>
      </el-card>
    </div>

    <!-- 活动隧道快速看板 (卡片流) -->
    <el-card shadow="never" class="section-card">
      <template #header>
        <div class="card-header">
          <div class="title-with-badge">
            <span class="main-title">活跃隧道实时动态</span>
            <el-tag size="small" type="success" effect="plain" v-if="activeTunnels.length > 0">
              {{ activeTunnels.length }} 个活动中
            </el-tag>
          </div>
          <div>
            <el-button type="primary" size="small" @click="$router.push('/tunnels')">
              前往隧道管理
            </el-button>
          </div>
        </div>
      </template>

      <div v-if="activeTunnels.length === 0" class="empty-state">
        <el-empty description="当前暂无运行中的隧道，请前往「隧道管理」开启或创建隧道">
          <el-button type="primary" size="small" @click="$router.push('/tunnels')">
            去开启隧道
          </el-button>
        </el-empty>
      </div>

      <div v-else class="active-tunnels-grid">
        <el-row :gutter="16" class="equal-height-row">
          <el-col
            v-for="tunnel in activeTunnels"
            :key="tunnel.id"
            :xs="24"
            :sm="24"
            :md="12"
            :lg="8"
            class="active-col"
          >
            <div class="active-tunnel-mini-card">
              <div class="mini-header">
                <div class="mini-title-wrap">
                  <el-tag
                    :type="tunnel.type === 'forward' ? 'primary' : 'warning'"
                    size="small"
                    effect="dark"
                  >
                    {{ tunnel.type === 'forward' ? '正向' : '反向' }}
                  </el-tag>
                  <strong class="mini-name" :title="tunnel.name">{{ tunnel.name }}</strong>
                </div>
                <div class="mini-status-wrap">
                  <el-tag
                    v-if="tunnel.runtime.health === 'unhealthy'"
                    size="small"
                    type="danger"
                    effect="light"
                    :title="tunnel.runtime.health_message"
                  >
                    异常
                  </el-tag>
                  <StatusBadge :status="tunnel.runtime.status" />
                </div>
              </div>

              <!-- 路由 -->
              <div class="mini-route-box">
                <span class="route-point">{{ tunnel.listen_host }}:{{ tunnel.listen_port }}</span>
                <el-icon class="route-arrow"><Right /></el-icon>
                <span class="route-point target">{{ tunnel.target_host }}:{{ tunnel.target_port }}</span>
              </div>

              <!-- 指标 -->
              <div class="mini-metrics-row">
                <div class="m-item">
                  <span class="lbl">连接:</span>
                  <strong>{{ tunnel.runtime.active_conns }}</strong>
                </div>
                <div class="m-item">
                  <span class="lbl">时长:</span>
                  <span>{{ formatDuration(tunnel.runtime.uptime) }}</span>
                </div>
                <div class="m-item">
                  <span class="lbl">流量:</span>
                  <span>↓{{ formatBytes(tunnel.runtime.bytes_in) }} / ↑{{ formatBytes(tunnel.runtime.bytes_out) }}</span>
                </div>
              </div>

              <!-- 操作 -->
              <div class="mini-actions">
                <el-button
                  type="danger"
                  size="small"
                  link
                  @click="handleStop(tunnel)"
                >
                  停止
                </el-button>
                <el-button
                  type="primary"
                  size="small"
                  link
                  @click="handleRestart(tunnel)"
                >
                  重启
                </el-button>
              </div>
            </div>
          </el-col>
        </el-row>
      </div>
    </el-card>

    <!-- 系统指南与架构速览 (等高排列) -->
    <el-row :gutter="16" class="info-row-grid">
      <el-col :md="12" :sm="24" class="info-col">
        <el-card shadow="never" class="info-card">
          <template #header>
            <div class="info-header">
              <el-icon class="info-title-icon"><Check /></el-icon>
              <span>快速使用指南</span>
            </div>
          </template>
          <ol class="guide-list">
            <li><strong>添加 SSH 主机</strong>：在「主机管理」中配置远程服务器 IP、端口、密码或私钥，支持保存前一键测试连通性。</li>
            <li><strong>创建正向隧道 (-L)</strong>：在 Hub 本地监听端口，外部请求将经加密 SSH 通道透传至远端可达的目标机器服务（包括远端局域网内其他服务）。</li>
            <li><strong>创建反向隧道 (-R)</strong>：在远端公网服务器监听端口，将外部请求穿透转发回本地或 Hub 所在网络的服务。</li>
            <li><strong>心跳保活与自愈</strong>：隧道内置 TCP KeepAlive 与 `keepalive@openssh.com` 定时探测，网络中断时通过指数退避自动重连。</li>
          </ol>
        </el-card>
      </el-col>
      <el-col :md="12" :sm="24" class="info-col">
        <el-card shadow="never" class="info-card">
          <template #header>
            <div class="info-header">
              <el-icon class="info-title-icon warning"><InfoFilled /></el-icon>
              <span>反向隧道配置注意事项</span>
            </div>
          </template>
          <div class="guide-note">
            <p>若配置反向隧道（<code>-R</code>）且需要被远程主机外部的其他客户端访问，请确认远程 SSH 服务器的 <code>/etc/ssh/sshd_config</code> 中包含以下配置：</p>
            <pre class="code-block">GatewayPorts yes</pre>
            <p>修改配置后请在远程主机执行 <code>sudo systemctl restart sshd</code> 生效。若未开启，远程 SSH 默认仅监听其本机的 <code>127.0.0.1</code>。</p>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import {
  Platform,
  Connection,
  VideoPlay,
  Link,
  Sort,
  Right,
  Check,
  InfoFilled,
} from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { useTunnelStore } from '../store/useTunnelStore'
import StatusBadge from '../components/StatusBadge.vue'
import { formatBytes, formatDuration } from '../utils/format'
import { tunnelApi } from '../api'
import type { TunnelVO } from '../types'

const store = useTunnelStore()

const activeTunnels = computed(() => {
  return store.tunnels.filter(
    (t) => t.runtime.status === 'running' || t.runtime.status === 'reconnecting'
  )
})

async function handleStop(tunnel: TunnelVO) {
  try {
    await tunnelApi.stop(tunnel.id)
    ElMessage.success(`隧道 [${tunnel.name}] 已停止`)
    store.fetchTunnels()
    store.fetchStats()
  } catch (err: any) {
    ElMessage.error(err.message || '停止失败')
  }
}

async function handleRestart(tunnel: TunnelVO) {
  try {
    await tunnelApi.restart(tunnel.id)
    ElMessage.success(`隧道 [${tunnel.name}] 已发送重启指令`)
    store.fetchTunnels()
    store.fetchStats()
  } catch (err: any) {
    ElMessage.error(err.message || '重启失败')
  }
}

onMounted(() => {
  store.fetchStats()
  store.fetchTunnels()
  store.fetchHosts()
})
</script>

<style scoped>
.dashboard-container {
  padding: 4px;
}

/* 5 个指标卡片网格布局，严格等宽等高 */
.metrics-grid-container {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 16px;
  margin-bottom: 20px;
}

@media (max-width: 1400px) {
  .metrics-grid-container {
    grid-template-columns: repeat(3, 1fr);
  }
}

@media (max-width: 900px) {
  .metrics-grid-container {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 540px) {
  .metrics-grid-container {
    grid-template-columns: 1fr;
  }
}

.metric-card {
  border-radius: 10px;
  border: 1px solid #e5e7eb;
  min-height: 98px;
  height: 100%;
  box-sizing: border-box;
  display: flex;
}

:deep(.metric-card .el-card__body) {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 16px 18px;
  width: 100%;
  height: 100%;
  box-sizing: border-box;
}

.metric-icon {
  width: 48px;
  height: 48px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  flex-shrink: 0;
}

.metric-icon.host {
  background: rgba(64, 158, 255, 0.12);
  color: #409eff;
}

.metric-icon.tunnel {
  background: rgba(103, 194, 58, 0.12);
  color: #67c23a;
}

.metric-icon.running {
  background: rgba(230, 162, 60, 0.12);
  color: #e6a23c;
}

.metric-icon.conn {
  background: rgba(144, 147, 153, 0.12);
  color: #909399;
}

.metric-icon.traffic {
  background: rgba(155, 89, 182, 0.12);
  color: #9b59b6;
}

.metric-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  justify-content: center;
}

.metric-title {
  font-size: 13px;
  color: #64748b;
  margin-bottom: 4px;
  font-weight: 500;
  white-space: nowrap;
}

.metric-value-wrap {
  display: flex;
  align-items: baseline;
  gap: 6px;
  line-height: 1.2;
}

.metric-number {
  font-size: 24px;
  font-weight: 700;
  color: #1e293b;
  line-height: 1;
}

.metric-number.running-num {
  color: #16a34a;
}

.metric-unit {
  font-size: 12px;
  color: #94a3b8;
  font-weight: 500;
}

/* 流量卡片专属微型双行展示 */
.traffic-dual-wrap {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.traffic-line {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  line-height: 1.2;
}

.traffic-badge {
  font-size: 11px;
  font-weight: 600;
  border-radius: 3px;
  padding: 1px 4px;
}

.traffic-badge.in {
  background: #eff6ff;
  color: #2563eb;
}

.traffic-badge.out {
  background: #fdf4ff;
  color: #c026d3;
}

.traffic-val {
  font-family: monospace;
  font-weight: 600;
  color: #334155;
  font-size: 12px;
}

.section-card {
  border-radius: 10px;
  border: 1px solid #e5e7eb;
  margin-bottom: 20px;
}

:deep(.section-card .el-card__header) {
  padding: 16px 20px;
  border-bottom: 1px solid #f1f5f9;
}

:deep(.section-card .el-card__body) {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.title-with-badge {
  display: flex;
  align-items: center;
  gap: 10px;
}

.main-title {
  font-weight: 700;
  font-size: 16px;
  color: #1f2937;
}

.empty-state {
  padding: 20px 0;
}

/* 活跃隧道卡片流 */
.active-tunnels-grid {
  width: 100%;
}

.equal-height-row {
  display: flex;
  flex-wrap: wrap;
}

.active-col {
  margin-bottom: 16px;
  display: flex;
}

.active-tunnel-mini-card {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 14px 16px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  width: 100%;
  height: 100%;
  box-sizing: border-box;
  transition: all 0.2s;
  justify-content: space-between;
}

.active-tunnel-mini-card:hover {
  border-color: #cbd5e1;
  background: #f1f5f9;
}

.mini-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.mini-title-wrap {
  display: flex;
  align-items: center;
  gap: 8px;
  overflow: hidden;
}

.mini-status-wrap {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}

.mini-name {
  font-size: 14px;
  color: #1e293b;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mini-route-box {
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  align-items: center;
  gap: 8px;
  font-family: monospace;
  font-size: 12px;
  background: #f1f5f9;
  border-radius: 6px;
  padding: 6px 10px;
}

.route-point {
  background: #ffffff;
  border: 1px solid #e2e8f0;
  padding: 3px 6px;
  border-radius: 4px;
  text-align: center;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.route-point.target {
  background: #eff6ff;
  border-color: #bfdbfe;
  color: #2563eb;
  font-weight: 600;
}

.route-arrow {
  color: #3b82f6;
  font-size: 13px;
  display: flex;
  justify-content: center;
}

.mini-metrics-row {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 12px;
  color: #64748b;
  flex-wrap: wrap;
}

.m-item .lbl {
  color: #94a3b8;
  margin-right: 2px;
}

.mini-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  border-top: 1px dashed #e2e8f0;
  padding-top: 8px;
}

/* 底部提示卡片等高布局 */
.info-row-grid {
  display: flex;
  flex-wrap: wrap;
}

.info-col {
  display: flex;
  margin-bottom: 16px;
}

.info-card {
  border-radius: 10px;
  border: 1px solid #e5e7eb;
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
}

:deep(.info-card .el-card__header) {
  padding: 16px 20px;
  border-bottom: 1px solid #f1f5f9;
}

:deep(.info-card .el-card__body) {
  padding: 20px 24px;
  flex: 1;
}

.info-header {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 700;
  font-size: 15px;
  color: #1f2937;
}

.info-title-icon {
  color: #16a34a;
  font-size: 18px;
}

.info-title-icon.warning {
  color: #ea580c;
}

.guide-list {
  padding-left: 20px;
  line-height: 2;
  color: #475569;
  font-size: 13px;
  margin: 0;
}

.guide-note {
  font-size: 13px;
  color: #475569;
  line-height: 1.7;
}

.code-block {
  background: #1e293b;
  color: #38bdf8;
  padding: 10px 14px;
  border-radius: 6px;
  font-family: monospace;
  font-size: 13px;
  margin: 10px 0;
}
</style>
