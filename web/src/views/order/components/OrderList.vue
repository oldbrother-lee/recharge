<script setup lang="tsx">
import { ref, onMounted, watch, nextTick } from 'vue';
import OrderSearchForm from './OrderSearchForm.vue';
import { request } from '@/service/request';
import type { Order } from '@/typings/api';
import { NDataTable, NCard, useMessage, NTag, NButton, NModal, NInput, NForm, NFormItem, NDatePicker } from 'naive-ui';
import type { DataTableColumns } from 'naive-ui';
import { useAuthStore } from '@/store/modules/auth';
import { formatISP } from '@/utils/format';


const authStore = useAuthStore();

const hasRole = (role: string) => {
  return authStore.userInfo.roles.includes(role);
};

const props = withDefaults(defineProps<{
  platform?: string;
  platform_code?: string;
}>(), {
  platform_code: ''
});
const message = useMessage();
const loading = ref(false);
const data = ref<Order[]>([]);
const pagination = ref({ page: 1, pageSize: 10, itemCount: 0 });
const searchParams = ref<any>({});
const showFailModal = ref(false);
const failRemark = ref('');
const currentFailOrder = ref<Order | null>(null);
const showSuccessModal = ref(false);
const currentSuccessOrder = ref<Order | null>(null);
const showDeleteModal = ref(false);
const currentDeleteOrder = ref<Order | null>(null);
const showCleanupModal = ref(false);
const cleanupRange = ref<{ startTime: number | null; endTime: number | null }>({ startTime: null, endTime: null });
const cleanupLoading = ref(false);

// 多选相关状态
const selectedRowKeys = ref<string[]>([]);
const showBatchDeleteModal = ref(false);
const showBatchSuccessModal = ref(false);
const showBatchFailModal = ref(false);
const showBatchNotificationModal = ref(false);
const batchFailRemark = ref('');
const batchLoading = ref(false);

const statusMap: Record<string, { type: 'success' | 'warning' | 'error' | 'info' | 'default', text: string }> = {
  '1': { type: 'warning', text: '待支付' },
  '2': { type: 'warning', text: '待充值' },
  '3': { type: 'info', text: '充值中' },
  '4': { type: 'success', text: '充值成功' },
  '5': { type: 'error', text: '充值失败' },
  '6': { type: 'info', text: '已退款' },
  '7': { type: 'error', text: '已取消' },
  '8': { type: 'warning', text: '部分充值' },
  '9': { type: 'info', text: '已拆单' },
  '10': { type: 'info', text: '处理中' }
};

const handleFail = async (row: Order) => {
  try {
    await request({
      url: `/order/${row.id}/fail`,
      method: 'POST',
      data: { remark: row.remark }
    });
    message.success('订单已标记为失败');
    fetchOrders();
  } catch (error) {
    message.error('操作失败');
  }
};

const handleCancel = async (row: Order) => {
  try {
    await request({ url: `/order/${row.id}/cancel`, method: 'POST', data: { remark: row.remark } });
    message.success('订单已取消');
    fetchOrders();
  } catch (error) {
    message.error('操作失败');
  }
};

const openFailModal = (row: Order) => {
  currentFailOrder.value = row;
  failRemark.value = '';
  showFailModal.value = true;
};

const handleFailConfirm = async () => {
  if (!failRemark.value.trim()) {
    message.error('请填写失败原因');
    return;
  }
  try {
    await request({
      url: `/order/${currentFailOrder.value!.id}/fail`,
      method: 'POST',
      data: { remark: failRemark.value }
    });
    message.success('订单已标记为失败');
    showFailModal.value = false;
    fetchOrders();
  } catch (error) {
    message.error('操作失败');
  }
};

const openSuccessModal = (row: Order) => {
  currentSuccessOrder.value = row;
  showSuccessModal.value = true;
};

const handleSuccessConfirm = async () => {
  try {
    await request({
      url: `/order/${currentSuccessOrder.value!.id}/success`,
      method: 'POST'
    });
    message.success('订单已标记为成功');
    showSuccessModal.value = false;
    fetchOrders();
  } catch (error) {
    message.error('操作失败');
  }
};

