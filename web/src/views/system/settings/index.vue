<template>
  <div class="system-settings">
    <div class="page-header">
      <h2>系统设置</h2>
      <p>管理系统基本配置信息</p>
    </div>

    <div class="settings-container">
      <!-- 基本信息设置 -->
      <n-card title="基本信息" class="settings-card">
        <template #header-extra>
          <span class="card-description">配置系统的基本信息</span>
        </template>
        <n-form
          ref="basicFormRef"
          :model="basicForm"
          :rules="basicRules"
          label-placement="left"
          :label-width="120"
          @submit.prevent="handleBasicSubmit"
        >
          <n-form-item label="系统名称" path="systemName">
            <n-input
              v-model:value="basicForm.systemName"
              placeholder="请输入系统名称"
              :maxlength="50"
              show-count
            />
          </n-form-item>
          <n-form-item label="系统版本" path="systemVersion">
            <n-input
              v-model:value="basicForm.systemVersion"
              placeholder="请输入系统版本"
              :maxlength="20"
              show-count
            />
          </n-form-item>
          <n-form-item label="系统描述" path="systemDescription">
            <n-input
              v-model:value="basicForm.systemDescription"
              type="textarea"
              placeholder="请输入系统描述"
              :rows="3"
              :maxlength="200"
              show-count
            />
          </n-form-item>
          <n-form-item label="系统Logo" path="systemLogo">
            <!-- 调试信息 -->
            <div style="margin-bottom: 10px; padding: 10px; background: #f5f5f5; border-radius: 4px; font-size: 12px; color: #666;">
              调试信息: basicForm.systemLogo = {{ basicForm.systemLogo ? '有值 (长度: ' + basicForm.systemLogo.length + ')' : '无值' }}
            </div>
            <div class="logo-container">
              <div class="logo-preview" v-if="basicForm.systemLogo">
                <img :src="basicForm.systemLogo" alt="系统Logo" class="logo-image" />
                <div class="logo-actions">
                  <n-button size="small" @click="handleLogoPreview">预览</n-button>
                  <n-button size="small" type="error" @click="handleLogoRemove">删除</n-button>
                </div>
              </div>
              <div class="logo-upload" v-else>
                <n-upload
                  ref="logoUploadRef"
                  :max="1"
                  :file-list="logoFileList"
                  accept="image/*"
                  @change="handleLogoChange"
                  @before-upload="beforeLogoUpload"
                  @remove="handleLogoRemove"
                >
                  <n-upload-dragger>
                    <div style="margin-bottom: 12px">
                      <n-icon size="48" :depth="3">
                        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">
                          <path fill="currentColor" d="M14,2H6A2,2 0 0,0 4,4V20A2,2 0 0,0 6,22H18A2,2 0 0,0 20,20V8L14,2M18,20H6V4H13V9H18V20Z" />
                        </svg>
                      </n-icon>
                    </div>
                    <n-text style="font-size: 16px">点击或者拖动文件到该区域来上传</n-text>
                    <n-p depth="3" style="margin: 8px 0 0 0">
                      支持 JPG、PNG、GIF 格式，文件大小不超过 2MB
                    </n-p>
                  </n-upload-dragger>
                </n-upload>
              </div>
            </div>
          </n-form-item>
          <n-form-item>
            <n-button type="primary" :loading="basicLoading" @click="handleBasicSubmit">
              保存基本信息
            </n-button>
          </n-form-item>
        </n-form>
      </n-card>

      <!-- 系统配置 -->
      <n-card title="系统配置" class="settings-card">
        <template #header-extra>
          <span class="card-description">管理系统运行参数</span>
        </template>
        <n-form
          ref="configFormRef"
          :model="configForm"
          :rules="configRules"
          label-placement="left"
          :label-width="120"
          @submit.prevent="handleConfigSubmit"
        >
          <n-form-item label="维护模式" path="maintenanceMode">
            <n-switch
              v-model:value="configForm.maintenanceMode"
              checked-value="true"
              unchecked-value="false"
            >
              <template #checked>开启</template>
              <template #unchecked>关闭</template>
            </n-switch>
            <div class="form-help">开启后，系统将进入维护模式，普通用户无法访问</div>
          </n-form-item>
          <n-form-item label="会话超时" path="sessionTimeout">
            <n-input-number
              v-model:value="configForm.sessionTimeout"
              :min="300"
              :max="86400"
              :step="60"
              style="width: 200px"
            >
              <template #suffix>秒</template>
            </n-input-number>
            <div class="form-help">用户会话超时时间，范围：300-86400秒</div>
          </n-form-item>
          <n-form-item label="最大上传大小" path="maxUploadSize">
            <n-input-number
              v-model:value="configForm.maxUploadSize"
              :min="1"
              :max="100"
              :step="1"
              style="width: 200px"
            >
              <template #suffix>MB</template>
            </n-input-number>
            <div class="form-help">单个文件最大上传大小，范围：1-100MB</div>
          </n-form-item>
          <n-form-item>
            <n-button type="primary" :loading="configLoading" @click="handleConfigSubmit">
              保存系统配置
            </n-button>
          </n-form-item>
        </n-form>
      </n-card>

      <!-- 系统信息 -->
      <n-card title="系统信息" class="settings-card">
        <template #header-extra>
          <span class="card-description">查看当前系统运行状态</span>
        </template>
        <div class="info-grid">
          <div class="info-item">
            <span class="label">当前版本：</span>
            <span class="value">{{ systemInfo.version }}</span>
          </div>
          <div class="info-item">
            <span class="label">运行时间：</span>
            <span class="value">{{ systemInfo.uptime }}</span>
          </div>
          <div class="info-item">
            <span class="label">系统状态：</span>
            <n-tag :type="getStatusType(systemInfo.status)" size="small">
              {{ systemInfo.statusText }}
            </n-tag>
          </div>
          <div class="info-item">
            <span class="label">最后更新：</span>
            <span class="value">{{ systemInfo.lastUpdate }}</span>
          </div>
        </div>
      </n-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import { useMessage, NCard, NForm, NFormItem, NInput, NInputNumber, NSwitch, NButton, NTag } from 'naive-ui'
