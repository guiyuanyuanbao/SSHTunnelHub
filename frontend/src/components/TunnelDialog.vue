<template>
  <el-dialog
    v-model="visible"
    :title="isEdit ? '编辑 SSH 隧道' : '创建新 SSH 隧道'"
    width="640px"
    destroy-on-close
    :close-on-click-modal="false"
  >
    <el-form ref="formRef" :model="form" :rules="rules" label-width="110px">
      <el-form-item label="隧道名称" prop="name">
        <el-input v-model="form.name" placeholder="例如: 生产MySQL转发 / 本地Web穿透" />
      </el-form-item>

      <el-form-item label="关联主机" prop="host_id">
        <el-select v-model="form.host_id" placeholder="请选择 SSH 目标主机" style="width: 100%">
          <el-option
            v-for="h in hosts"
            :key="h.id"
            :label="`${h.name} (${h.username}@${h.host}:${h.port})`"
            :value="h.id"
          />
        </el-select>
      </el-form-item>

      <el-form-item label="隧道类型" prop="type">
        <el-radio-group v-model="form.type" @change="handleTypeChange">
          <el-radio-button value="forward">
            正向隧道 (-L 本地转发)
          </el-radio-button>
          <el-radio-button value="reverse">
            反向隧道 (-R 远程转发)
          </el-radio-button>
        </el-radio-group>
      </el-form-item>

      <!-- 模式图解提示 -->
      <el-alert
        v-if="form.type === 'forward'"
        type="info"
        :closable="false"
        style="margin-bottom: 18px"
      >
        <template #title>
          <div style="font-size: 13px; line-height: 1.6">
            <strong>正向隧道 (Local Forwarding)</strong>：<br />
            在 <b>Hub 本地机器</b> 监听端口。访问 Hub 本地端口的流量将通过 SSH 加密通道，转发到 <b>远端主机可达的目标服务</b>（可为远程机器自身的 127.0.0.1，或远端局域网内任意 IP）。
          </div>
        </template>
      </el-alert>

      <el-alert
        v-else
        type="warning"
        :closable="false"
        style="margin-bottom: 18px"
      >
        <template #title>
          <div style="font-size: 13px; line-height: 1.6">
            <strong>反向隧道 (Remote Forwarding)</strong>：<br />
            在 <b>远程 SSH 服务器</b> 监听端口。访问远程端口的流量将回传至 Hub，并转发到 <b>Hub 可达的目标服务</b>（例如本机 127.0.0.1 或内网服务）。<br />
            <small style="color: #e6a23c">💡 提示：若希望远程端口允许外部公网连接，远程 SSH 主机需在 sshd_config 开启 GatewayPorts yes</small>
          </div>
        </template>
      </el-alert>

      <!-- 监听端 -->
      <el-card shadow="never" style="margin-bottom: 16px; background-color: #fafafa">
        <template #header>
          <span style="font-weight: 600; font-size: 14px">
            {{ form.type === 'forward' ? '1. Hub 本地监听端配置' : '1. 远程 SSH 监听端配置' }}
          </span>
        </template>
        <el-row :gutter="12">
          <el-col :span="14">
            <el-form-item label="监听地址" prop="listen_host">
              <el-input v-model="form.listen_host" placeholder="127.0.0.1 或 0.0.0.0" />
            </el-form-item>
          </el-col>
          <el-col :span="10">
            <el-form-item label="监听端口" prop="listen_port">
              <el-input-number v-model="form.listen_port" :min="1" :max="65535" style="width: 100%" />
            </el-form-item>
          </el-col>
        </el-row>
      </el-card>

      <!-- 目标端 -->
      <el-card shadow="never" style="margin-bottom: 16px; background-color: #fafafa">
        <template #header>
          <span style="font-weight: 600; font-size: 14px">
            {{ form.type === 'forward' ? '2. 远程目标服务地址' : '2. 本地/内网目标服务地址' }}
          </span>
        </template>
        <el-row :gutter="12">
          <el-col :span="14">
            <el-form-item label="目标地址" prop="target_host">
              <el-input v-model="form.target_host" placeholder="127.0.0.1 或内网目标IP" />
            </el-form-item>
          </el-col>
          <el-col :span="10">
            <el-form-item label="目标端口" prop="target_port">
              <el-input-number v-model="form.target_port" :min="1" :max="65535" style="width: 100%" />
            </el-form-item>
          </el-col>
        </el-row>
      </el-card>

      <el-row :gutter="12">
        <el-col :span="12">
          <el-form-item label="自启动" prop="auto_start">
            <el-switch v-model="form.auto_start" active-text="系统启动时自动建立隧道" />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="备注说明" prop="remark">
            <el-input v-model="form.remark" placeholder="备注用途 (可选)" />
          </el-form-item>
        </el-col>
      </el-row>
    </el-form>

    <template #footer>
      <div style="display: flex; justify-content: flex-end; gap: 10px">
        <el-button @click="visible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">
          {{ isEdit ? '保存更新' : '创建隧道' }}
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage } from 'element-plus'
import type { HostVO, TunnelVO } from '../types'
import { tunnelApi } from '../api'

