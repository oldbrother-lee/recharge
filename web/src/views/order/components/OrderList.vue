<script setup lang="tsx">
import { ref, onMounted, watch, nextTick, computed } from 'vue';
import OrderSearchForm from './OrderSearchForm.vue';
import { request } from '@/service/request';
import type { Order } from '@/typings/api';
import { NDataTable, NCard, useMessage, NTag, NButton, NModal, NInput, NForm, NFormItem, NDatePicker } from 'naive-ui';
import type { DataTableColumns } from 'naive-ui';
import { useAuthStore } from '@/store/modules/auth';
import { formatISP } from '@/utils/format';
// 本地定义 RowKey 类型以兼容 Naive UI DataTable 的选中键类型
type RowKey = string | number;


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
// 成功统计（按运营商与面值）
const successStatsLoading = ref(false);
const successStats = ref<Array<{ isp: number; denom: number; successCount: number; successAmount: number }>>([]);
const hasSearch = computed(() => {
  const p = searchParams.value || {};
  return Object.keys(p).length > 0;
});
// 成功统计总计
const successStatsTotals = computed(() => {
  const list = successStats.value || [];
  const count = list.reduce((acc, r) => acc + Number(r.successCount || 0), 0);
  const amount = list.reduce((acc, r) => acc + Number(r.successAmount || 0), 0);
  return { count, amount };
});

// 成功统计分组（按运营商）
const successStatsGroups = computed(() => {
  const map = new Map<string, Array<{ denom: number; successCount: number; successAmount: number }>>();
  for (const r of successStats.value || []) {
    const ispLabel = formatISP(String(r.isp));
    if (!map.has(ispLabel)) map.set(ispLabel, []);
    map.get(ispLabel)!.push({
      denom: Number(r.denom || 0),
      successCount: Number(r.successCount || 0),
      successAmount: Number(r.successAmount || 0)
    });
  }
  return Array.from(map.entries()).map(([isp, items]) => ({ isp, items }));
});
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
const selectedRowKeys = ref<RowKey[]>([]);
const showBatchDeleteModal = ref(false);
const showBatchSuccessModal = ref(false);
const showBatchFailModal = ref(false);
const showBatchNotificationModal = ref(false);
const batchFailRemark = ref('');
const batchLoading = ref(false);

// 余额验证状态
const balanceVerificationEnabled = ref(false);

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
      },
      timeout: 600000 // 30秒超时，因为清理操作可能需要较长时间
    });

    message.success(`清理成功，删除了 ${res.data?.deleted || 0} 条订单`);
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
      data: { order_ids: selectedRowKeys.value.map((id: RowKey) => Number(id)) }
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
      data: { order_ids: selectedRowKeys.value.map((id: RowKey) => Number(id)) }
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
      data: { order_ids: selectedRowKeys.value.map((id: RowKey) => Number(id)) }
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
        order_ids: selectedRowKeys.value.map((id: RowKey) => Number(id)),
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

// 初始化余额验证状态
const initBalanceVerificationStatus = async () => {
  try {
    const res = await request({
      url: '/systemManage/key/balance_verification_enabled',
      method: 'GET'
    });
    balanceVerificationEnabled.value = res.data?.config_value === 'true';
  } catch (error) {
    console.error('获取余额验证状态失败:', error);
  }
};

