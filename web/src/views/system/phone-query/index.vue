<template>
  <div class="phone-query-management">
    <div class="page-header">
      <h2>手机查询管理</h2>
      <p>手机余额查询和缴费记录查询</p>
    </div>

    <div class="query-container">
      <!-- 余额查询 -->
      <n-card title="手机余额查询" class="query-card">
        <template #header-extra>
          <span class="card-description">查询手机号码余额信息</span>
        </template>
        <n-form
          ref="balanceFormRef"
          :model="balanceForm"
          :rules="balanceRules"
          label-placement="left"
          :label-width="120"
          @submit.prevent="handleBalanceQuery"
        >
          <n-form-item label="手机号码" path="phone">
            <n-input
              v-model:value="balanceForm.phone"
              placeholder="请输入手机号码"
              :maxlength="11"
              show-count
            />
          </n-form-item>
          <n-form-item label="运营商类型" path="isp_type">
            <n-select
              v-model:value="balanceForm.isp_type"
              :options="ispOptions"
              placeholder="请选择运营商类型"
            />
          </n-form-item>
          <n-form-item>
            <n-button type="primary" :loading="balanceLoading" @click="handleBalanceQuery">
              查询余额
            </n-button>
          </n-form-item>
        </n-form>
        
        <!-- 余额查询结果 -->
        <div v-if="balanceResult" class="query-result">
          <n-divider title-placement="left">查询结果</n-divider>
          <div class="result-grid">
            <div class="result-item">
              <span class="label">手机号码：</span>
              <span class="value">{{ balanceForm.phone }}</span>
            </div>
            <div class="result-item">
              <span class="label">运营商：</span>
              <span class="value">{{ getIspName(balanceForm.isp_type) }}</span>
            </div>
            <div class="result-item">
              <span class="label">余额：</span>
              <span class="value balance-amount">¥{{ balanceResult.datas }}</span>
            </div>

          </div>
        </div>
      </n-card>

      <!-- 缴费记录查询 -->
      <n-card title="缴费记录查询" class="query-card">
        <template #header-extra>
          <span class="card-description">查询手机号码缴费记录</span>
        </template>
        <n-form
          ref="recordFormRef"
          :model="recordForm"
          :rules="recordRules"
          label-placement="left"
          :label-width="120"
          @submit.prevent="handleRecordQuery"
        >
          <n-form-item label="手机号码" path="phone">
            <n-input
              v-model:value="recordForm.phone"
              placeholder="请输入手机号码"
              :maxlength="11"
              show-count
            />
          </n-form-item>
          <n-form-item label="运营商类型" path="isp_type">
            <n-select
              v-model:value="recordForm.isp_type"
              :options="ispOptions"
              placeholder="请选择运营商类型"
            />
          </n-form-item>

    
          <n-form-item>
            <n-button type="primary" :loading="recordLoading" @click="handleRecordQuery">
              查询记录
            </n-button>
          </n-form-item>
        </n-form>
        
        <!-- 缴费记录查询结果 -->
        <div v-if="recordResult" class="query-result">
          <n-divider title-placement="left">查询结果</n-divider>
          <div class="result-info">
            <div class="result-item">
              <span class="label">手机号码：</span>
              <span class="value">{{ recordForm.phone }}</span>
            </div>
            <div class="result-item">
              <span class="label">运营商：</span>
              <span class="value">{{ getIspName(recordForm.isp_type) }}</span>
            </div>
            <div class="result-item">
              <span class="label">记录数量：</span>
              <span class="value">{{ recordResult.records?.length || 0 }} 条</span>
            </div>
            <div class="result-item">
              <span class="label">查询时间：</span>
              <span class="value">{{ new Date().toLocaleString() }}</span>
            </div>
          </div>
          
          <!-- 缴费记录表格 -->
          <n-data-table
          v-if="recordResult.records && recordResult.records.length > 0"
          :columns="recordColumns"
          :data="recordResult.records"
          :pagination="false"
          :bordered="false"
          :scroll-x="600"
          size="small"
          class="record-table"
        />
          <n-empty v-else description="暂无缴费记录" />
        </div>
      </n-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useMessage, NCard, NForm, NFormItem, NInput, NSelect, NInputNumber, NButton, NDivider, NDataTable, NEmpty } from 'naive-ui'
import type { FormInst, FormRules, DataTableColumns } from 'naive-ui'
import { request } from '@/service/request'

const message = useMessage()

// 运营商选项
const ispOptions = [
  { label: '中国移动', value: 'yd' },
  { label: '中国联通', value: 'lt' },
  { label: '中国电信', value: 'dx' }
]

// 余额查询表单
const balanceForm = reactive({
  phone: '',
  isp_type: null
})

