<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { clientRuntime, nativeApp, publicOrigin } from './runtime'

const visible = ref(false)
const server = ref(publicOrigin())
const testing = ref(false)
const saving = ref(false)
const connectionStatus = ref('')
const warning = ref(window.__OPC_CLIENT_STORAGE_WARNING__ || '')
const show = () => { server.value = publicOrigin(); visible.value = true }
const credentialWarning = (e: Event) => { warning.value = (e as CustomEvent).detail }
const downloadError = (e: Event) => { MessagePlugin.error((e as CustomEvent).detail) }

async function testConnection() {
  testing.value = true
  connectionStatus.value = '检测中…'
  try {
    // Test the ACTIVE connection, never a free-form arbitrary target through the relay.
    const response = await fetch('/api/v1/auth/config', { signal: AbortSignal.timeout(15000) })
    if (!response.ok) throw new Error(`HTTP ${response.status}`)
    const data = await response.json()
    if (!data.success) throw new Error(data.message || '服务端返回了无效响应')
    connectionStatus.value = '当前服务端连接正常'
  } catch (error: any) { connectionStatus.value = `连接失败：${error.message || error}` }
  finally { testing.value = false }
}

async function save() {
  if (!nativeApp()) { MessagePlugin.info('开发预览请在启动命令中设置服务端地址'); return }
  saving.value = true
  try {
    await nativeApp()!.SetServerURL(server.value.trim())
    MessagePlugin.success('连接配置已保存，退出并重新打开客户端后生效')
    visible.value = false
  } catch (error: any) { MessagePlugin.error(String(error.message || error)) }
  finally { saving.value = false }
}

onMounted(() => {
  window.addEventListener('opc:connection-settings', show)
  window.addEventListener('opc:credential-warning', credentialWarning)
  window.addEventListener('opc:download-error', downloadError)
})
onUnmounted(() => {
  window.removeEventListener('opc:connection-settings', show)
  window.removeEventListener('opc:credential-warning', credentialWarning)
  window.removeEventListener('opc:download-error', downloadError)
})
</script>

<template>
  <button v-if="clientRuntime()" class="opc-connection-button" type="button" @click="show">服务端连接</button>
  <t-dialog v-model:visible="visible" header="OPC Client 服务端连接" :footer="false" width="560px">
    <p>当前连接：{{ publicOrigin() }}</p>
    <t-input v-model="server" placeholder="https://admin.tokendance.sh" aria-label="服务端地址" />
    <p class="opc-connection-note">修改地址将在重新打开客户端后生效。账号、知识库和对话保存在所连接的服务端；不同服务端的登录状态分别保存。</p>
    <p>此版本支持账号密码登录；企业微信登录将在后续版本接入。</p>
    <p v-if="warning" role="alert" class="opc-connection-warning">{{ warning }}</p>
    <p v-if="connectionStatus" role="status">{{ connectionStatus }}</p>
    <div class="opc-connection-actions">
      <t-button variant="outline" :loading="testing" @click="testConnection">检测当前连接</t-button>
      <t-button :loading="saving" @click="save">保存配置</t-button>
    </div>
  </t-dialog>
</template>

<style scoped>
.opc-connection-button { position: fixed; right: 16px; bottom: 12px; z-index: 500; border: 1px solid var(--td-component-border); border-radius: 6px; background: var(--td-bg-color-container); color: var(--td-text-color-secondary); padding: 6px 10px; cursor: pointer; font-size: 12px; }
.opc-connection-note { color: var(--td-text-color-secondary); line-height: 1.6; }
.opc-connection-warning { color: var(--td-error-color); }
.opc-connection-actions { display: flex; gap: 12px; justify-content: flex-end; margin-top: 20px; }
</style>