import type { FormInst, FormRules } from 'naive-ui'
import { systemConfigApi } from '@/api/system'

const message = useMessage()

// 基本信息表单
const basicForm = reactive({
  systemName: '',
  systemVersion: '',
  systemDescription: '',
  systemLogo: ''
})

// 系统配置表单
const configForm = reactive({
  maintenanceMode: false,
  sessionTimeout: 3600,
  maxUploadSize: 10
})

// 系统信息
const systemInfo = reactive({
  version: '1.0.0',
  uptime: '0天0小时0分钟',
  status: 'healthy',
  statusText: '正常运行',
  lastUpdate: ''
})

const basicLoading = ref(false)
  const configLoading = ref(false)
  const logoUploadRef = ref(null)
  const logoFileList = ref([])
  const logoPreviewVisible = ref(false)
const basicFormRef = ref<FormInst | null>(null)
const configFormRef = ref<FormInst | null>(null)

// 表单验证规则
const basicRules: FormRules = {
  systemName: [
    { required: true, message: '请输入系统名称', trigger: 'blur' },
    { min: 2, max: 50, message: '系统名称长度应在2-50个字符之间', trigger: 'blur' }
  ],
  systemVersion: [
    { required: true, message: '请输入系统版本', trigger: 'blur' },
    { max: 20, message: '系统版本长度不能超过20个字符', trigger: 'blur' }
  ]
}

const configRules: FormRules = {
  sessionTimeout: [
    { required: true, message: '请输入会话超时时间', trigger: 'blur' },
    { type: 'number', min: 300, max: 86400, message: '会话超时时间范围：300-86400秒', trigger: 'blur' }
  ],
  maxUploadSize: [
    { required: true, message: '请输入最大上传大小', trigger: 'blur' },
    { type: 'number', min: 1, max: 100, message: '最大上传大小范围：1-100MB', trigger: 'blur' }
  ]
}

// 获取状态类型
const getStatusType = (status: string) => {
  switch (status) {
    case 'healthy':
      return 'success'
    case 'warning':
      return 'warning'
    case 'error':
      return 'error'
    default:
      return 'default'
  }
}

// 加载系统配置
const loadSystemConfig = async () => {
  try {
    const response = await systemConfigApi.getSystemInfo()
    console.log(response)
    // const resp = response.data
    console.log("ssssssfff",response)
    if (response.data) {
      console.log('加载系统配置成功:', response.data)
      const configs = response.data.configs || {}
      
      // 填充基本信息
      basicForm.systemName = configs.system_name || ''
      basicForm.systemVersion = configs.system_version || ''
      basicForm.systemDescription = configs.system_description || ''
      basicForm.systemLogo = configs.system_logo || ''
      
      // 填充系统配置
      configForm.maintenanceMode = configs.maintenance_mode === 'true'
      configForm.sessionTimeout = parseInt(configs.session_timeout) || 3600
      configForm.maxUploadSize = Math.round((parseInt(configs.max_upload_size) || 10485760) / 1024 / 1024)
      
      // 填充系统信息
      systemInfo.version = configs.system_version || '1.0.0'
      systemInfo.lastUpdate = new Date().toLocaleString()
      
      // 填充真实的系统运行时间信息
      const systemInfoData = response.data.system_info || {}
      systemInfo.uptime = systemInfoData.uptime || '0天0小时0分钟'
    }
  } catch (error) {
    console.error('加载系统配置失败:', error)
    message.error('加载系统配置失败')
  }
}