const openDeleteModal = (row: Order) => {
  currentDeleteOrder.value = row;
  showDeleteModal.value = true;
};

const handleDeleteConfirm = async () => {
  try {
    await request({
      url: `/order/${currentDeleteOrder.value!.id}/delete`,
      method: 'POST'
    });
    message.success('订单已删除');
    showDeleteModal.value = false;
    fetchOrders();
  } catch (error) {
    message.error('操作失败');
  }
};

const handleCleanup = async () => {
  if (!cleanupRange.value.startTime || !cleanupRange.value.endTime) {
    message.warning('请选择完整的时间范围');
    return;
  }
  cleanupLoading.value = true;
  try {
    const res = await request({
      url: '/order/cleanup',
      method: 'DELETE',
      params: {
        start: formatLocalDatetime(cleanupRange.value.startTime),
        end: formatLocalDatetime(cleanupRange.value.endTime)
      }
    });

    message.success(`清理成功，删除了 ${res.data.deleted} 条订单`);
    showCleanupModal.value = false;
    fetchOrders();
  } catch (error: any) {
    message.error('清理失败: ' + (error?.msg || error?.message || ''));
  } finally {
    cleanupLoading.value = false;
  }
};

// 批量操作函数
const handleBatchDelete = () => {
  if (selectedRowKeys.value.length === 0) {
    message.warning('请选择要删除的订单');
    return;
  }
  showBatchDeleteModal.value = true;
};

const handleBatchSuccess = () => {
  if (selectedRowKeys.value.length === 0) {
    message.warning('请选择要设置为成功的订单');
    return;
  }
  showBatchSuccessModal.value = true;
};

const handleBatchFail = () => {
  if (selectedRowKeys.value.length === 0) {
    message.warning('请选择要设置为失败的订单');
    return;
  }
  batchFailRemark.value = '';
  showBatchFailModal.value = true;
};

const handleBatchNotification = () => {
  if (selectedRowKeys.value.length === 0) {
    message.warning('请选择要发送回调通知的订单');
    return;
  }
  showBatchNotificationModal.value = true;
};

const confirmBatchNotification = async () => {
  batchLoading.value = true;
  try {
    await request({
      url: '/order/batch-notification',
      method: 'POST',
      data: { order_ids: selectedRowKeys.value.map(id => Number(id)) }
    });
    message.success(`成功推送 ${selectedRowKeys.value.length} 个订单到通知队列`);
    selectedRowKeys.value = [];
    showBatchNotificationModal.value = false;
    fetchOrders();
  } catch (error) {
    message.error('批量发送回调通知失败');
  } finally {
    batchLoading.value = false;
  }
};

const confirmBatchDelete = async () => {
  batchLoading.value = true;
  try {
    await request({
      url: '/order/batch-delete',
      method: 'POST',
      data: { order_ids: selectedRowKeys.value.map(id => Number(id)) }
    });
    message.success(`成功删除 ${selectedRowKeys.value.length} 个订单`);
    selectedRowKeys.value = [];
    showBatchDeleteModal.value = false;
    fetchOrders();
  } catch (error) {
    message.error('批量删除失败');
  } finally {
    batchLoading.value = false;
  }
};

const confirmBatchSuccess = async () => {
  batchLoading.value = true;
  try {
    await request({
      url: '/order/batch-success',
      method: 'POST',
      data: { order_ids: selectedRowKeys.value.map(id => Number(id)) }
    });
    message.success(`成功设置 ${selectedRowKeys.value.length} 个订单为成功`);
    selectedRowKeys.value = [];
    showBatchSuccessModal.value = false;
    fetchOrders();
  } catch (error) {
    message.error('批量设置成功失败');
  } finally {
    batchLoading.value = false;
  }
};