// 缴费记录查询表单
const recordForm = reactive({
  phone: '',
  isp_type: null
})

const balanceLoading = ref(false)
const recordLoading = ref(false)
const balanceResult = ref(null)
const recordResult = ref(null)

const balanceFormRef = ref<FormInst | null>(null)
const recordFormRef = ref<FormInst | null>(null)

// 表单验证规则
const balanceRules: FormRules = {
  phone: [
    { required: true, message: '请输入手机号码', trigger: 'blur' },
    { pattern: /^1[3-9]\d{9}$/, message: '请输入正确的手机号码', trigger: 'blur' }
  ],
  isp_type: [
    { required: true, type: 'string', message: '请选择运营商类型', trigger: 'change' }
  ]
}

const recordRules: FormRules = {
  phone: [
    { required: true, message: '请输入手机号码', trigger: 'blur' },
    { pattern: /^1[3-9]\d{9}$/, message: '请输入正确的手机号码', trigger: 'blur' }
  ],
  isp_type: [
    { required: true, type: 'string', message: '请选择运营商类型', trigger: 'change' }
  ]
}

// 缴费记录表格列定义
const recordColumns: DataTableColumns = [
  {
    title: '缴费时间',
    key: 'payTime',
    width: 180
  },
  {
    title: '缴费金额',
    key: 'payAmount',
    width: 120,
    render(row: any) {
      return `¥${row.payAmount}`
    }
  },
  {
    title: '缴费渠道',
    key: 'channel',

  },
  {
    title: '时间戳',
    key: 'payTimeStamp',
    width: 120
  }
]

// 获取运营商名称
const getIspName = (ispType: string) => {
  const isp = ispOptions.find(item => item.value === ispType)
  return isp ? isp.label : '未知'
}

// 格式化时间
const formatTime = (time: string) => {
  if (!time) return '-'
  return new Date(time).toLocaleString()
}

// 余额查询
const handleBalanceQuery = async () => {
  try {
    await balanceFormRef.value?.validate()
    balanceLoading.value = true
    balanceResult.value = null
    
    const response = await request({
      url: '/phone/balance',
      method: 'POST',
      data: balanceForm
    })
    
    if (response.data) {
      balanceResult.value = response.data
      message.success('余额查询成功')
    }
  } catch (error: any) {
    message.error(error?.message || '余额查询失败')
  } finally {
    balanceLoading.value = false
  }
}

// 缴费记录查询
const handleRecordQuery = async () => {
  try {
    await recordFormRef.value?.validate()
    recordLoading.value = true
    recordResult.value = null
    
    const response = await request({
      url: '/phone/payment-records',
      method: 'POST',
      data: recordForm
    })
    
    if (response.data) {
      // 适配API返回的数据结构
      if (response.data && response.data.datas) {
        recordResult.value = {
          ...response.data.data,
          records: response.data.datas
        }
      } else {
        recordResult.value = response.data
      }
      message.success('缴费记录查询成功')
    }
  } catch (error: any) {
    message.error(error?.message || '缴费记录查询失败')
  } finally {
    recordLoading.value = false
  }
}
</script>

<style scoped>
.phone-query-management {
  padding: 20px;
}

.page-header {
  margin-bottom: 24px;
}

.page-header h2 {
  margin: 0 0 8px 0;
  font-size: 24px;
  font-weight: 600;
  color: #1a1a1a;
}

.page-header p {
  margin: 0;
  color: #666;
  font-size: 14px;
}

.query-container {
  display: grid;
  gap: 24px;
  grid-template-columns: 1fr;
}

.query-card {
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.card-description {
  color: #666;
  font-size: 14px;
}

.query-result {
  margin-top: 24px;
  padding: 16px;
  background: #f8f9fa;
  border-radius: 6px;
}

.result-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
  margin-bottom: 16px;
}

.result-info {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
  margin-bottom: 16px;
}

.result-item {
  display: flex;
  align-items: center;
}

.result-item .label {
  font-weight: 500;
  color: #666;
  margin-right: 8px;
  min-width: 80px;
}

.result-item .value {
  color: #1a1a1a;
}

.balance-amount {
  font-weight: 600;
  color: #18a058;
  font-size: 16px;
}

.record-table {
  margin-top: 16px;
}

@media (max-width: 768px) {
  .query-container {
    padding: 0 16px;
  }
  
  .result-grid,
  .result-info {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 640px) {
  .n-data-table .n-data-table-td,
  .n-data-table .n-data-table-th {
    white-space: nowrap !important;
    padding-top: 4px !important;
    padding-bottom: 4px !important;
    font-size: 13px !important;
  }
  .n-data-table .n-data-table-td {
    min-height: 28px !important;
  }
}
</style>