<script setup lang="ts">
import { ref, onMounted, computed, h } from 'vue';
import { NCard, NDataTable, NButton, NTag, NModal, NForm, NFormItem, NSelect, NInput, NDatePicker, NSpace, NStatistic, useMessage, NGrid, NGridItem, NPagination } from 'naive-ui';
import type { DataTableColumns } from 'naive-ui';
import { fetchOrderExceptionList, fetchOrderExceptionStatistics, updateOrderExceptionStatus, type OrderException, type OrderExceptionListParams } from '@/api/order-exception';

const message = useMessage();
const loading = ref(false);
const data = ref<OrderException[]>([]);
const pagination = ref({ page: 1, pageSize: 10, itemCount: 0 });
type NTagType = 'default' | 'primary' | 'success' | 'info' | 'warning' | 'error';
type DateValue = number | null;
type OrderExceptionSearchParams = Omit<OrderExceptionListParams, 'start_date' | 'end_date'> & {
  start_date?: DateValue;
  end_date?: DateValue;
};
const searchParams = ref<OrderExceptionSearchParams>({});

// 初始化默认日期范围（最近一天）
const initDefaultDateRange = () => {
  const today = new Date();
  const yesterday = new Date(today);
  yesterday.setDate(today.getDate() - 1);
  
  searchParams.value.start_date = yesterday.getTime();
  searchParams.value.end_date = today.getTime();
};
const showStatusModal = ref(false);
const currentException = ref<OrderException | null>(null);
const newStatus = ref('');
const statusRemark = ref('');
const statistics = ref({
  total_count: 0,
  pending_count: 0,
  processing_count: 0,
  resolved_count: 0,
  ignored_count: 0,
  balance_verification_count: 0
});

// 异常类型映射
const exceptionTypeMap: Record<string, { color: NTagType, text: string }> = {
  'balance_verification': { color: 'warning', text: '余额验证异常' },
  'payment_timeout': { color: 'error', text: '支付超时' },
  'recharge_failed': { color: 'error', text: '充值失败' },
  'system_error': { color: 'error', text: '系统错误' }
};

// 状态映射
const statusMap: Record<string, { color: NTagType, text: string }> = {
  'pending': { color: 'warning', text: '待处理' },
  'processing': { color: 'info', text: '处理中' },
  'resolved': { color: 'success', text: '已解决' },
  'ignored': { color: 'default', text: '已忽略' }
};

// 状态选项
const statusOptions = [
  { label: '全部', value: '' },
  { label: '待处理', value: 'pending' },
  { label: '处理中', value: 'processing' },
  { label: '已解决', value: 'resolved' },
  { label: '已忽略', value: 'ignored' }
];

// 异常类型选项
const exceptionTypeOptions = [
  { label: '全部', value: '' },
  { label: '余额验证异常', value: 'balance_verification' },
  { label: '支付超时', value: 'payment_timeout' },
  { label: '充值失败', value: 'recharge_failed' },
  { label: '系统错误', value: 'system_error' }
];

// 更新状态选项
const updateStatusOptions = [
  { label: '处理中', value: 'processing' },
  { label: '已解决', value: 'resolved' },
  { label: '已忽略', value: 'ignored' }
];

// 表格列定义
const columns: DataTableColumns<OrderException> = [

  {
    key: 'order_id',
    title: '订单ID',
    width: 200,
    align: 'center',
    render(row) {
      return row.order_id;
    }
  },
  {
    key: 'phone',
    title: '手机号',
    width: 120,
    align: 'center',
    render(row) {
      return row.order?.phone || '-';
    }
  },
  {
    key: 'amount',
    title: '充值金额',
    width: 100,
    align: 'center',
    render(row) {
      return typeof row.order?.amount === 'number' ? `¥${row.order.amount}` : '-';
    }
  },

  {    key: 'exception_reason',    title: '异常原因',    width: 400,    align: 'left',    render(row) {      return row.exception_reason || '-';    }  },
  {
    key: 'status',
    title: '处理状态',
    width: 100,
    align: 'center',
    render(row) {
      const statusInfo = statusMap[row.status as keyof typeof statusMap] || { color: 'default' as NTagType, text: row.status };
      return h(NTag, { type: statusInfo.color }, { default: () => statusInfo.text });
    }
  },
  {
    key: 'create_time',
    title: '创建时间',
    width: 180,
    align: 'center',
    render(row) {
      return new Date(row.create_time).toLocaleString();
    }
  },
  {
    key: 'actions',
    title: '操作',
    width: 120,
    align: 'center',
    render(row) {
      return h(NSpace, {}, {
        default: () => [
          h(NButton, {
            size: 'small',
            type: 'primary',
            ghost: true,
            onClick: () => openStatusModal(row),
            disabled: row.status === 'resolved'
          }, { default: () => '处理' })
        ]
      });
    }
  }
];