const confirmBatchFail = async () => {
  if (!batchFailRemark.value.trim()) {
    message.error('请填写失败原因');
    return;
  }
  batchLoading.value = true;
  try {
    await request({
      url: '/order/batch-fail',
      method: 'POST',
      data: { 
        order_ids: selectedRowKeys.value.map(id => Number(id)),
        remark: batchFailRemark.value 
      }
    });
    message.success(`成功设置 ${selectedRowKeys.value.length} 个订单为失败`);
    selectedRowKeys.value = [];
    showBatchFailModal.value = false;
    fetchOrders();
  } catch (error) {
    message.error('批量设置失败失败');
  } finally {
    batchLoading.value = false;
  }
};

const columns: DataTableColumns<Order> = [
  {
    type: 'selection'
  },
  { key: 'order_number', title: '订单号', align: 'center', minWidth: 180 },
  { key: 'out_trade_num', title: '外部订单号', align: 'center', minWidth: 180 },
  { key: 'mobile', title: '手机号', align: 'center', width: 120 },
  { 
    key: 'isp', 
    title: '运营商', 
    align: 'center', 
    width: 120,
    render(row) {
      return formatISP(row.isp);
    }
  },
  { key: 'account_location', title: '归属地', align: 'center', width: 100 },
  { key: 'denom', title: '订单金额', align: 'center', width: 100 },
  {
    key: 'status',
    title: '订单状态',
    align: 'center',
    width: 100,
    render(row) {
      const status = statusMap[String(row.status)] || { type: 'default', text: String(row.status) };
      return <NTag type={status.type}>{status.text}</NTag>;
    }
  },

  {
    key: 'notification_time',
    title: '通知时间',
    align: 'center',
    width: 180,
    render(row) {
      if (!(row as any).notification_time) {
        return '-';
      }
      const d = new Date((row as any).notification_time);
      const pad = (n: number) => n.toString().padStart(2, '0');
      return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
    }
  },
    {
    key: 'notification_status',
    title: '通知状态',
    align: 'center',
    width: 100,
    render(row) {
      const status = (row as any).notification_status;
      if (!status) {
        return '-';
      }
      const statusMap: { [key: string]: { type: string; text: string } } = {
        '1': { type: 'warning', text: '待通知' },
        '2': { type: 'info', text: '通知中' },
        '3': { type: 'success', text: '成功' },
        '4': { type: 'error', text: '失败' }
      };
      const statusInfo = statusMap[String(status)] || { type: 'default', text: String(status) };
      return <NTag type={statusInfo.type}>{statusInfo.text}</NTag>;
    }
  },
  {
    key: 'platform_name',
    title: '来源',
    align: 'center',
    width: 100,
    render(row) {
      return (row as any).platform_name || 'API下单';
    }
  },
  {
    key: 'create_time',
    title: '创建时间',
    align: 'center',
    width: 180,
    render(row) {
      const d = new Date((row as any).create_time || (row as any).createTime || '');
      const pad = (n: number) => n.toString().padStart(2, '0');
      return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
    }
  },
  {
    key: 'operate',
    title: '操作',
    align: 'center',
    width: 400,
    render(row) {
      return (
        <div style={{ display: 'flex', gap: '8px', justifyContent: 'center' }}>
          <NButton size="small" type="success" ghost onClick={() => openSuccessModal(row)}>
            设置为成功
          </NButton>
          <NButton size="small" type="error" ghost onClick={() => openFailModal(row)}>
            失败订单
          </NButton>
          <NButton size="small" type="warning" ghost onClick={() => openDeleteModal(row)}>
            删除订单
          </NButton>
        </div>
      );
    }
  }
];

const fetchOrders = async () => {
  try {
    loading.value = true;
    const params: any = {
      page: pagination.value.page,
      pageSize: pagination.value.pageSize,
      ...searchParams.value
    };
    if (props.platform_code) {
      params.platform_code = props.platform_code;
    } else if (props.platform && props.platform !== 'all') {
      params.platform = props.platform;
    }
    const res = await request({ url: '/order/list', method: 'GET', params });
    if (res.data) {
      data.value = res.data.list;
      pagination.value.itemCount = res.data.total;
    }
  } catch (error) {
    message.error('获取订单列表失败');
  } finally {
    loading.value = false;
  }
};

