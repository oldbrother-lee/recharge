<script setup lang="tsx">
import { computed, onMounted, ref } from 'vue';
import { NButton, NCard, NDataTable, NForm, NFormItem, NSelect, NSpace, NTag, useMessage } from 'naive-ui';
import type { DataTableColumns, SelectOption } from 'naive-ui';
import { getBalanceLogs } from '@/api/balance';
import { getUserList } from '@/api/user';
import { ORDER_STATUS_MAP } from '@/constants/business';
import { useAppStore } from '@/store/modules/app';

defineOptions({ name: 'SystemBalanceLog' });

const appStore = useAppStore();
const message = useMessage();
const loading = ref(false);
const userId = ref<number | null>(null);
const userOptions = ref<SelectOption[]>([]);
const userLoading = ref(false);
const logs = ref<any[]>([]);
const pagination = ref({
  page: 1,
  pageSize: 20,
  itemCount: 0,
  pageSizes: [10, 20, 50, 100],
  showSizePicker: true,
  prefix: (info: { itemCount?: number }) => `共 ${Number(info.itemCount ?? 0)} 条`
});

const responsivePagination = computed(() => ({
  ...pagination.value,
  showSizePicker: true,
  pageSlot: appStore.isMobile ? 3 : 9,
  prefix: appStore.isMobile ? undefined : pagination.value.prefix
}));

const hasSearched = computed(() => logs.value.length > 0 || (userId.value != null && pagination.value.itemCount === 0));

const typeMap: Record<number, { type: 'success' | 'error' | 'default'; text: string }> = {
  1: { type: 'success', text: '收入' },
  2: { type: 'error', text: '支出' }
};

const styleMap: Record<number, string> = {
  1: '订单扣款',
  2: '退款',
  3: '手动调整',
  4: '充值',
  5: '授信扣款',
  6: '授信恢复',
  7: '授信调整'
};

const fundSourceMap: Record<string, string> = {
  balance: '余额',
  credit: '授信'
};

function renderFullText(value: string | undefined | null) {
  const text = value?.trim() ? value : '-';
  return <span class="cell-full-text">{text}</span>;
}

const columns: DataTableColumns<any> = [
  { key: 'id', title: 'ID', width: 70, align: 'center' },
  {
    key: 'order_number',
    title: '订单号',
    align: 'left',
    minWidth: 200,
    render(row) {
      return renderFullText(row.order_number);
    }
  },
  {
    key: 'out_trade_num',
    title: '外部订单号',
    align: 'left',
    minWidth: 200,
    render(row) {
      return renderFullText(row.out_trade_num);
    }
  },
  {
    key: 'order_status',
    title: '订单状态',
    width: 100,
    align: 'center',
    render(row) {
      if (!row.order_id) {
        return '-';
      }
      const info = ORDER_STATUS_MAP[row.order_status] || { type: 'default' as const, text: `状态${row.order_status}` };
      return <NTag type={info.type}>{info.text}</NTag>;
    }
  },
  { key: 'mobile', title: '手机号', align: 'center', width: 120 },
  {
    key: 'type',
    title: '类型',
    width: 80,
    align: 'center',
    render(row) {
      const info = typeMap[row.type] || { type: 'default', text: String(row.type) };
      return <NTag type={info.type}>{info.text}</NTag>;
    }
  },
  {
    key: 'fund_source',
    title: '资金类型',
    width: 90,
    align: 'center',
    render(row) {
      const label = fundSourceMap[row.fund_source] ?? row.fund_source ?? '余额';
      return <NTag type={row.fund_source === 'credit' ? 'warning' : 'info'}>{label}</NTag>;
    }
  },
  {
    key: 'style',
    title: '方式',
    width: 100,
    align: 'center',
    render(row) {
      return styleMap[row.style] ?? String(row.style);
    }
  },
  {
    key: 'amount',
    title: '金额',
    width: 110,
    align: 'right',
    render(row) {
      const amt = Number(row.amount);
      const prefix = amt >= 0 ? '+' : '';
      return (
        <span class={amt >= 0 ? 'text-green-600' : 'text-red-600'}>
          {prefix}¥{amt.toFixed(2)}
        </span>
      );
    }
  },
  {
    key: 'balance_before',
    title: '变动前',
    width: 110,
    align: 'right',
    render(row) {
      const prefix = row.fund_source === 'credit' ? '授信 ' : '';
      return `${prefix}¥${Number(row.balance_before || 0).toFixed(2)}`;
    }
  },
  {
    key: 'balance',
    title: '变动后',
    width: 110,
    align: 'right',
    render(row) {
      const prefix = row.fund_source === 'credit' ? '授信 ' : '';
      return `${prefix}¥${Number(row.balance || 0).toFixed(2)}`;
    }
  },
  { key: 'remark', title: '备注', ellipsis: { tooltip: true }, minWidth: 140 },
  { key: 'operator', title: '操作人', width: 90, align: 'center' },
  {
    key: 'created_at',
    title: '时间',
    width: 170,
    align: 'center',
    render(row) {
      const t = row.created_at;
      if (!t) return '-';
      const d = new Date(t);
      const pad = (n: number) => String(n).padStart(2, '0');
      return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
    }
  }
];