// 日期格式转换函数
const formatDateParam = (dateValue: any) => {
  if (!dateValue) return undefined;
  if (typeof dateValue === 'number') {
    return new Date(dateValue).toISOString().split('T')[0];
  }
  return dateValue;
};

// 获取异常列表
const fetchExceptions = async () => {
  try {
    loading.value = true;
    const params = {
      page: pagination.value.page,
      pageSize: pagination.value.pageSize,
      ...searchParams.value,
      start_date: formatDateParam(searchParams.value.start_date),
      end_date: formatDateParam(searchParams.value.end_date)
    };
    const res = await fetchOrderExceptionList(params);
    if (res.data) {
      data.value = res.data.list;
      pagination.value.itemCount = res.data.total;
    }
  } catch (error) {
    message.error('获取异常列表失败');
  } finally {
    loading.value = false;
  }
};

// 获取统计信息
const fetchStatistics = async () => {
  try {
    const startDate = formatDateParam(searchParams.value.start_date);
    const endDate = formatDateParam(searchParams.value.end_date);
    const res = await fetchOrderExceptionStatistics(startDate, endDate);
    if (res.data) {
      const raw: Record<string, number> = res.data as any;
      const entries = Object.entries(raw || {});

      const sumBy = (predicate: (key: string) => boolean) =>
        entries.reduce((sum, [key, val]) => (predicate(key) ? sum + (val || 0) : sum), 0);

      const total_count = entries.reduce((sum, [, val]) => sum + (val || 0), 0);
      const pending_count = sumBy((k) => k.endsWith('_pending'));
      const processing_count = sumBy((k) => k.endsWith('_processing'));
      const resolved_count = sumBy((k) => k.endsWith('_resolved'));
      const ignored_count = sumBy((k) => k.endsWith('_ignored'));
      const balance_verification_count = sumBy((k) => k.startsWith('balance_verification_'));

      statistics.value = {
        total_count,
        pending_count,
        processing_count,
        resolved_count,
        ignored_count,
        balance_verification_count
      };
    }
  } catch (error) {
    console.error('获取统计信息失败:', error);
  }
};

// 搜索
const handleSearch = () => {
  pagination.value.page = 1;
  fetchExceptions();
  fetchStatistics();
};

// 重置搜索
const handleReset = () => {
  searchParams.value = {};
  pagination.value.page = 1;
  fetchExceptions();
  fetchStatistics();
};

// 分页变化
const handlePageChange = (page: number) => {
  pagination.value.page = page;
  fetchExceptions();
};

const handlePageSizeChange = (pageSize: number) => {
  pagination.value.pageSize = pageSize;
  pagination.value.page = 1;
  fetchExceptions();
};

// 打开状态更新模态框
const openStatusModal = (exception: OrderException) => {
  currentException.value = exception;
  newStatus.value = exception.status === 'pending' ? 'processing' : exception.status;
  statusRemark.value = '';
  showStatusModal.value = true;
};

// 更新状态
const handleUpdateStatus = async () => {
  if (!currentException.value || !newStatus.value) {
    message.error('请选择状态');
    return;
  }

  try {
    await updateOrderExceptionStatus(currentException.value.id, newStatus.value, statusRemark.value);
    message.success('状态更新成功');
    showStatusModal.value = false;
    fetchExceptions();
    fetchStatistics();
  } catch (error) {
    message.error('状态更新失败');
  }
};

// 格式化异常数据
const formatExceptionData = (data: any) => {
  if (!data) return '-';
  if (typeof data === 'string') return data;
  return JSON.stringify(data, null, 2);
};

onMounted(() => {
  initDefaultDateRange();
  fetchExceptions();
  fetchStatistics();
});
</script>

