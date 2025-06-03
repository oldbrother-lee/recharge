<script setup lang="tsx">
import { ref, onMounted, h } from 'vue';
import { useMessage, NButton, NPopconfirm, NModal, NDropdown, useDialog } from 'naive-ui';
import { request } from '@/service/request';
import { useAppStore } from '@/store/modules/app';
import UserSearch from './modules/user-search.vue';
import UserOperateDrawer from './modules/user-operate-drawer.vue';
import UserRechargeModal from './modules/user-recharge-modal.vue';
import UserDeductModal from './modules/user-deduct-modal.vue';
import UserCreditModal from './modules/user-credit-modal.vue';
import UserRoleModal from './modules/user-role-modal.vue';
import type { DataTableColumns } from 'naive-ui';

const appStore = useAppStore();
const message = useMessage();
const dialog = useDialog();

// 用户数据
const loading = ref(false);
const data = ref([]);
const pagination = ref({ page: 1, pageSize: 10, itemCount: 0 });
const selectedRowKeys = ref<number[]>([]);

// 搜索参数
const searchParams = ref({
  user_name: '',
  phone: '',
  email: '',
  status: null,
  balance_min: null,
  balance_max: null
});

// 弹窗/抽屉状态
const operateDrawerVisible = ref(false);
const operateType = ref<'add' | 'edit'>('add');
const editingUser = ref<any>(null);

const rechargeModalVisible = ref(false);
const rechargeUser = ref<any>(null);

const deductModalVisible = ref(false);
const deductUser = ref<any>(null);

const creditModalVisible = ref(false);
const creditUser = ref<any>(null);

const resetPasswordModalVisible = ref(false);
const resetPasswordUser = ref<any>(null);
const newPassword = ref('');

const roleModalVisible = ref(false);
const roleUser = ref<any>(null);

// 表格列
const columns: DataTableColumns<any> = [
  {
    type: 'selection' as const,
    align: 'center',
    width: 48
  },
  {
    key: 'username',
    title: '用户名',
    align: 'center',
    minWidth: 100
  },
  {
    key: 'balance',
    title: '余额',
    align: 'center',
    width: 120,
    render(row: any) {
      return `¥${row.balance?.toFixed(2) || '0.00'}`;
    }
  },
  {
    key: 'credit',
    title: '授信额度',
    align: 'center',
    width: 120,
    render(row: any) {
      return `¥${row.credit?.toFixed(2) || '0.00'}`;
    }
  },
  {
    key: 'credit_used',
    title: '已用授信',
    align: 'center',
    width: 120,
    render(row: any) {
      return `¥${row.credit_used?.toFixed(2) || '0.00'}`;
    }
  },
  {
    key: 'nickname',
    title: '昵称',
    align: 'center',
    minWidth: 100
  },
  {
    key: 'email',
    title: '邮箱',
    align: 'center',
    minWidth: 200
  },
  {
    key: 'phone',
    title: '手机号',
    align: 'center',
    width: 120
  },
  {
    key: 'status',
    title: '状态',
    align: 'center',
    width: 80,
    render(row: any) {
      return row.status === 1 ? '正常' : '禁用';
    }
  },
  {
    key: 'created_at',
    title: '创建时间',
    align: 'center',
    width: 180,
    render(row: any) {
      return new Date(row.created_at).toLocaleString();
    }
  },
  {
    key: 'operate',
    title: '操作',
    align: 'center',
    width: 340,
    render: (row: any) => {
      const dropdownOptions = [
        { label: '分配角色', key: 'assignRole' },
        { label: '扣款', key: 'deduct' },
        { label: '授信', key: 'credit' },
        { label: '重置密码', key: 'resetPassword' },
        { label: '删除', key: 'delete', type: 'danger' }
      ];
      const handleDropdownSelect = (key: string) => {
        if (key === 'assignRole') onAssignRole(row);
        else if (key === 'deduct') onDeduct(row);
        else if (key === 'credit') onCredit(row);
        else if (key === 'resetPassword') onResetPassword(row);
        else if (key === 'delete') {
          dialog.warning({
            title: '确认删除？',
            content: '此操作不可恢复，是否继续？',
            positiveText: '确认',
            negativeText: '取消',
            onPositiveClick: () => onDelete(row)
          });
        }
      };
      return (
        <div class="flex-center gap-8px">
          <NButton type="primary" ghost size="small" onClick={() => onEdit(row)}>编辑</NButton>
          <NButton type="success" ghost size="small" onClick={() => onRecharge(row)}>充值</NButton>
          <NDropdown
            trigger="click"
            options={dropdownOptions}
            onSelect={handleDropdownSelect}
          >
            <NButton type="default" ghost size="small">更多</NButton>
          </NDropdown>
        </div>
      );
    }
  }
];