async function loadUserOptions() {
  userLoading.value = true;
  try {
    const resUser = await getUserList({
      page: 1,
      page_size: 5000
    });
    const list = Array.isArray(resUser.data?.list) ? resUser.data.list : [];
    userOptions.value = list.map((u: any) => ({
      label: u.username || u.nickname || String(u.id),
      value: u.id
    }));
  } catch (e: any) {
    message.error(e?.message || '获取用户列表失败');
  } finally {
    userLoading.value = false;
  }
}

onMounted(() => {
  loadUserOptions();
});

async function fetchLogs() {
  if (userId.value == null || userId.value <= 0) {
    message.warning('请选择用户');
    return;
  }
  loading.value = true;
  try {
    const res = await getBalanceLogs({
      user_id: userId.value,
      page: pagination.value.page,
      page_size: pagination.value.pageSize
    });
    logs.value = Array.isArray(res.data?.list) ? res.data.list : [];
    pagination.value.itemCount = Number(res.data?.total ?? 0);
  } catch (e: any) {
    message.error(e?.message || '获取流水失败');
    logs.value = [];
    pagination.value.itemCount = 0;
  } finally {
    loading.value = false;
  }
}

function handleSearch() {
  pagination.value.page = 1;
  fetchLogs();
}

function handlePageChange(page: number) {
  pagination.value.page = page;
  fetchLogs();
}

function handlePageSizeChange(size: number) {
  pagination.value.pageSize = size;
  pagination.value.page = 1;
  fetchLogs();
}
</script>

<template>
  <div class="min-h-500px flex-col-stretch gap-16px overflow-hidden lt-sm:overflow-auto">
    <NCard title="资金流水（余额 + 授信）" :bordered="false" size="small" class="sm:flex-1-hidden card-wrapper">
      <NForm label-placement="left" :label-width="80" class="mb-4">
        <NSpace align="center" wrap>
          <NFormItem label="用户" required>
            <NSelect
              v-model:value="userId"
              placeholder="选择用户"
              filterable
              clearable
              :options="userOptions"
              :loading="userLoading"
              style="width: 260px"
            />
          </NFormItem>
          <NButton type="primary" :loading="loading" @click="handleSearch">查询流水</NButton>
        </NSpace>
      </NForm>
      <div class="balance-log-table-wrap">
        <NDataTable
          :columns="columns"
          :data="logs"
          :loading="loading"
          :pagination="responsivePagination"
          :flex-height="!appStore.isMobile"
          :scroll-x="1680"
          remote
          size="small"
          :row-key="(row: any) => row.id"
          class="sm:h-full"
          @update:page="handlePageChange"
          @update:page-size="handlePageSizeChange"
        />
      </div>
      <div v-if="hasSearched && !loading && logs.length === 0" class="py-8 text-center text-gray-500">
        该用户暂无流水记录
      </div>
    </NCard>
  </div>
</template>

<style scoped>
.card-wrapper {
  display: flex;
  flex-direction: column;
  height: 100%;
}
.card-wrapper :deep(.n-card__content) {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  overflow: hidden;
}
.balance-log-table-wrap {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.text-green-600 {
  color: #16a34a;
}
:deep(.cell-full-text) {
  display: inline-block;
  max-width: 100%;
  white-space: normal;
  word-break: break-all;
  line-height: 1.4;
}
:deep(.n-data-table-td) {
  vertical-align: top;
}
.text-red-600 {
  color: #dc2626;
}
</style>