<template>
  <div class="order-exception-page">
    <!-- 统计卡片 -->
    <NGrid :cols="6" :x-gap="16" class="mb-4">
      <NGridItem>
        <NCard>
          <NStatistic label="总异常数" :value="statistics.total_count" />
        </NCard>
      </NGridItem>
      <NGridItem>
        <NCard>
          <NStatistic label="待处理" :value="statistics.pending_count" />
        </NCard>
      </NGridItem>
      <NGridItem>
        <NCard>
          <NStatistic label="处理中" :value="statistics.processing_count" />
        </NCard>
      </NGridItem>
      <NGridItem>
        <NCard>
          <NStatistic label="已解决" :value="statistics.resolved_count" />
        </NCard>
      </NGridItem>
      <NGridItem>
        <NCard>
          <NStatistic label="已忽略" :value="statistics.ignored_count" />
        </NCard>
      </NGridItem>
      <NGridItem>
        <NCard>
          <NStatistic label="余额验证异常" :value="statistics.balance_verification_count" />
        </NCard>
      </NGridItem>
    </NGrid>

    <!-- 搜索表单 -->
    <NCard class="mb-4">
      <NForm inline :model="searchParams">
        <NFormItem label="订单ID">
          <NInput v-model:value="searchParams.order_id" placeholder="请输入订单ID" clearable />
        </NFormItem>
        <NFormItem label="异常类型">
          <NSelect
            v-model:value="searchParams.exception_type"
            :options="exceptionTypeOptions"
            placeholder="请选择异常类型"
            clearable
            style="width: 160px"
          />
        </NFormItem>
        <NFormItem label="处理状态">
          <NSelect
            v-model:value="searchParams.status"
            :options="statusOptions"
            placeholder="请选择状态"
            clearable
            style="width: 120px"
          />
        </NFormItem>
        <NFormItem label="开始日期">
          <NDatePicker
            v-model:value="searchParams.start_date"
            type="date"
            placeholder="开始日期"
            clearable
          />
        </NFormItem>
        <NFormItem label="结束日期">
          <NDatePicker
            v-model:value="searchParams.end_date"
            type="date"
            placeholder="结束日期"
            clearable
          />
        </NFormItem>
        <NFormItem>
          <NSpace>
            <NButton type="primary" @click="handleSearch">搜索</NButton>
            <NButton @click="handleReset">重置</NButton>
          </NSpace>
        </NFormItem>
      </NForm>
    </NCard>

    <!-- 数据表格 -->
    <NCard>
      <NDataTable
        :columns="columns"
        :data="data"
        :loading="loading"
        :pagination="false"
        :scroll-x="1400"
        size="small"
      />
      
      <!-- 分页 -->
      <div class="flex justify-end mt-4">
        <NPagination
          v-model:page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :item-count="pagination.itemCount"
          :page-sizes="[10, 50, 100,500,1000]"
          show-size-picker
          show-quick-jumper
          @update:page="handlePageChange"
          @update:page-size="handlePageSizeChange"
        />
      </div>
    </NCard>

    <!-- 状态更新模态框 -->
    <NModal v-model:show="showStatusModal" preset="dialog" title="更新处理状态">
      <div v-if="currentException">
        <div class="mb-4">
          <p><strong>订单ID:</strong> {{ currentException.order_id }}</p>
          <p><strong>异常类型:</strong> {{ exceptionTypeMap[currentException.exception_type]?.text || currentException.exception_type }}</p>
          <p><strong>异常原因:</strong> {{ currentException.exception_reason }}</p>
          <p><strong>当前状态:</strong> {{ statusMap[currentException.status]?.text || currentException.status }}</p>
          <div v-if="currentException.data">
            <p><strong>异常数据:</strong></p>
            <pre class="bg-gray-100 p-2 rounded text-sm">{{ formatExceptionData(currentException.data) }}</pre>
          </div>
        </div>
        
        <NForm>
          <NFormItem label="新状态" required>
            <NSelect
              v-model:value="newStatus"
              :options="updateStatusOptions"
              placeholder="请选择新状态"
            />
          </NFormItem>
          <NFormItem label="处理备注">
            <NInput
              v-model:value="statusRemark"
              type="textarea"
              placeholder="请输入处理备注（可选）"
              :rows="3"
            />
          </NFormItem>
        </NForm>
      </div>
      
      <template #action>
        <NSpace>
          <NButton @click="showStatusModal = false">取消</NButton>
          <NButton type="primary" @click="handleUpdateStatus">确认更新</NButton>
        </NSpace>
      </template>
    </NModal>
  </div>
</template>

<style scoped>
.order-exception-page {
  padding: 16px;
}

pre {
  white-space: pre-wrap;
  word-wrap: break-word;
}
</style>