const handleSearch = (params: any) => {
  searchParams.value = params;
  pagination.value.page = 1;
  fetchOrders();
};

const handlePageChange = (page: number) => {
  pagination.value.page = page;
  fetchOrders();
};

const handlePageSizeChange = (size: number) => {
  pagination.value.pageSize = size;
  fetchOrders();
};

const handleRowKeysUpdate = async (keys: string[]) => {
  try {
    selectedRowKeys.value = keys;
    await nextTick();
  } catch (error) {
    console.warn('更新选中行时出现错误:', error);
    // 如果出现错误，延迟更新
    setTimeout(() => {
      selectedRowKeys.value = keys;
    }, 0);
  }
};

watch(() => [props.platform, props.platform_code], () => {
  fetchOrders();
});

onMounted(() => {
  fetchOrders();
});

function formatLocalDatetime(ts: number | null) {
  if (!ts) return '';
  const d = new Date(ts);
  const pad = (n: number) => n.toString().padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}
</script>

<template>
  <div class="min-h-1200px flex-col-stretch gap-16px overflow-hidden lt-sm:overflow-auto">
    <!-- 搜索表单 -->
    <OrderSearchForm @search="handleSearch" />
    
    <!-- 数据表格 -->
    <NCard size="small" class="sm:flex-1-hidden card-wrapper">
      <template #header>
        <div class="header-wrapper">
          <span>订单列表</span>
          <div class="button-group">
            <NButton
              v-show="selectedRowKeys.length > 0"
              type="success"
              size="small"
              @click="handleBatchSuccess"
              class="batch-btn"
            >
              <span class="btn-text">
                <span class="btn-text-full">批量设置成功 ({{ selectedRowKeys.length }})</span>
                <span class="btn-text-short">成功 ({{ selectedRowKeys.length }})</span>
              </span>
            </NButton>
            <NButton
              v-show="selectedRowKeys.length > 0"
              type="error"
              size="small"
              @click="handleBatchFail"
              class="batch-btn"
            >
              <span class="btn-text">
                <span class="btn-text-full">批量设置失败 ({{ selectedRowKeys.length }})</span>
                <span class="btn-text-short">失败 ({{ selectedRowKeys.length }})</span>
              </span>
            </NButton>
            <NButton
              v-show="selectedRowKeys.length > 0"
              type="warning"
              size="small"
              @click="handleBatchDelete"
              class="batch-btn"
            >
              <span class="btn-text">
                <span class="btn-text-full">批量删除 ({{ selectedRowKeys.length }})</span>
                <span class="btn-text-short">删除 ({{ selectedRowKeys.length }})</span>
              </span>
            </NButton>
            <NButton
              v-show="selectedRowKeys.length > 0"
              type="info"
              size="small"
              @click="handleBatchNotification"
              class="batch-btn"
            >
              <span class="btn-text">
                <span class="btn-text-full">批量发送回调 ({{ selectedRowKeys.length }})</span>
                <span class="btn-text-short">回调 ({{ selectedRowKeys.length }})</span>
              </span>
            </NButton>
            <NButton
              v-if="props.platform === 'all' && hasRole('SUPER_ADMIN')"
              type="error"
              size="small"
              @click="showCleanupModal = true"
              class="batch-btn"
            >
              <span class="btn-text">
                <span class="btn-text-full">清理订单</span>
                <span class="btn-text-short">清理</span>
              </span>
            </NButton>
          </div>
        </div>
      </template>
      <NDataTable
        :columns="columns"
        :data="data"
        :loading="loading"
        :pagination="pagination"
        :flex-height="true"
        :scroll-x="1800"
        remote
        checkable
        :row-key="row => String(row.id)"
        :checked-row-keys="selectedRowKeys"
        @update:checked-row-keys="handleRowKeysUpdate"
        @update:page="handlePageChange"
        @update:page-size="handlePageSizeChange"
        class="h-full"
        size="small"
      />
    </NCard>
    <NModal v-model:show="showFailModal" title="标记为失败" preset="dialog">
      <NForm>
        <NFormItem label="失败原因" required>
          <NInput v-model:value="failRemark" type="textarea" placeholder="请输入失败原因" />
        </NFormItem>
      </NForm>
      <template #action>
        <NButton @click="() => (showFailModal = false)">取消</NButton>
        <NButton type="primary" @click="handleFailConfirm">确定</NButton>
      </template>
    </NModal>
    <NModal v-model:show="showSuccessModal" title="设置为成功" preset="dialog">
      <div>确认将该订单设置为成功吗？</div>
      <template #action>
        <NButton @click="() => (showSuccessModal = false)">取消</NButton>
        <NButton type="primary" @click="handleSuccessConfirm">确定</NButton>
      </template>
    </NModal>
    <NModal v-model:show="showDeleteModal" title="删除订单" preset="dialog">
      <div>确认要删除该订单吗？</div>
      <template #action>
        <NButton @click="() => (showDeleteModal = false)">取消</NButton>
        <NButton type="primary" @click="handleDeleteConfirm">确定</NButton>
      </template>
    </NModal>
    <NModal v-model:show="showCleanupModal" title="清理订单" preset="dialog">
      <NForm>
        <NFormItem label="开始时间" required>
          <NDatePicker
            v-model:value="cleanupRange.startTime"
            type="datetime"
            clearable
            style="width: 100%"
            placeholder="选择开始时间"
          />
        </NFormItem>
        <NFormItem label="结束时间" required>
          <NDatePicker
            v-model:value="cleanupRange.endTime"
            type="datetime"
            clearable
            style="width: 100%"
            placeholder="选择结束时间"
          />
        </NFormItem>
      </NForm>
      <template #action>
        <NButton @click="() => (showCleanupModal = false)">取消</NButton>
        <NButton type="error" :loading="cleanupLoading" @click="handleCleanup" style="margin-left: 12px">确认清理</NButton>
      </template>
    </NModal>
    
    <!-- 批量操作模态框 -->
    <NModal v-model:show="showBatchDeleteModal" title="批量删除订单" preset="dialog">
      <div>确认要删除选中的 {{ selectedRowKeys.length }} 个订单吗？</div>
      <template #action>
        <NButton @click="() => (showBatchDeleteModal = false)">取消</NButton>
        <NButton type="error" :loading="batchLoading" @click="confirmBatchDelete">确定删除</NButton>
      </template>
    </NModal>
    
    <NModal v-model:show="showBatchSuccessModal" title="批量设置成功" preset="dialog">
      <div>确认将选中的 {{ selectedRowKeys.length }} 个订单设置为成功吗？</div>
      <template #action>
        <NButton @click="() => (showBatchSuccessModal = false)">取消</NButton>
        <NButton type="success" :loading="batchLoading" @click="confirmBatchSuccess">确定</NButton>
      </template>
    </NModal>
    
    <NModal v-model:show="showBatchFailModal" title="批量设置失败" preset="dialog">
      <NForm>
        <NFormItem label="失败原因" required>
          <NInput v-model:value="batchFailRemark" type="textarea" placeholder="请输入失败原因" />
        </NFormItem>
        <div style="margin-bottom: 12px; color: #666;">将对选中的 {{ selectedRowKeys.length }} 个订单进行操作</div>
      </NForm>
      <template #action>
        <NButton @click="() => (showBatchFailModal = false)">取消</NButton>
        <NButton type="error" :loading="batchLoading" @click="confirmBatchFail">确定</NButton>
      </template>
    </NModal>
    
    <NModal v-model:show="showBatchNotificationModal" title="批量发送回调通知" preset="dialog">
      <div>确认将选中的 {{ selectedRowKeys.length }} 个订单推送到通知队列进行回调通知吗？</div>
      <template #action>
        <NButton @click="() => (showBatchNotificationModal = false)">取消</NButton>
        <NButton type="info" :loading="batchLoading" @click="confirmBatchNotification">确定发送</NButton>
      </template>
    </NModal>
  </div>
