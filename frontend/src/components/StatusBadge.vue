<template>
  <el-tag :type="tagType" :effect="effect" class="status-badge" round>
    <span class="status-dot" :class="status"></span>
    <span class="status-text">{{ statusText }}</span>
  </el-tag>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    status: 'stopped' | 'starting' | 'running' | 'reconnecting' | 'error'
    effect?: 'light' | 'dark' | 'plain'
  }>(),
  {
    effect: 'light',
  }
)

const tagType = computed(() => {
  switch (props.status) {
    case 'running':
      return 'success'
    case 'starting':
      return 'primary'
    case 'reconnecting':
      return 'warning'
    case 'error':
      return 'danger'
    default:
      return 'info'
  }
})

const statusText = computed(() => {
  switch (props.status) {
    case 'running':
      return '运行中'
    case 'starting':
      return '启动中'
    case 'reconnecting':
      return '重连中'
    case 'error':
      return '异常'
    default:
      return '已停止'
  }
})
</script>

<style scoped>
.status-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-weight: 500;
  padding: 0 10px;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  display: inline-block;
}

.status-dot.running {
  background-color: #67c23a;
  box-shadow: 0 0 6px #67c23a;
  animation: pulse 2s infinite;
}

.status-dot.reconnecting {
  background-color: #e6a23c;
  box-shadow: 0 0 6px #e6a23c;
  animation: pulse 1.2s infinite;
}

.status-dot.starting {
  background-color: #409eff;
}

.status-dot.error {
  background-color: #f56c6c;
}

.status-dot.stopped {
  background-color: #909399;
}

@keyframes pulse {
  0% {
    transform: scale(0.95);
    opacity: 0.8;
  }
  50% {
    transform: scale(1.2);
    opacity: 1;
  }
  100% {
    transform: scale(0.95);
    opacity: 0.8;
  }
}
</style>
