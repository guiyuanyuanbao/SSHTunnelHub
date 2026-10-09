<template>
  <el-dialog
    v-model="visible"
    :title="isEdit ? '编辑 SSH 主机' : '添加 SSH 主机'"
    width="580px"
    destroy-on-close
    :close-on-click-modal="false"
  >
    <el-form ref="formRef" :model="form" :rules="rules" label-width="110px">
      <el-form-item label="主机名称" prop="name">
        <el-input v-model="form.name" placeholder="例如: 生产网关 / 测试服务器" />
      </el-form-item>

      <el-row :gutter="12">
        <el-col :span="16">
          <el-form-item label="主机地址" prop="host">
            <el-input v-model="form.host" placeholder="IP 或 域名 (如: 192.168.1.100)" />
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="端口" prop="port" label-width="50px">
            <el-input-number v-model="form.port" :min="1" :max="65535" style="width: 100%" />
          </el-form-item>
        </el-col>
      </el-row>

      <el-form-item label="登录用户" prop="username">
        <el-input v-model="form.username" placeholder="如: root / ubuntu" />
      </el-form-item>

      <el-form-item label="认证方式" prop="auth_type">
        <el-radio-group v-model="form.auth_type">
          <el-radio-button value="password">密码认证</el-radio-button>
          <el-radio-button value="private_key">私钥认证</el-radio-button>
        </el-radio-group>
      </el-form-item>

      <!-- 密码认证 -->
      <template v-if="form.auth_type === 'password'">
        <el-form-item
          label="SSH 密码"
          prop="password"
          :rules="[
            { required: !isEdit, message: '请输入 SSH 密码', trigger: 'blur' },
          ]"
        >
          <el-input
            v-model="form.password"
            type="password"
            show-password
            :placeholder="isEdit && currentHost?.has_password ? '留空则保持原密码不变' : '请输入登录密码'"
          />
        </el-form-item>
      </template>

      <!-- 私钥认证 -->
      <template v-if="form.auth_type === 'private_key'">
        <el-form-item
          label="SSH 私钥"
          prop="private_key"
          :rules="[
            { required: !isEdit && !currentHost?.has_private_key, message: '请输入或上传私钥', trigger: 'blur' },
          ]"
        >
          <div style="width: 100%">
            <el-input
              v-model="form.private_key"
              type="textarea"
              :rows="5"
              font-family="monospace"
              :placeholder="isEdit && currentHost?.has_private_key ? '留空则保持原私钥不变。如需更新请输入 PEM 私钥...' : '-----BEGIN OPENSSH PRIVATE KEY----- 或 RSA 私钥内容'"
            />
            <div style="margin-top: 6px; display: flex; justify-content: flex-end">
              <el-upload
                :auto-upload="false"
                :show-file-list="false"
                :on-change="handleKeyFileUpload"
                accept=".pem,.key,.pub,id_rsa,id_ed25519"
              >
                <el-button size="small" type="primary" plain>从文件导入私钥</el-button>
              </el-upload>
            </div>
          </div>
        </el-form-item>

        <el-form-item label="私钥密码" prop="passphrase">
          <el-input
            v-model="form.passphrase"
            type="password"
            show-password
            :placeholder="isEdit && currentHost?.has_passphrase ? '留空则保持原 Passphrase 不变' : '若私钥无密码保护可留空'"
          />
        </el-form-item>
      </template>

      <el-form-item label="备注说明" prop="remark">
        <el-input v-model="form.remark" placeholder="环境或用途备注 (可选)" />
      </el-form-item>

      <!-- 连通性测试结果提示条 -->
      <el-alert
        v-if="testResult"
        :title="testResult.success ? `连接成功！延迟: ${testResult.latency}ms (${testResult.banner || 'SSH-2.0'})` : `连接失败: ${testResult.error}`"
        :type="testResult.success ? 'success' : 'error'"
        :closable="false"
        show-icon
        style="margin-bottom: 12px"
      />
    </el-form>

    <template #footer>
      <div style="display: flex; justify-content: space-between; align-items: center">
        <el-button :loading="testing" @click="handleTestConnection">
          测试连通性
        </el-button>
        <div>
          <el-button @click="visible = false">取消</el-button>
          <el-button type="primary" :loading="submitting" @click="handleSubmit">
            保存
          </el-button>
        </div>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, reactive, computed } from 'vue'