// 保存基本信息
const handleBasicSubmit = async () => {
  if (!basicFormRef.value) return
  
  try {
    await basicFormRef.value.validate()
    basicLoading.value = true
    
    const configs = {
      system_name: basicForm.systemName,
      system_version: basicForm.systemVersion,
      system_description: basicForm.systemDescription,
      system_logo: basicForm.systemLogo
    }
    
    const response = await systemConfigApi.batchUpdate(configs)
    const resp = response.response.data
    if (resp.code === 0) {
      message.success('基本信息保存成功')
      systemInfo.lastUpdate = new Date().toLocaleString()
      
      // 发射全局事件，通知其他组件更新系统Logo
      window.dispatchEvent(new CustomEvent('system-logo-updated', {
        detail: { systemLogo: basicForm.systemLogo }
      }))
    } else {
      message.error(response.message || '保存失败')
    }
  } catch (error) {
    console.error('保存基本信息失败:', error)
    if (error instanceof Error) {
      message.error('保存基本信息失败')
    }
  } finally {
    basicLoading.value = false
  }
}

// 保存系统配置
const handleConfigSubmit = async () => {
  if (!configFormRef.value) return
  
  try {
    await configFormRef.value.validate()
    configLoading.value = true
    
    const configs = {
      maintenance_mode: configForm.maintenanceMode.toString(),
      session_timeout: configForm.sessionTimeout.toString(),
      max_upload_size: (configForm.maxUploadSize * 1024 * 1024).toString()
    }
    
    const response = await systemConfigApi.batchUpdate(configs)
    if (response.code === 200) {
      message.success('系统配置保存成功')
      systemInfo.lastUpdate = new Date().toLocaleString()
    } else {
      message.error(response.message || '保存失败')
    }
  } catch (error) {
    console.error('保存系统配置失败:', error)
    if (error instanceof Error) {
      message.error('保存系统配置失败')
    }
  } finally {
    configLoading.value = false
  }
}

// Logo上传前验证
const beforeLogoUpload = (data: { file: { file: File } }) => {
  console.log('=== beforeLogoUpload 被调用 ===')
  const file = data.file.file || data.file
  const isImage = file.type.startsWith('image/')
  const isLt2M = file.size / 1024 / 1024 < 2

  console.log('文件信息:', {
    name: file.name,
    size: file.size,
    type: file.type,
    sizeInMB: file.size / 1024 / 1024
  })

  if (!isImage) {
    console.log('文件类型验证失败: 不是图片文件')
    message.error('只能上传图片文件!')
    return false
  }
  if (!isLt2M) {
    console.log('文件大小验证失败: 超过2MB')
    message.error(`图片大小不能超过 2MB! 当前大小: ${(file.size / 1024 / 1024).toFixed(2)}MB`)
    return false
  }
  console.log('文件验证通过，允许上传')
  return true
}

// 处理Logo上传
const handleLogoUpload = async (options: any) => {
  console.log('=== handleLogoUpload 被调用 ===')
  console.log('上传选项:', options)
  const { file } = options
  console.log('开始处理Logo上传:', options)
  
  try {
    // 获取实际的文件对象
    const actualFile = file.file || file
    console.log('实际文件对象:', actualFile)
    
    // 将文件转换为base64
    const reader = new FileReader()
    reader.onload = (e) => {
      console.log('文件读取完成，设置Logo')
      const base64Result = e.target?.result as string
      console.log('读取到的base64数据长度:', base64Result ? base64Result.length : 0)
      console.log('base64数据前100个字符:', base64Result ? base64Result.substring(0, 100) : 'null')
      
      basicForm.systemLogo = base64Result
      logoFileList.value = []
      
      console.log('Logo已设置到basicForm:', basicForm.systemLogo ? '有值' : '无值')
      console.log('basicForm.systemLogo长度:', basicForm.systemLogo ? basicForm.systemLogo.length : 0)
      
      // 强制触发响应式更新
      console.log('当前basicForm对象:', JSON.stringify(basicForm, null, 2))
      
      message.success('Logo上传成功')
    }
    reader.onerror = (error) => {
      console.error('文件读取失败:', error)
      message.error('文件读取失败')
    }
    reader.readAsDataURL(actualFile)
  } catch (error) {
    console.error('Logo上传失败:', error)
    message.error('Logo上传失败')
  }
}

