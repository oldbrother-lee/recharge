<script setup lang="ts">
import { ref, h, watch, onMounted } from 'vue';
import {
  NCard, NSpace, NButton, NInput, NSelect, NDatePicker, NForm, NFormItem, NDataTable, NPagination, NTag,
  NModal, NCheckbox, NCheckboxGroup, NSpin, useMessage, NPopconfirm
} from 'naive-ui';
import type { SelectOption, DataTableColumns } from 'naive-ui';
import { getChannelList } from '@/api/platform';
import { request } from '@/service/request';
import { getTaskConfigList, deleteTaskConfig, updateTaskConfig } from '@/api/taskConfig'

// 搜索表单数据
const searchForm = ref({
  yr_order_id: '',
  status: undefined as number | undefined,
  dateRange: null as [number, number] | null
});

// 订单状态选项
const statusOptions: SelectOption[] = [
  { label: '全部', value: undefined },
  { label: '待充值', value: 1 },
  { label: '充值中', value: 2 },
  { label: '充值成功', value: 3 },
  { label: '充值失败', value: 4 },
  { label: '已取消', value: 5 }
];

// 表格数据与分页
const loading = ref(false);
const data = ref<any[]>([]);
const pagination = ref({ page: 1, pageSize: 10, itemCount: 0 });

// 表格列定义
const columns: DataTableColumns<any> = [
  {
    title: '订单号',
    key: 'yr_order_id',
    width: 220
  },
  {
    title: '账号',
    key: 'account',
    width: 120
  },
  {
    title: '面值',
    key: 'denom',
    width: 100
  },
  {
    title: '结算价',
    key: 'settlePrice',
    width: 100
  },
  {
    title: '创建时间',
    key: 'createTime',
    width: 180,
    render: (row) => {
      return formatDate(row.createTime)
    }
  },
  {
    title: '充值时间',
    key: 'chargeTime',
    width: 180,
    render: (row) => {
      return row.chargeTime ? formatDate(row.chargeTime) : '-'
    }
  },
  {
    title: '上报时间',
    key: 'uploadTime',
    width: 180,
    render: (row) => {
      return row.uploadTime ? formatDate(row.uploadTime) : '-'
    }
  },
  {
    title: '状态',
    key: 'status',
    width: 100,
    render: (row) => {
      return getStatusText(row.status)
    }
  },
  {
    title: '结算状态',
    key: 'settleStatus',
    width: 100,
    render: (row) => {
      return getSettleStatusText(row.settleStatus)
    }
  },
  {
    title: '运营商',
    key: 'yunying',
    width: 150
  },
  {
    title: '省份',
    key: 'prov',
    width: 100
  },
  { title: '操作', key: 'actions',
    render(row) {
      return h(NSpace, {}, [
        h(NButton, { size: 'small', type: 'primary', onClick: () => handleDetail(row) }, { default: () => '详情' })
      ]);
    }
  }
];