import type { FormInstance, FormRules, UploadFile } from 'element-plus'
import { ElMessage } from 'element-plus'
import type { HostVO } from '../types'
import { hostApi } from '../api'

const emit = defineEmits<{
  (e: 'success'): void
}>()

const visible = ref(false)
const isEdit = ref(false)
const currentHost = ref<HostVO | null>(null)
const formRef = ref<FormInstance>()
const submitting = ref(false)
const testing = ref(false)
const testResult = ref<{ success: boolean; latency?: number; banner?: string; error?: string } | null>(null)

const form = reactive({
  name: '',
  host: '',
  port: 22,
  username: 'root',
  auth_type: 'password' as 'password' | 'private_key',
  password: '',
  private_key: '',
  passphrase: '',
  remark: '',
})

const rules: FormRules = {
  name: [{ required: true, message: '请输入主机名称', trigger: 'blur' }],
  host: [{ required: true, message: '请输入主机地址', trigger: 'blur' }],
  port: [{ required: true, message: '请输入端口', trigger: 'blur' }],
  username: [{ required: true, message: '请输入登录用户', trigger: 'blur' }],
  auth_type: [{ required: true, message: '请选择认证方式', trigger: 'change' }],
}

function open(host?: HostVO) {
  testResult.value = null
  if (host) {
    isEdit.value = true
    currentHost.value = host
    form.name = host.name
    form.host = host.host
    form.port = host.port
    form.username = host.username
    form.auth_type = host.auth_type
    form.password = ''
    form.private_key = ''
    form.passphrase = ''
    form.remark = host.remark || ''
  } else {
    isEdit.value = false
    currentHost.value = null
    form.name = ''
    form.host = ''
    form.port = 22
    form.username = 'root'
    form.auth_type = 'password'
    form.password = ''
    form.private_key = ''
    form.passphrase = ''
    form.remark = ''
  }
  visible.value = true
}

function handleKeyFileUpload(file: UploadFile) {
  if (!file.raw) return
  const reader = new FileReader()
  reader.onload = (e) => {
    form.private_key = (e.target?.result as string) || ''
    ElMessage.success('私钥文件读取成功')
  }
  reader.readAsText(file.raw)
}

async function handleTestConnection() {
  if (!form.host || !form.username) {
    ElMessage.warning('请先填写主机地址和登录用户名')
    return
  }

  testing.value = true
  testResult.value = null
  try {
    if (isEdit.value && !form.password && !form.private_key && currentHost.value) {
      // Test using saved credentials in backend
      const res = await hostApi.testSaved(currentHost.value.id)
      testResult.value = res
    } else {
      // Test using input credentials
      const res = await hostApi.testRaw({
        host: form.host,
        port: form.port,
        username: form.username,
        auth_type: form.auth_type,
        password: form.password,
        private_key: form.private_key,
        passphrase: form.passphrase,
      })
      testResult.value = res
    }
  } catch (err: any) {
    testResult.value = {
      success: false,
      error: err.message,
    }
  } finally {
    testing.value = false
  }
}

async function handleSubmit() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    submitting.value = true
    try {
      if (isEdit.value && currentHost.value) {
        await hostApi.update(currentHost.value.id, {
          name: form.name,
          host: form.host,
          port: form.port,
          username: form.username,
          auth_type: form.auth_type,
          password: form.password,
          private_key: form.private_key,
          passphrase: form.passphrase,
          remark: form.remark,
        })
        ElMessage.success('主机信息更新成功')
      } else {
        await hostApi.create({
          name: form.name,
          host: form.host,
          port: form.port,
          username: form.username,
          auth_type: form.auth_type,
          password: form.password,
          private_key: form.private_key,
          passphrase: form.passphrase,
          remark: form.remark,
        })
        ElMessage.success('主机添加成功')
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