const props = defineProps<{
  hosts: HostVO[]
}>()

const emit = defineEmits<{
  (e: 'success'): void
}>()

const visible = ref(false)
const isEdit = ref(false)
const currentTunnel = ref<TunnelVO | null>(null)
const formRef = ref<FormInstance>()
const submitting = ref(false)

const form = reactive({
  name: '',
  host_id: undefined as number | undefined,
  type: 'forward' as 'forward' | 'reverse',
  listen_host: '127.0.0.1',
  listen_port: 8080,
  target_host: '127.0.0.1',
  target_port: 3306,
  auto_start: false,
  remark: '',
})

const rules: FormRules = {
  name: [{ required: true, message: '请输入隧道名称', trigger: 'blur' }],
  host_id: [{ required: true, message: '请选择关联主机', trigger: 'change' }],
  type: [{ required: true, message: '请选择隧道类型', trigger: 'change' }],
  listen_host: [{ required: true, message: '请输入监听地址', trigger: 'blur' }],
  listen_port: [{ required: true, message: '请输入监听端口', trigger: 'blur' }],
  target_host: [{ required: true, message: '请输入目标地址', trigger: 'blur' }],
  target_port: [{ required: true, message: '请输入目标端口', trigger: 'blur' }],
}

function handleTypeChange() {
  if (form.type === 'forward') {
    if (!form.listen_host) form.listen_host = '127.0.0.1'
  } else {
    if (!form.listen_host) form.listen_host = '127.0.0.1'
  }
}

function open(tunnel?: TunnelVO) {
  if (tunnel) {
    isEdit.value = true
    currentTunnel.value = tunnel
    form.name = tunnel.name
    form.host_id = tunnel.host_id
    form.type = tunnel.type
    form.listen_host = tunnel.listen_host
    form.listen_port = tunnel.listen_port
    form.target_host = tunnel.target_host
    form.target_port = tunnel.target_port
    form.auto_start = tunnel.auto_start
    form.remark = tunnel.remark || ''
  } else {
    isEdit.value = false
    currentTunnel.value = null
    form.name = ''
    form.host_id = props.hosts.length > 0 ? props.hosts[0].id : undefined
    form.type = 'forward'
    form.listen_host = '127.0.0.1'
    form.listen_port = 8080
    form.target_host = '127.0.0.1'
    form.target_port = 3306
    form.auto_start = false
    form.remark = ''
  }
  visible.value = true
}

async function handleSubmit() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid || !form.host_id) return
    submitting.value = true
    try {
      if (isEdit.value && currentTunnel.value) {
        await tunnelApi.update(currentTunnel.value.id, {
          name: form.name,
          host_id: form.host_id,
          type: form.type,
          listen_host: form.listen_host,
          listen_port: form.listen_port,
          target_host: form.target_host,
          target_port: form.target_port,
          auto_start: form.auto_start,
          remark: form.remark,
        })
        ElMessage.success('隧道更新成功')
      } else {
        await tunnelApi.create({
          name: form.name,
          host_id: form.host_id,
          type: form.type,
          listen_host: form.listen_host,
          listen_port: form.listen_port,
          target_host: form.target_host,
          target_port: form.target_port,
          auto_start: form.auto_start,
          remark: form.remark,
        })
        ElMessage.success('隧道创建成功')
      }
      visible.value = false
      emit('success')
    } catch (err: any) {
      ElMessage.error(err.message || '操作失败')
    } finally {
      submitting.value = false
    }
  })
}

defineExpose({ open })
</script>