// 获取订单列表
const getOrderList = async () => {
  loading.value = true
  try {
    const params = {
      page: pagination.value.page,
      page_size: pagination.value.pageSize,
      yr_order_id: searchForm.value.yr_order_id || undefined,
      status: searchForm.value.status,
      start_time: searchForm.value.dateRange?.[0],
      end_time: searchForm.value.dateRange?.[1]
    }
    const { data: res } = await request<{
      list: any[];
      total: number;
    }>({
      url: '/daichong-order',
      method: 'get',
      params
    })
    if (res) {
      data.value = res.list
      pagination.value.itemCount = res.total
    }
  } catch (error) {
    console.error('获取订单列表失败:', error)
    message.error('获取订单列表失败')
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  pagination.value.page = 1 // 重置到第一页
  getOrderList()
}

function handleReset() {
  searchForm.value = {
    yr_order_id: '',
    status: undefined,
    dateRange: null
  }
  pagination.value.page = 1 // 重置到第一页
  getOrderList()
}

function handlePageChange(page: number) {
  pagination.value.page = page
  getOrderList()
}

function handleDetail(row: any) {
  // TODO: 跳转或弹窗显示订单详情
  window.$message?.info(`查看订单：${row.order_id}`);
}

// 渠道及运营商选择相关
const showModal = ref(false);
const loadingChannels = ref(false);
const channels = ref<Channel[]>([]);
const selected = ref<{ [channelId: number]: number[] }>({});
const faceValues = ref<{ [channelId: number]: string }>({});
const minSettleAmounts = ref<{ [channelId: number]: string }>({});
const message = useMessage();

function openChannelModal() {
  showModal.value = true;
  loadingChannels.value = true;
  getChannelList().then(res => {
    const list = Array.isArray(res.data) ? res.data : [];
    channels.value = list;
    const faceInit: { [channelId: number]: string } = {};
    const minSettleInit: { [channelId: number]: string } = {};
    list.forEach((c: Channel) => {
      faceInit[c.channelId] = '';
      minSettleInit[c.channelId] = '';
    });
    const selectedInit: { [channelId: number]: number[] } = {};
    list.forEach((c: Channel) => {
      selectedInit[c.channelId] = [];
    });
    selected.value = selectedInit;
    faceValues.value = faceInit;
    minSettleAmounts.value = minSettleInit;
  }).finally(() => {
    loadingChannels.value = false;
  });
}

function handleChannelChange(channelId: number, productIds: number[]) {
  selected.value[channelId] = productIds;
}

async function handleSave() {
  const payload = Object.entries(selected.value)
    .map(([cid, pids]) => {
      const productIds = (pids as number[]);
      return {
        channel_id: Number(cid),
        channel_name: channels.value.find(c => c.channelId === Number(cid))?.channelName || '',
        face_values: faceValues.value[Number(cid)] || '',
        min_settle_amounts: minSettleAmounts.value[Number(cid)] || '',
        product_id: productIds.join(','),
        product_name: productIds
          .map(pid => channels.value.find(c => c.channelId === Number(cid))?.productList.find(p => p.productId === pid)?.productName || '')
          .join(',')
      }
    })
    .filter(item => item.product_id);
  if (!payload.length) {
    message.warning('请选择渠道及运营商');
    return;
  }
  try {
    await request({ url: '/task-config', method: 'post', data: payload });
    message.success('写入成功');
    showModal.value = false;
    fetchConfigList();
  } catch (e: any) {
    message.error(e?.message || '写入失败');
  }
}

interface Channel {
  channelId: number;
  channelName: string;
  productList: Product[];
}
interface Product {
  productId: number;
  productName: string;
}

const showConfigModal = ref(false);
const configList = ref<any[]>([]);
const selectedConfigKeys = ref<number[]>([]);

function openConfigModal() {
  showConfigModal.value = true;
  fetchConfigList();
}

function handleAddConfig() {
  message.info('TODO: 打开新增配置表单');
}

const configColumns: DataTableColumns<any> = [
  { type: 'selection', width: 40 },
  { title: 'ID', key: 'id', width: 60 },
  { title: '渠道ID', key: 'channel_name', width: 80 },
  { title: '运营商ID', key: 'product_name', width: 80 },
  { title: '面值', key: 'face_values' },
  { title: '最低结算价', key: 'min_settle_amounts' },
  { 
    title: '状态', 
    key: 'status', 
    render(row) { 
      return h(NTag, {
        type: row.status === 1 ? 'success' : 'error',
        bordered: false
      }, { default: () => row.status === 1 ? '启用' : '禁用' })
    } 
  },
  { title: '创建时间', key: 'created_at', render(row) { return formatDateTime(row.created_at) } },
  {
    title: '操作', key: 'actions',
    render(row) {
      return h(NSpace, {}, [
        h(NButton, { size: 'small', type: 'primary', onClick: () => handleEdit(row) }, { default: () => '编辑' }),
        h(NPopconfirm, {
          onPositiveClick: () => handleDelete(row)
        }, {
          default: () => '确认删除该配置吗？',
          trigger: () => h(NButton, { size: 'small', type: 'error' }, { default: () => '删除' })
        })
      ]);
    }
  }
]

async function fetchConfigList() {
  loading.value = true;
  try {
    const res = await getTaskConfigList({ page: pagination.value.page, page_size: pagination.value.pageSize });
    if (res.data) {
      configList.value = res.data.list;
      pagination.value.itemCount = res.data.total;
    } else {
      message.error(res.data.error || '获取配置失败');
    }
  } finally {
    loading.value = false;
  }
}

function formatDateTime(val: string) {
  if (!val) return '';
  const date = new Date(val);
  if (isNaN(date.getTime())) return val;
  const pad = (n: number) => n.toString().padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`;
}

// 添加编辑相关的状态
const showEditModal = ref(false);
const editForm = ref({
  id: 0,
  channel_id: 0,
  product_id: 0,
  face_values: '',
  min_settle_amounts: '',
  status: 1
});

// 修改 handleEdit 方法
function handleEdit(row: any) {
  editForm.value = {
    id: row.id,
    channel_id: row.channel_id,
    product_id: row.product_id,
    face_values: row.face_values,
    min_settle_amounts: row.min_settle_amounts,
    status: row.status
  };
  showEditModal.value = true;
}

// 添加保存编辑的方法
async function handleSaveEdit() {
  try {
    await updateTaskConfig(editForm.value);
    message.success('更新成功');
    showEditModal.value = false;
    // 重新加载列表
    fetchConfigList();
  } catch (error: any) {
    message.error(error?.message || '更新失败');
  }
}

async function handleDelete(row: any) {
  try {
    await deleteTaskConfig(row.id);
    message.success('删除成功');
    // 重新加载列表
    fetchConfigList();
  } catch (error: any) {
    message.error(error?.message || '删除失败');
  }
}

// 格式化日期
const formatDate = (timestamp: number): string => {
  if (!timestamp) return '-'
  return new Date(timestamp).toLocaleString()
}

// 获取状态文本
const getStatusText = (status: number): string => {
  const statusMap: Record<number, string> = {
    1: '待充值',
    2: '充值中',
    3: '充值成功',
    4: '充值失败',
    5: '已取消'
  }
  return statusMap[status] || '未知状态'
}

// 获取结算状态文本
const getSettleStatusText = (status: number): string => {
  const statusMap: Record<number, string> = {
    0: '未结算',
    1: '已结算',
    2: '结算中'
  }
  return statusMap[status] || '未知状态'
}

// 监听分页变化
watch(
  () => [pagination.value.page, pagination.value.pageSize],
  () => {
    getOrderList()
  }
)

// 初始化加载
onMounted(() => {
  getOrderList()
})

async function batchSetStatus(status: number) {
  if (!selectedConfigKeys.value.length) return;
  try {
    for (const id of selectedConfigKeys.value) {
      await updateTaskConfig({ id, status });
    }
    message.success(status === 1 ? '批量开启成功' : '批量关闭成功');
    fetchConfigList();
  } catch (e: any) {
    message.error(e?.message || '批量操作失败');
  }
}
</script>

<template>
  <div class="flex flex-col gap-4">
    <!-- 功能操作区 -->
    <div class="flex justify-between items-center">
      <NCard title="订单查询" size="small" style="flex:1;">
        <NForm
          :model="searchForm"
          label-placement="left"
          label-width="auto"
          require-mark-placement="right-hanging"
        >
          <NSpace vertical>
            <NSpace>
              <NFormItem label="订单号" path="yr_order_id">
                <NInput v-model:value="searchForm.yr_order_id" placeholder="请输入订单号" />
              </NFormItem>
              <NFormItem label="订单状态" path="status">
                <NSelect
                  v-model:value="searchForm.status"
                  :options="statusOptions"
                  placeholder="请选择订单状态"
                  style="width: 200px"
                />
              </NFormItem>
              <NFormItem label="下单时间" path="dateRange">
                <NDatePicker
                  v-model:value="searchForm.dateRange"
                  type="daterange"
                  clearable
                  style="width: 300px"
                />
              </NFormItem>
            </NSpace>
            <NSpace>
              <NButton type="primary" @click="handleSearch">查询</NButton>
              <NButton @click="handleReset">重置</NButton>
              <!-- <NButton type="primary" @click="openChannelModal" style="margin-left: 16px;">获取渠道及运营商编码</NButton> -->
              <!-- <NButton type="primary" @click="openConfigModal" style="margin-left: 16px;">拉取订单配置</NButton> -->
            </NSpace>
          </NSpace>
        </NForm>
      </NCard>
    </div>

    <!-- 表格列表区 -->
    <NCard title="订单列表" size="small">
      <NDataTable :columns="columns" :data="data" :loading="loading" />
      <div class="flex justify-end mt-4">
        <NPagination
          v-model:page="pagination.page"
          :page-size="pagination.pageSize"
          :item-count="pagination.itemCount"
          @update:page="handlePageChange"
        />
      </div>
    </NCard>

    <!-- 渠道及运营商选择弹窗 -->
    <NModal
      v-model:show="showModal"
      title="选择渠道及运营商"
      preset="dialog"
      style="width: 500px;"
    >
      <NSpin :show="loadingChannels">
        <div v-for="channel in channels" :key="channel.channelId" style="margin-bottom: 16px;">
          <div style="font-weight: bold;">{{ channel.channelName }}</div>
          <NCheckboxGroup
            v-model:value="selected[channel.channelId]"
            @update:value="val => handleChannelChange(channel.channelId, val as number[])"
          >
            <NCheckbox
              v-for="product in channel.productList"
              :key="product.productId"
              :value="product.productId"
              :label="product.productName"
            />
          </NCheckboxGroup>
          <div class="flex items-center" style="margin-top: 8px;">
            <span style="width: 90px;">拉取面值</span>
            <div style="flex: 1;">
              <NInput
                v-model:value="faceValues[channel.channelId]"
                size="small"
                placeholder="50,100,200,500,1000"
                style="width: 200px;"
              />
              <div style="color: #888; font-size: 12px; margin-top: 2px;">
                支持多个，逗号隔开，最多5个面值，不要重复
              </div>
            </div>
          </div>
          <div class="flex items-center" style="margin-top: 8px;">
            <span style="width: 90px;">最低结算价格</span>
            <NInput
              v-model:value="minSettleAmounts[channel.channelId]"
              size="small"
              placeholder="最低结算价格"
              style="width: 200px;"
            />
            <div style="color: #888; font-size: 12px; margin-top: 2px;">
              支持多个,逗号隔开,faceValues对应
              </div>
          </div>  
        </div>
        <div style="text-align: right;">
          <NButton type="primary" @click="handleSave">确定</NButton>
          <NButton @click="showModal = false" style="margin-left: 8px;">取消</NButton>
        </div>
      </NSpin>
    </NModal>

    <n-modal v-model:show="showConfigModal" title="订单配置" preset="dialog" style="width: 900px;">
      <template #header>
        <div style="display: flex; align-items: center; width: 100%; box-sizing: border-box;">
          <span style="flex: 1;">订单配置</span>
          <NButton type="primary" @click="openChannelModal">增加配置</NButton>
          <NButton
            type="success"
            style="margin-left: 8px;"
            :disabled="selectedConfigKeys.length === 0"
            @click="batchSetStatus(1)"
          >批量开启</NButton>
          <NButton
            type="error"
            style="margin-left: 8px;"
            :disabled="selectedConfigKeys.length === 0"
            @click="batchSetStatus(0)"
          >批量关闭</NButton>
        </div>
      </template>
      <n-data-table
        :columns="configColumns"
        :data="configList"
        :pagination="pagination"
        :loading="loading"
        :row-key="row => row.id"
        v-model:checked-row-keys="selectedConfigKeys"
        style="margin-top: 16px;"
      />
    </n-modal>

    <!-- 添加编辑弹窗 -->
    <n-modal v-model:show="showEditModal" title="编辑配置" preset="dialog" style="width: 500px;">
      <n-form
        :model="editForm"
        label-placement="left"
        label-width="auto"
        require-mark-placement="right-hanging"
      >
        <n-form-item label="渠道ID" path="channel_id">
          <n-input-number v-model:value="editForm.channel_id" :min="1" disabled />
        </n-form-item>
        <n-form-item label="运营商ID" path="product_id">
          <n-input-number v-model:value="editForm.product_id" :min="0" />
        </n-form-item>
        <n-form-item label="面值" path="face_values">
          <n-input v-model:value="editForm.face_values" placeholder="50,100,200" />
        </n-form-item>
        <n-form-item label="最低结算价" path="min_settle_amounts">
          <n-input v-model:value="editForm.min_settle_amounts" placeholder="49.5,99,198" />
        </n-form-item>
        <n-form-item label="状态" path="status">
          <n-select
            v-model:value="editForm.status"
            :options="[
              { label: '启用', value: 1 },
              { label: '禁用', value: 0 }
            ]"
          />
        </n-form-item>
      </n-form>
      <template #action>
        <n-space>
          <n-button @click="showEditModal = false">取消</n-button>
          <n-button type="primary" @click="handleSaveEdit">确定</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<style scoped>
.flex {
  display: flex;
}
.flex-col {
  flex-direction: column;
}
.gap-4 {
  gap: 1rem;
}
.justify-end {
  justify-content: flex-end;
}
.mt-4 {
  margin-top: 1rem;
}
.justify-between {
  justify-content: space-between;
}
.items-center {
  align-items: center;
}
</style> 