// 处理Logo文件变更
const handleLogoChange = (options: { fileList: any[] }) => {
  console.log('=== handleLogoChange 被调用 ===', options)
  const { fileList } = options
  
  // 如果没有文件，直接返回
  if (!fileList || fileList.length === 0) {
    console.log('没有选择文件')
    return
  }
  
  // 获取文件对象
  const fileInfo = fileList[0]
  const file = fileInfo.file || fileInfo
  
  console.log('文件信息:', {
    name: file.name,
    size: file.size,
    type: file.type
  })
  
  try {
    // 将文件转换为base64
    const reader = new FileReader()
    reader.onload = (e) => {
      console.log('文件读取完成，设置Logo')
      const base64Result = e.target?.result as string
      console.log('读取到的base64数据长度:', base64Result ? base64Result.length : 0)
      console.log('base64数据前100个字符:', base64Result ? base64Result.substring(0, 100) : 'null')
      
      basicForm.systemLogo = base64Result
      
      console.log('Logo已设置到basicForm:', basicForm.systemLogo ? '有值' : '无值')
      console.log('basicForm.systemLogo长度:', basicForm.systemLogo ? basicForm.systemLogo.length : 0)
      
      // 强制触发响应式更新
      console.log('当前basicForm对象:', JSON.stringify(basicForm, null, 2))
      
      message.success('Logo上传成功')
    }
    reader.onerror = (error) => {
      console.error('文件读取失败:', error)
      message.error('文件读取失败')
    }
    reader.readAsDataURL(file)
  } catch (error) {
    console.error('Logo上传处理失败:', error)
    message.error('Logo上传失败')
  }
}

// 删除Logo
const handleLogoRemove = () => {
  basicForm.systemLogo = ''
  logoFileList.value = []
}

// 预览Logo
const handleLogoPreview = () => {
  if (basicForm.systemLogo) {
    window.open(basicForm.systemLogo, '_blank')
  }
}

// 运行时间更新定时器
let uptimeTimer = null

// 更新运行时间
const updateUptime = async () => {
  try {
    const response = await systemConfigApi.getSystemInfo()
    if (response.data && response.data.system_info) {
      systemInfo.uptime = response.data.system_info.uptime || '0天0小时0分钟'
    }
  } catch (error) {
    console.error('更新运行时间失败:', error)
  }
}

onMounted(() => {
  loadSystemConfig()
  
  // 每分钟更新一次运行时间
  uptimeTimer = setInterval(() => {
    updateUptime()
  }, 60000)
})

onUnmounted(() => {
  if (uptimeTimer) {
    clearInterval(uptimeTimer)
  }
})
</script>

<style scoped>
.system-settings {
  padding: 24px;
  background: #f5f5f5;
  min-height: 100vh;
}

.page-header {
  margin-bottom: 24px;
}

.page-header h2 {
  margin: 0 0 8px 0;
  font-size: 24px;
  font-weight: 600;
  color: #262626;
}

.page-header p {
  margin: 0;
  color: #8c8c8c;
  font-size: 14px;
}

.settings-container {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.settings-card {
  margin-bottom: 24px;
}

.card-description {
  color: #8c8c8c;
  font-size: 14px;
}

.form-help {
  margin-top: 4px;
  color: #8c8c8c;
  font-size: 12px;
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 16px;
}

.info-item {
  display: flex;
  align-items: center;
  padding: 12px;
  background: #fafafa;
  border-radius: 6px;
}

.info-item .label {
  font-weight: 500;
  color: #595959;
  margin-right: 8px;
  min-width: 80px;
}

.info-item .value {
  color: #262626;
}

.logo-upload-container {
  width: 100%;
}

.logo-preview {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 16px;
  border: 1px solid var(--n-border-color);
  border-radius: 6px;
  background: var(--n-color);
}

.logo-image {
  width: 80px;
  height: 80px;
  object-fit: contain;
  border-radius: 4px;
  border: 1px solid var(--n-border-color);
}

.logo-actions {
  display: flex;
  gap: 8px;
}

.logo-upload {
  width: 100%;
}
</style>