// 获取用户列表
async function fetchUsers() {
  try {
    loading.value = true;
    const params = {
      page: pagination.value.page,
      pageSize: pagination.value.pageSize,
      ...searchParams.value
    };
    const res = await request({
      url: '/users',
      method: 'GET',
      params
    });
    if (res.data) {
      data.value = res.data.list;
      pagination.value.itemCount = res.data.total;
    }
  } catch (error) {
    message.error('获取用户列表失败');
  } finally {
    loading.value = false;
  }
}

function onAdd() {
  operateType.value = 'add';
  editingUser.value = null;
  operateDrawerVisible.value = true;
}
function onEdit(row: any) {
  operateType.value = 'edit';
  editingUser.value = { ...row };
  operateDrawerVisible.value = true;
}
function onRecharge(row: any) {
  rechargeUser.value = { ...row };
  rechargeModalVisible.value = true;
}
function onDeduct(row: any) {
  deductUser.value = { ...row };
  deductModalVisible.value = true;
}
function onCredit(row: any) {
  creditUser.value = { ...row };
  creditModalVisible.value = true;
}
async function onDelete(row: any) {
  try {
    await request({ url: `/users/${row.id}`, method: 'DELETE' });
    message.success('删除成功');
    fetchUsers();
  } catch (error) {
    message.error('删除失败');
  }
}
function onResetPassword(row: any) {
  resetPasswordUser.value = { ...row };
  resetPasswordModalVisible.value = true;
}

async function handleResetPassword() {
  if (newPassword.value) {
    try {
      await request({ url: `/users/${resetPasswordUser.value.id}/reset-password`, method: 'POST', data: { newPassword: newPassword.value } });
      message.success('重置密码成功');
      resetPasswordModalVisible.value = false;
      newPassword.value = '';
    } catch (error) {
      message.error('重置密码失败');
    }
  }
}

function handlePageChange(page: number) {
  pagination.value.page = page;
  fetchUsers();
}
function handlePageSizeChange(pageSize: number) {
  pagination.value.pageSize = pageSize;
  fetchUsers();
}
function handleSearch() {
  pagination.value.page = 1;
  fetchUsers();
}
function handleReset() {
  searchParams.value = {
    user_name: '',
    phone: '',
    email: '',
    status: null,
    balance_min: null,
    balance_max: null
  };
  fetchUsers();
}

function onAssignRole(row: any) {
  roleUser.value = row;
  roleModalVisible.value = true;
}

onMounted(() => {
  fetchUsers();
});
</script>

<template>
  <div class="min-h-500px flex-col-stretch gap-16px overflow-hidden lt-sm:overflow-auto">
    <UserSearch v-model:model="searchParams" @search="handleSearch" @reset="handleReset" />
    <n-card :title="'用户管理'" :bordered="false" size="small" class="card-wrapper">
      <template #header-extra>
        <n-button type="primary" @click="onAdd">新增用户</n-button>
      </template>
      <n-data-table
        :columns="columns"
        :data="data"
        :loading="loading"
        :pagination="pagination"
        :flex-height="!appStore.isMobile"
        :scroll-x="962"
        remote
        :row-key="row => row.id"
        @update:page="handlePageChange"
        @update:page-size="handlePageSizeChange"
        v-model:checked-row-keys="selectedRowKeys"
        class="sm:h-full"
      />
    </n-card>
    <UserOperateDrawer
      v-model:visible="operateDrawerVisible"
      :operate-type="operateType"
      :row-data="editingUser"
      @submitted="fetchUsers"
    />
    <UserRechargeModal
      v-model:visible="rechargeModalVisible"
      :user="rechargeUser"
      @submitted="fetchUsers"
    />
    <UserDeductModal
      v-model:visible="deductModalVisible"
      :user="deductUser"
      @submitted="fetchUsers"
    />
    <UserCreditModal
      v-model:visible="creditModalVisible"
      :user="creditUser"
      @submitted="fetchUsers"
    />
    <UserRoleModal
      v-model:visible="roleModalVisible"
      :user="roleUser"
      @success="fetchUsers"
    />
    <n-modal
      v-model:show="resetPasswordModalVisible"
      title="重置密码"
      preset="dialog"
      positive-text="确定"
      negative-text="取消"
      @positive-click="handleResetPassword"
      @negative-click="() => { resetPasswordModalVisible = false; newPassword.value = '' }"
    >
      <div style="margin-bottom: 12px;">
        <n-input
          v-model:value="newPassword"
          type="password"
          placeholder="请输入新密码"
          style="width: 300px"
          maxlength="32"
          show-password-on="click"
        />
      </div>
    </n-modal>
  </div>
</template>

<style scoped>
.card-wrapper {
  flex: 1 1 auto;
}
</style>