const toggleBalanceQuery = async () => {
  try {
    const res = await request({
      url: '/systemManage/key/balance_verification_enabled',
      method: 'GET'
    });
    
    const currentValue = res.data?.config_value === 'true';
    const newValue = !currentValue;
    
    await request({
      url: '/systemManage/settings/batch',
      method: 'PUT',
      data: {
        balance_verification_enabled: newValue.toString()
      }
    });
    
    // 更新本地状态
    balanceVerificationEnabled.value = newValue;
    
    message.success(`余额查询已${newValue ? '开启' : '关闭'}`);
  } catch (error) {
    message.error('切换余额查询状态失败');
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
      const value = (row as any).isp;
      const s = typeof value === 'string' ? value : Array.isArray(value) ? value.join(',') : '';
      return formatISP(s);
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
      const statusMap: { [key: string]: { type: 'default' | 'error' | 'info' | 'success' | 'warning'; text: string } } = {
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

// 成功统计列表列定义
const successStatsColumns: DataTableColumns<any> = [
  { key: 'isp', title: '运营商', align: 'center', width: 120, render(row) {
      const v = String(row.isp);
      return formatISP(v);
    }
  },
  { key: 'denom', title: '面值', align: 'center', width: 100 },
  { key: 'successCount', title: '成功笔数', align: 'center', width: 120 },
  { key: 'successAmount', title: '成功金额', align: 'center', width: 120, render(row) {
      const amt = Number(row.successAmount || 0);
      return amt.toFixed(2);
    }
  }
];

const fetchOrders = async () => {
  try {
    loading.value = true;
    // 规范化查询参数：
    const normalizeParams = (raw: any) => {
      const p: any = { ...raw };
      // 日期范围转换为后端需要的 start_time / end_time 字符串
      if (Array.isArray(p.date_range) && p.date_range.length === 2 && p.date_range[0] && p.date_range[1]) {
        const startMs = Number(p.date_range[0]);
        const endMs = Number(p.date_range[1]);
        // 包含结束当日：设为当天 23:59:59
        const endOfDayMs = endMs + 24 * 60 * 60 * 1000 - 1000;
        const fmt = (ms: number) => {
          const d = new Date(ms);
          const pad = (n: number) => n.toString().padStart(2, '0');
          return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
        };
        p.start_time = fmt(startMs);
        p.end_time = fmt(endOfDayMs);
        delete p.date_range;
      }
      return p;
    };
    const params: any = {
      page: pagination.value.page,
      page_size: pagination.value.pageSize,
      ...normalizeParams(searchParams.value)
    };
    if (props.platform_code) {
      params.platform_code = props.platform_code;
    } else if (props.platform && props.platform !== 'all') {
      params.platform = props.platform;
    }
    const res = await request<{ list: Order[]; total: number }>({ url: '/order/list', method: 'GET', params });
    // createFlatRequest 返回 { data, error, response }
    // 成功时使用 res.data.list / res.data.total
    if (!res.error) {
      data.value = Array.isArray(res.data?.list) ? res.data!.list : [];
      pagination.value.itemCount = Number(res.data?.total || 0);
    } else {
      // 失败则重置为空，避免保留上一次内容
      data.value = [];
      pagination.value.itemCount = 0;
    }
  } catch (error) {
    message.error('获取订单列表失败');
    // 发生错误时重置为空，避免保留上一次内容
    data.value = [];
    pagination.value.itemCount = 0;
  } finally {
    loading.value = false;
  }
};

// 获取按运营商与面值的成功统计
const fetchSuccessStats = async () => {
  try {
    successStatsLoading.value = true;
    const normalizeParams = (raw: any) => {
      const p: any = { ...raw };
      if (Array.isArray(p.date_range) && p.date_range.length === 2 && p.date_range[0] && p.date_range[1]) {
        const startMs = Number(p.date_range[0]);
        const endMs = Number(p.date_range[1]);
        const endOfDayMs = endMs + 24 * 60 * 60 * 1000 - 1000;
        const fmt = (ms: number) => {
          const d = new Date(ms);
          const pad = (n: number) => n.toString().padStart(2, '0');
          return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
        };
        p.start_time = fmt(startMs);
        p.end_time = fmt(endOfDayMs);
        delete p.date_range;
      }
      return p;
    };
    const params: any = { ...normalizeParams(searchParams.value) };
    if (props.platform_code) {
      params.platform_code = props.platform_code;
    } else if (props.platform && props.platform !== 'all') {
      params.platform = props.platform;
    }
    const res = await request<{ list: Array<{ isp: number; denom: number; successCount: number; successAmount: number }>; total: number }>({ url: '/orders/statistics/isp-denom-success', method: 'GET', params });
    successStats.value = Array.isArray(res.data?.list) ? res.data!.list : [];
  } catch (error) {
    message.error('获取成功统计失败');
  } finally {
    successStatsLoading.value = false;
  }
};

const handleSearch = (params: any) => {
  searchParams.value = params;
  pagination.value.page = 1;
  fetchOrders();
  // 触发成功统计数据的获取，仅在搜索后加载
  fetchSuccessStats();
};

const handlePageChange = (page: number) => {
  pagination.value.page = page;
  fetchOrders();
};

const handlePageSizeChange = (size: number) => {
  pagination.value.pageSize = size;
  fetchOrders();
};

const handleRowKeysUpdate = (keys: RowKey[]) => {
  try {
    selectedRowKeys.value = keys;
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
  initBalanceVerificationStatus();
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
    <!-- 成功统计（按运营商分组展示，置于卡片中） -->
    <NCard v-if="hasSearch" size="small" class="stats-card" :class="{ 'opacity-60': successStatsLoading }">
      <template #header>成功统计</template>
      <div class="stats-text">
        <div class="stats-summary">总笔数 {{ successStatsTotals.count }}，总金额 ¥{{ successStatsTotals.amount.toFixed(2) }}</div>
        <div v-if="successStatsGroups.length === 0">暂无数据</div>
        <div v-else class="stats-groups">
          <div class="stats-group" v-for="group in successStatsGroups" :key="group.isp">
            <span class="isp">{{ group.isp }}：</span>
            <span class="detail">
              <template v-for="(it, idx) in group.items" :key="idx">
                {{ it.denom.toFixed(2) }}: {{ it.successCount }}笔/¥{{ it.successAmount.toFixed(2) }}<span v-if="idx < group.items.length - 1">；</span>
              </template>
            </span>
          </div>
        </div>
      </div>
    </NCard>
    
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
            <NButton
              v-if="props.platform === 'all' && hasRole('SUPER_ADMIN')"
              type="primary"
              size="small"
              @click="toggleBalanceQuery"
              class="batch-btn"
            >
              <span class="btn-text">
                <span class="btn-text-full">{{ balanceVerificationEnabled ? '关闭查询余额' : '开启查询余额' }}</span>
                <span class="btn-text-short">查询余额</span>
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
/* 统计文字样式 */
.stats-text {
  font-size: 13px;
  color: #666;
}
.stats-card {
  /* 让统计卡片与列表保持一致的间距与视觉层级 */
}
.stats-summary {
  margin-bottom: 4px;
}
.stats-groups {
  margin-top: 2px;
}
.stats-group {
  margin-top: 2px;
}
.stats-group .isp {
  font-weight: 500;
  color: #333;
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