</template>

<style scoped>
.min-h-500px {
  min-height: 500px;
}
.flex-col-stretch {
  display: flex;
  flex-direction: column;
  height: 100%;
}
.gap-16px {
  gap: 16px;
}
.lt-sm\:overflow-auto {
  @media (max-width: 640px) {
    overflow: auto;
  }
}
.overflow-hidden {
  overflow: hidden;
}
.sm\:flex-1-hidden {
  @media (min-width: 640px) {
    flex: 1;
    overflow: hidden;
  }
}
.card-wrapper {
  display: flex;
  flex-direction: column;
  height: 100%;
}
.h-full {
  height: 100%;
}
.flex-center {
  display: flex;
  align-items: center;
  justify-content: center;
}
.gap-8px {
  gap: 8px;
}

/* 头部样式 */
.header-wrapper {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.button-group {
  display: flex;
  gap: 8px;
  margin-left: auto;
  flex-wrap: wrap;
}

.batch-btn .btn-text-short {
  display: none;
}

.batch-btn .btn-text-full {
  display: inline;
}

/* 操作按钮样式 */
.operation-buttons {
  display: flex;
  gap: 8px;
  justify-content: center;
  flex-wrap: wrap;
}

.op-btn .op-btn-text-short {
  display: none;
}

.op-btn .op-btn-text-full {
  display: inline;
}

/* 移动端样式 */
@media (max-width: 640px) {
  .header-wrapper {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }
  
  .button-group {
    margin-left: 0;
    width: 100%;
    justify-content: flex-start;
  }
  
  .batch-btn .btn-text-full {
    display: none;
  }
  
  .batch-btn .btn-text-short {
    display: inline;
  }
  
  .operation-buttons {
    gap: 4px;
  }
  
  .op-btn .op-btn-text-full {
    display: none;
  }
  
  .op-btn .op-btn-text-short {
    display: inline;
  }
  
  /* 表格移动端优化 */
  .n-data-table {
    font-size: 12px !important;
  }
  
  .n-data-table .n-data-table-td,
  .n-data-table .n-data-table-th {
    white-space: nowrap !important;
    padding: 6px 4px !important;
    font-size: 12px !important;
    line-height: 1.2 !important;
  }
  
  .n-data-table .n-data-table-td {
    min-height: 32px !important;
  }
  
  /* 表格内容优化 */
  .n-data-table .n-tag {
    font-size: 11px !important;
    padding: 2px 6px !important;
    line-height: 1.2 !important;
  }
  
  /* 分页器移动端优化 */
  .n-pagination {
    justify-content: center !important;
  }
  
  .n-pagination .n-pagination-item {
    min-width: 28px !important;
    height: 28px !important;
    font-size: 12px !important;
  }
}

@media (max-width: 480px) {
  .button-group {
    gap: 4px;
    flex-wrap: wrap;
  }
  
  .batch-btn {
    font-size: 11px !important;
    padding: 3px 6px !important;
    min-width: auto !important;
  }
  
  .operation-buttons {
    gap: 2px;
    flex-direction: column;
    align-items: center;
  }
  
  .op-btn {
    font-size: 10px !important;
    padding: 2px 4px !important;
    min-width: 36px !important;
    line-height: 1.2 !important;
  }
  
  /* 极小屏幕表格优化 */
  .n-data-table .n-data-table-td,
  .n-data-table .n-data-table-th {
    padding: 4px 2px !important;
    font-size: 11px !important;
  }
  
  .n-data-table .n-tag {
    font-size: 10px !important;
    padding: 1px 4px !important;
  }
  
  /* 分页器极小屏幕优化 */
  .n-pagination .n-pagination-item {
    min-width: 24px !important;
    height: 24px !important;
    font-size: 11px !important;
  }
  
  .n-pagination .n-pagination-prefix,
  .n-pagination .n-pagination-suffix {
    font-size: 11px !important;
  }
}
</style>
