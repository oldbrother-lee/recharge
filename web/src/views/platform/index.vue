<script setup lang="tsx">
import { computed, h, nextTick, onMounted, ref, watch } from 'vue';
import {
  NButton,
  NCard,
  NDataTable,
  NDropdown,
  NForm,
  NFormItem,
  NFormItemGi,
  NGrid,
  NGridItem,
  NIcon,
  NInput,
  NModal,
  NPopconfirm,
  NSelect,
  NSpace,
  NStatistic,
  NSwitch,
  NTag,
  useMessage
} from 'naive-ui';
import type { DataTableColumns } from 'naive-ui';
import { request } from '@/service/request';
import { getChannelList } from '@/api/platform';
import { queryPlatformBalance } from '@/api/balance';
import { useAppStore } from '@/store/modules/app';
import { useTable } from '@/hooks/useTable';
import { useModal } from '@/hooks/useModal';
import { useForm } from '@/hooks/useForm';
import type { ApiResponse } from '@/types/api';
import PlatformAccountForm from './components/PlatformAccountForm.vue';
import BeeProductManagement from './components/BeeProductManagement.vue';
import ProductPriceForm from './components/ProductPriceForm.vue';
// 复制任务配置接口方法，前端内置两套：得众与闲赚侠
// 得众平台：平台账号变体接口
function getTaskConfigList(params: { page: number; page_size: number; platform_account_id?: number }) {
  return request({ url: '/platform/account/variants', method: 'GET', params });
}
function deleteTaskConfig(id: number) {
  return request({ url: `/platform/account/variants/${id}`, method: 'DELETE' });
}
function createTaskConfig(data: any) {
  return request({ url: '/platform/account/variants', method: 'POST', data });
}
function updateTaskConfig(data: any) {
  const id = (data && data.id) || 0;
  return request({ url: `/platform/account/variants/${id}`, method: 'PUT', data });
}
// 闲赚侠平台：原有任务配置接口
function getXianzhuanxiaTaskConfigList(params: { page: number; page_size: number; platform_account_id?: number }) {
  return request({ url: '/task-config', method: 'GET', params });
}
function deleteXianzhuanxiaTaskConfig(id: number) {
  return request({ url: `/task-config/${id}`, method: 'DELETE' });
}
function createXianzhuanxiaTaskConfig(data: any) {
  return request({ url: '/task-config', method: 'POST', data });
}
function updateXianzhuanxiaTaskConfig(data: any) {
  return request({ url: '/task-config', method: 'PUT', data });
}

interface Platform {
  id: number;
  name: string;
  code: string;
  api_url: string;
  description: string;
  status: number;
  created_at: string;
}

interface PlatformAccount {
  id: number;
  platform_id: number;
  platform_code: string;
  account_name: string;
  type: number;
  app_key: string;
  app_secret: string;
  account_password: string;
  description: string;
  status: number;
  created_at: string;
  bind_user_id?: number;
  bind_user_name?: string;
  enable_pull_order?: boolean;
  push_status: number;
  max_concurrency?: number;
  poll_interval_sec?: number;
}

interface TaskConfig {
  id: number;
  channel_id: number;
  channel_name: string;
  product_id: string;
  product_name: string;
  face_values: string;
  min_settle_amounts: string;
  status: number;
  create_time: string;
  platform_id: number;
  platform_name: string;
  platform_account_id: number;
  platform_account: string;
}

interface TaskConfigListParams {
  page: number;
  page_size: number;
  platform_account_id?: number;
}

interface TaskConfigListData {
  list: TaskConfig[];
  total: number;
}

const appStore = useAppStore();
const message = useMessage();
const { loading, data, pagination, handlePageChange, handlePageSizeChange, handleSearch } = useTable<Platform>();
const { visible, showModal, hideModal } = useModal();
const { formRef, formModel, rules, handleSubmit, resetForm } = useForm();
const currentPlatformCode = ref('');
const beeProductManagementRef = ref();

// 添加 computed 属性
  const isXianzhuanxia = computed(() => {
    console.log('Computing isXianzhuanxia:', currentPlatformCode.value);
    return currentPlatformCode.value === 'xianzhuanxia';
  });

  const isMf178 = computed(() => {
    return currentPlatformCode.value === 'mifeng';
  });

  // 弹窗标题：闲赚侠使用原有文案，其它平台独立显示
  function getTaskConfigModalTitle() {
    return isXianzhuanxia.value ? '拉取订单配置' : '任务配置';
  }

const isDz = computed(() => {
  return currentPlatformCode.value === 'dz';
});

// 账号相关状态
const accountVisible = ref(false);
const accountFormRef = ref();
const accountData = ref<PlatformAccount[]>([]);
const accountLoading = ref(false);
const accountPagination = ref({
  page: 1,
  pageSize: 10,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 30, 40],
  onChange: (page: number) => {
    accountPagination.value.page = page;
    if (accountData.value.length > 0) {
      fetchPlatformAccounts(accountData.value[0].platform_id, currentPlatformCode.value);
    }
  },
  onUpdatePageSize: (pageSize: number) => {
    accountPagination.value.pageSize = pageSize;
    accountPagination.value.page = 1;
    if (accountData.value.length > 0) {
      fetchPlatformAccounts(accountData.value[0].platform_id, currentPlatformCode.value);
    }
  }
});

// 新增绑定账号弹窗相关状态
const bindUserDialogVisible = ref(false);
const bindUserLoading = ref(false);
const selectedUserId = ref<number | null>(null);
const userOptions = ref<{ label: string; value: number }[]>([
  { label: 'admin', value: 1 },
  { label: 'test2', value: 3 }
]);
const currentPlatformAccount = ref<PlatformAccount | null>(null);

// 多选绑定
const selectedAccountIds = ref<number[]>([]);

// 表格列定义
const columns: DataTableColumns<Platform> = [
  {
    type: 'selection',
    align: 'center',
    width: 48
  },
  {
    key: 'name',
    title: '平台名称',
    align: 'center',
    width: 120
  },
  {
    key: 'code',
    title: '平台代码',
    align: 'center',
    width: 120
  },
  {
    key: 'api_url',
    title: 'API地址',
    align: 'center',
    width: 200
  },
  {
    key: 'status',
    title: '状态',
    align: 'center',
    width: 80,
    render(row: Platform) {
      return row.status === 1 ? '启用' : '禁用';
    }
  },
  {
    key: 'created_at',
    title: '创建时间',
    align: 'center',
    width: 180,
    render(row: Platform) {
      return new Date(row.created_at).toLocaleString();
    }
  },
  {
    key: 'operate',
    title: '操作',
    align: 'center',
    width: 200,
    render(row: Platform) {
      return (
        <div class="flex-center gap-8px">
          <NButton type="primary" ghost size="small" onClick={() => handleEdit(row)}>
            编辑
          </NButton>
          <NButton type="info" ghost size="small" onClick={() => showAccountDialog(row)}>
            账号管理
          </NButton>
          <NPopconfirm onPositiveClick={() => handleDelete(row)}>
            {{
              default: () => '确认删除？',
              trigger: () => (
                <NButton type="error" ghost size="small">
                  删除
                </NButton>
              )
            }}
          </NPopconfirm>
        </div>
      );
    }
  }
];

// 搜索表单
const searchForm = ref({
  name: '',
  code: '',
  status: null as number | null
});

// 获取平台列表
const fetchPlatforms = async () => {
  try {
    loading.value = true;
    const { page, pageSize } = pagination.value;

    // 过滤掉空值参数
    const searchParams = Object.fromEntries(
      Object.entries(searchForm.value).filter(([_, value]) => {
        if (value === null || value === undefined) return false;
        if (typeof value === 'string' && value.trim() === '') return false;
        return true;
      })
    );

    const params = {
      page,
      page_size: pageSize,
      ...searchParams
    };

    const res = await request({
      url: '/platform/list',
      method: 'GET',
      params
    });
    if (res.data) {
      data.value = res.data.list;
      pagination.value.itemCount = res.data.total;
    }
  } catch (error) {
    message.error('获取平台列表失败');
  } finally {
    loading.value = false;
  }
};

// 编辑平台
const handleEdit = (row: Platform) => {
  formModel.value = { ...row };
  showModal();
};

// 删除平台
const handleDelete = async (row: Platform) => {
  try {
    await request({
      url: `/platform/${row.id}`,
      method: 'DELETE'
    });
    message.success('删除成功');
    fetchPlatforms();
  } catch (error) {
    message.error('删除失败');
  }
};

// 提交表单
const handleFormSubmit = async () => {
  try {
    const isEdit = Boolean(formModel.value.id);
    // 新增与编辑均不进行表单校验，直接提交
    if (isEdit) {
      await request({
        url: `/platform/${formModel.value.id}`,
        method: 'PUT',
        data: formModel.value
      });
      message.success('更新成功');
    } else {
      await request({
        url: '/platform',
        method: 'POST',
        data: formModel.value
      });
      message.success('创建成功');
    }
    hideModal();
    fetchPlatforms();
  } catch (error) {
    message.error('操作失败');
  }
};

// 重置搜索表单
const handleReset = () => {
  searchForm.value = {
    name: '',
    code: '',
    status: null
  };
  fetchPlatforms();
};

// 添加这些处理函数
const onPageChange = (page: number) => {
  pagination.value.page = page;
  fetchPlatforms();
};

const onPageSizeChange = (pageSize: number) => {
  pagination.value.pageSize = pageSize;
  pagination.value.page = 1;
  fetchPlatforms();
};

// 获取平台账号列表
const fetchPlatformAccounts = async (platformId: number, code: string) => {
  try {
    accountLoading.value = true;
    currentPlatformCode.value = code;
    const { page, pageSize } = accountPagination.value;
    const res = await request({
      url: '/platform/account/list',
      method: 'GET',
      params: {
        platform_id: platformId,
        page,
        page_size: pageSize
      }
    });

    if (res.data) {
      console.log('Account data received:', res.data);
      const items = Array.isArray(res.data.items) ? res.data.items : [];
      accountData.value = items.map((item: PlatformAccount) => ({
        ...item,
        platform_id: platformId
      }));
      accountPagination.value.itemCount = res.data.total || 0;
    }
  } catch (error) {
    console.error('获取账号列表失败:', error);
    message.error('获取平台账号列表失败');
  } finally {
    accountLoading.value = false;
  }
};

// 显示账号管理对话框
const showAccountDialog = (platform: Platform) => {
  console.log('Opening account dialog for platform:', platform.code);
  console.log('Full platform object:', JSON.stringify(platform, null, 2));
  console.log('Current platform code before set:', currentPlatformCode.value);
  currentPlatformCode.value = platform.code;
  console.log('Current platform code after set:', currentPlatformCode.value);
  accountVisible.value = true;
  // 重置分页
  accountPagination.value.page = 1;
  // 获取账号列表
  fetchPlatformAccounts(platform.id, platform.code);
};

// 账号表格列定义
const accountColumns: DataTableColumns<PlatformAccount> = [
  {
    type: 'selection',
    align: 'center' as const,
    width: 48
  },
  {
    key: 'account_name',
    title: '账号名称',
    align: 'center' as const,
    width: 80
  },
  {
    key: 'type',
    title: '账号类型',
    align: 'center' as const,
    width: 80,
    render(row: PlatformAccount) {
      return row.type === 1 ? '测试账号' : '正式账号';
    }
  },
  {
    key: 'app_key',
    title: 'AppKey',
    align: 'center' as const,
    width: 100
  },
  {
    key: 'status',
    title: '状态',
    align: 'center' as const,
    width: 80,
    render(row: PlatformAccount) {
      return row.status === 1 ? '启用' : '禁用';
    }
  },
  {
    key: 'push_status',
    title: '推单状态',
    align: 'center' as const,
    width: 100,
      render(row: PlatformAccount) {
        if (row.push_status === 1) return '推单模式';
        if (row.enable_pull_order === true || row.push_status === 0) return '拉单模式';
        return '-';
      }
    },
  {
    key: 'bind_user_name',
    title: '绑定账号',
    align: 'center' as const,
    width: 120,
    render(row: PlatformAccount) {
      return row.bind_user_name || '未绑定';
    }
  },
  {
    key: 'operate',
    title: '操作',
    align: 'center' as const,
    width: 200,
    render(row: PlatformAccount) {
      // 构建下拉菜单选项
      const dropdownOptions = [];

      // 添加条件性按钮到下拉菜单
      if (isXianzhuanxia.value || isDz.value) {
        dropdownOptions.push({
          label: '拉单配置',
          key: 'taskConfig',
          props: {
            onClick: () => {
              console.log('Platform code:', currentPlatformCode.value);
              console.log('Is equal to xianzhuanxia:', isXianzhuanxia.value);
              console.log('Is equal to dz:', isDz.value);
              handleTaskConfig(row);
            }
          }
        });
      }

      if (isMf178.value) {
        dropdownOptions.push(
          {
            label: '商品管理',
            key: 'productManagement',
            props: {
              onClick: () => handleBeeProductManagement(row)
            }
          },
          {
            label: '省份配置',
            key: 'provinceConfig',
            props: {
              onClick: () => handleBeeProvinceConfig(row)
            }
          }
        );
      }

      dropdownOptions.push(
        {
          label: row.bind_user_id ? '更换绑定' : '绑定账号',
          key: 'bindUser',
          props: {
            onClick: () => handleBindUser(row)
          }
        },
        {
          label: '删除',
          key: 'delete',
          props: {
            onClick: () => {
              // 使用确认对话框
              if (window.confirm('确认删除？')) {
                handleDeleteAccount(row);
              }
            }
          }
        }
      );

      return (
        <div class="flex-center gap-8px">
          <NButton type="primary" ghost size="small" onClick={() => handleQueryBalance(row)}>
            查询余额
          </NButton>
          <NButton type="primary" ghost size="small" onClick={() => handleViewOrderStatistics(row)}>
            查看订单
          </NButton>
          <NButton type="primary" ghost size="small" onClick={() => accountFormRef.value?.edit(row)}>
            编辑
          </NButton>
          {dropdownOptions.length > 0 && (
            <NDropdown
              trigger="click"
              options={dropdownOptions}
              onSelect={key => {
                // 选项的点击事件已在props中定义
              }}
            >
              <NButton type="default" ghost size="small">
                更多
              </NButton>
            </NDropdown>
          )}
        </div>
      );
    }
  }
];

// 删除平台账号
const handleDeleteAccount = async (row: PlatformAccount) => {
  try {
    await request({
      url: `/platform/account/${row.id}`,
      method: 'DELETE'
    });
    message.success('删除成功');
    fetchPlatformAccounts(row.platform_id, currentPlatformCode.value);
  } catch (error) {
    message.error('删除失败');
  }
};

// 账号分页变化
const onAccountPageChange = (page: number) => {
  accountPagination.value.page = page;
  if (accountData.value.length > 0) {
    fetchPlatformAccounts(accountData.value[0].platform_id, currentPlatformCode.value);
  }
};

const onAccountPageSizeChange = (pageSize: number) => {
  accountPagination.value.pageSize = pageSize;
  accountPagination.value.page = 1;
  if (accountData.value.length > 0) {
    fetchPlatformAccounts(accountData.value[0].platform_id, currentPlatformCode.value);
  }
};

// 账号表单提交成功
const handleAccountSuccess = () => {
  if (accountData.value.length > 0) {
    fetchPlatformAccounts(accountData.value[0].platform_id, currentPlatformCode.value);
  }
};

// 批量开启/关闭推单
async function batchUpdatePushStatus(status: number) {
  if (!selectedAccountIds.value.length) {
    message.warning('请先选择账号');
    return;
  }
  const results = await Promise.allSettled(
    selectedAccountIds.value.map(id =>
      request({
        url: `/platform/push-status/${id}`,
        method: 'PUT',
        data: { status }
      })
    )
  );
  const successCount = results.filter(r => r.status === 'fulfilled').length;
  const failCount = results.length - successCount;
  message.success(`操作完成，成功${successCount}个，失败${failCount}个`);
  // 重新拉取账号列表
  if (accountData.value.length > 0) {
    fetchPlatformAccounts(accountData.value[0].platform_id, currentPlatformCode.value);
  }
}

// 打开绑定账号弹窗并拉取用户列表
const handleBindUser = async (row: PlatformAccount) => {
  currentPlatformAccount.value = row;
  selectedUserId.value = row.bind_user_id || null;
  bindUserDialogVisible.value = true;
  bindUserLoading.value = true;
  try {
    const res = await request({
      url: '/users',
      method: 'GET',
      params: { page: 1, page_size: 1000 }
    });
    userOptions.value = (res.data?.list || []).map((user: any) => ({
      label: user.username || user.name || user.id,
      value: user.id
    }));
  } finally {
    bindUserLoading.value = false;
  }
};

// 提交绑定账号
const submitBindUser = async () => {
  if (!currentPlatformAccount.value || !selectedUserId.value) return;
  bindUserLoading.value = true;
  try {
    await request({
      url: '/platform/account/bind_user',
      method: 'POST',
      data: {
        platform_account_id: currentPlatformAccount.value.id,
        user_id: selectedUserId.value
      }
    });
    message.success('绑定成功');
    bindUserDialogVisible.value = false;
    fetchPlatformAccounts(currentPlatformAccount.value.platform_id, currentPlatformCode.value);
  } catch (e) {
    message.error('绑定失败');
  } finally {
    bindUserLoading.value = false;
  }
};

// 添加 TaskConfig 相关的状态和方法
const showTaskConfigModal = ref(false);
const taskConfigList = ref<any[]>([]);
const selectedTaskConfigKeys = ref<number[]>([]);

// 打开任务配置弹窗
function handleTaskConfig(row: any) {
  // 确保当前平台代码与账号同步，避免平台判断失准
  if (row && row.platform_code) {
    currentPlatformCode.value = row.platform_code;
  }
  currentPlatformAccount.value = row;
  showTaskConfigModal.value = true;
  // 打开任务配置弹窗时重置页码，避免继承平台列表的第2页
  pagination.value.page = 1;
  fetchTaskConfigList();
}

// 获取任务配置列表
const fetchTaskConfigList = async () => {
  try {
    const params: TaskConfigListParams = {
      page: pagination.value.page,
      page_size: pagination.value.pageSize
    };
    if (currentPlatformAccount.value) {
      params.platform_account_id = currentPlatformAccount.value.id;
    }

    // 根据平台类型调用不同的API
    let res;
    if (isDz.value) {
      // 得众平台使用平台账号变体接口
      res = await getTaskConfigList(params);
    } else {
      // 闲赚侠等其他平台使用原有的任务配置接口
      res = await getXianzhuanxiaTaskConfigList(params);
    }

    console.log('rrrrrrr', res);
    if (res.data) {
      const list = res.data.items ?? res.data.list ?? [];
      taskConfigList.value = list;
      pagination.value.itemCount = res.data.total ?? (Array.isArray(list) ? list.length : 0);
    }
  } catch (error) {
    console.error('获取任务配置列表失败:', error);
  }
};

// 任务配置表格列定义
const taskConfigColumns = computed(() => {
  const baseColumns = [
    { type: 'selection', width: 40 },
    { title: 'ID', key: 'id', width: 60 }
  ];

  if (isDz.value) {
    // 得众平台的列
    return [
      ...baseColumns,
      {
        title: '运营商',
        key: 'isp',
        width: 100,
        render(row: any) {
          const ispMap: { [key: number]: string } = { 1: '移动', 2: '电信', 3: '联通', 0: '未知' };
          return ispMap[row.isp] || '未知';
        }
      },
      { title: '面值', key: 'face_value', width: 80 },
      { title: '产品ID', key: 'product_id', width: 100 },
      {
        title: '状态',
        key: 'enabled',
        width: 80,
        render(row: any) {
          return h(
            NTag,
            {
              type: row.enabled ? 'success' : 'error',
              bordered: false
            },
            { default: () => (row.enabled ? '启用' : '禁用') }
          );
        }
      },
      {
        title: '创建时间',
        key: 'created_at',
        render(row: any) {
          return formatDateTime(row.created_at);
        }
      },
      {
        title: '操作',
        key: 'actions',
        render(row: any) {
          return h(NSpace, {}, [
            h(
              NButton,
              { size: 'small', type: 'primary', onClick: () => handleEditTaskConfig(row) },
              { default: () => '编辑' }
            ),
            h(
              NPopconfirm,
              {
                onPositiveClick: () => handleDeleteTaskConfig(row)
              },
              {
                default: () => '确认删除该配置吗？',
                trigger: () => h(NButton, { size: 'small', type: 'error' }, { default: () => '删除' })
              }
            )
          ]);
        }
      }
    ] as any;
  }
  // 闲赚侠等其他平台的列（按备份文件还原原始结构）
  return [
    ...baseColumns,
    { title: '渠道ID', key: 'channel_id', width: 80 },
    { title: '渠道', key: 'channel_name', width: 120 },
    { title: '产品', key: 'product_name', width: 120 },
    { title: '面值', key: 'face_values' },
    { title: '最低结算价', key: 'min_settle_amounts' },
    {
      title: '状态',
      key: 'status',
      width: 80,
      render(row: any) {
        return h(
          NTag,
          {
            type: row.status === 1 ? 'success' : 'error',
            bordered: false
          },
          { default: () => (row.status === 1 ? '启用' : '禁用') }
        );
      }
    },
    {
      title: '创建时间',
      key: 'created_at',
      render(row: any) {
        return formatDateTime(row.created_at);
      }
    },
    {
      title: '操作',
      key: 'actions',
      render(row: any) {
        return h(NSpace, {}, [
          h(
            NButton,
            { size: 'small', type: 'primary', onClick: () => handleEditTaskConfig(row) },
            { default: () => '编辑' }
          ),
          h(
            NPopconfirm,
            {
              onPositiveClick: () => handleDeleteTaskConfig(row)
            },
            {
              default: () => '确认删除该配置吗？',
              trigger: () => h(NButton, { size: 'small', type: 'error' }, { default: () => '删除' })
            }
          )
        ]);
      }
    }
  ] as any;
});

// 编辑任务配置
const showEditTaskConfigModal = ref(false);
const editTaskConfigForm = ref({
  id: 0,
  platform_id: 0,
  platform_account_id: 0,
  // 闲赚侠平台字段
  channel_id: 0,
  face_values: '',
  min_settle_amounts: '',
  status: 1,
  // 得众平台字段
  isp: 1,
  face_value: 0,
  poll_interval_sec: 30,
  concurrency: 1,
  enabled: 1,
  // 通用字段
  product_id: ''
});

function handleEditTaskConfig(row: any) {
  if (isDz.value) {
    // 得众平台的字段映射
    editTaskConfigForm.value = {
      id: row.id,
      platform_id: row.platform_id,
      platform_account_id: row.platform_account_id,
      isp: row.isp || 1,
      face_value: row.face_value || 0,
      product_id: row.product_id || '',
      poll_interval_sec: row.poll_interval_sec || 30,
      concurrency: row.concurrency || 1,
      enabled: row.enabled !== undefined ? (row.enabled ? 1 : 0) : 1,
      // 闲赚侠字段设为默认值
      channel_id: 0,
      face_values: '',
      min_settle_amounts: '',
      status: 1
    };
  } else {
    // 闲赚侠平台的字段映射
    editTaskConfigForm.value = {
      id: row.id,
      platform_id: row.platform_id,
      platform_account_id: row.platform_account_id,
      channel_id: row.channel_id || 0,
      product_id: row.product_id || '',
      face_values: row.face_values || '',
      min_settle_amounts: row.min_settle_amounts || '',
      status: row.status || 1,
      // 得众字段设为默认值
      isp: 1,
      face_value: 0,
      poll_interval_sec: 30,
      concurrency: 1,
      enabled: 1
    };
  }
  showEditTaskConfigModal.value = true;
}

// 保存编辑的任务配置
async function handleSaveTaskConfig() {
  try {
    // 根据平台类型构造不同的数据结构
    let submitData: any = {
      id: editTaskConfigForm.value.id,
      platform_id: editTaskConfigForm.value.platform_id,
      platform_account_id: editTaskConfigForm.value.platform_account_id,
      product_id: editTaskConfigForm.value.product_id
    };

    if (isDz.value) {
      // 得众平台的数据结构
      submitData = {
        ...submitData,
        isp: editTaskConfigForm.value.isp,
        face_value: editTaskConfigForm.value.face_value,
        poll_interval_sec: editTaskConfigForm.value.poll_interval_sec,
        concurrency: editTaskConfigForm.value.concurrency,
        enabled: Boolean(editTaskConfigForm.value.enabled)
      };
    } else {
      // 闲赚侠等其他平台的数据结构
      submitData = {
        ...submitData,
        channel_id: editTaskConfigForm.value.channel_id,
        face_values: editTaskConfigForm.value.face_values,
        min_settle_amounts: editTaskConfigForm.value.min_settle_amounts,
        status: editTaskConfigForm.value.status
      };
    }

    // 根据平台类型调用不同的API
    if (isDz.value) {
      // 统一写入平台账号变体表，提交完整对象
      const accountId = currentPlatformAccount.value?.id ?? editTaskConfigForm.value.platform_account_id ?? 0;
      const payload: any = {
        id: editTaskConfigForm.value.id,
        platform_account_id: accountId,
        isp: editTaskConfigForm.value.isp,
        face_value: editTaskConfigForm.value.face_value,
        poll_interval_sec: editTaskConfigForm.value.poll_interval_sec,
        concurrency: editTaskConfigForm.value.concurrency,
        enabled: Boolean(editTaskConfigForm.value.enabled)
      };
      const pidNum = Number(editTaskConfigForm.value.product_id);
      if (!Number.isNaN(pidNum) && pidNum > 0) {
        payload.product_id = pidNum;
      }
      await updateTaskConfig(payload);
    } else {
      // 闲赚侠等其他平台使用原有的任务配置接口
      await updateXianzhuanxiaTaskConfig(submitData);
    }
    message.success('更新成功');
    showEditTaskConfigModal.value = false;
    fetchTaskConfigList();
  } catch (error: any) {
    message.error(error?.message || '更新失败');
  }
}

// 删除任务配置
async function handleDeleteTaskConfig(row: any) {
  try {
    // 根据平台类型调用不同的删除API
    if (isDz.value) {
      // 得众平台使用变体接口
      await deleteTaskConfig(row.id);
    } else {
      // 闲赚侠等其他平台使用原有的任务配置接口
      await deleteXianzhuanxiaTaskConfig(row.id);
    }
    message.success('删除成功');
    fetchTaskConfigList();
  } catch (error: any) {
    message.error(error?.message || '删除失败');
  }
}

// 批量设置任务配置状态
async function batchSetTaskConfigStatus(status: number) {
  if (!selectedTaskConfigKeys.value.length) return;
  try {
    for (const id of selectedTaskConfigKeys.value) {
      // 根据平台类型调用不同的API
      if (isDz.value) {
        const row = taskConfigList.value.find((r: any) => r.id === id);
        if (!row) continue;
        const accountId = row.platform_account_id ?? currentPlatformAccount.value?.id ?? 0;
        const payload: any = {
          id: row.id,
          platform_account_id: accountId,
          isp: row.isp,
          face_value: row.face_value,
          poll_interval_sec: row.poll_interval_sec,
          concurrency: row.concurrency,
          enabled: Boolean(status)
        };
        const pidNum = Number(row.product_id);
        if (!Number.isNaN(pidNum) && pidNum > 0) {
          payload.product_id = pidNum;
        }
        await updateTaskConfig(payload);
      } else {
        // 闲赚侠等其他平台使用原有的任务配置接口
        await updateXianzhuanxiaTaskConfig({ id, status });
      }
    }
    message.success(status === 1 ? '批量开启成功' : '批量关闭成功');
    fetchTaskConfigList();
  } catch (error: any) {
    message.error(error?.message || '批量操作失败');
  }
}

// 添加任务配置弹窗
const showAddTaskConfigModal = ref(false);
const addTaskConfigForm = ref({
  platform_id: 0,
  platform_account_id: 0,
  // 闲赚侠平台字段
  channel_id: 0,
  face_values: '',
  min_settle_amounts: '',
  status: 1,
  // 得众平台字段
  isp: 1,
  face_value: 0,
  poll_interval_sec: 30,
  concurrency: 1,
  enabled: 1,
  // 通用字段
  product_id: ''
});

function handleAddTaskConfig() {
  // 允许在账号状态缺失时也能弹窗，字段使用安全默认值
  const platformId = currentPlatformAccount.value?.platform_id ?? 0;
  const accountId = currentPlatformAccount.value?.id ?? 0;
  
  addTaskConfigForm.value = {
    platform_id: platformId,
    platform_account_id: accountId,
    // 闲赚侠平台字段
    channel_id: 0,
    face_values: '',
    min_settle_amounts: '',
    status: 1,
    // 得众平台字段
    isp: 1,
    face_value: 0,
    poll_interval_sec: 30,
    concurrency: 1,
    enabled: 1,
    // 通用字段
    product_id: ''
  };
  showAddTaskConfigModal.value = true;
}

// 保存新增的任务配置
async function handleSaveAddTaskConfig() {
  try {
    // 根据平台类型构造不同的数据结构
    let submitData: any = {
      platform_id: addTaskConfigForm.value.platform_id,
      platform_account_id: addTaskConfigForm.value.platform_account_id,
      product_id: addTaskConfigForm.value.product_id
    };

    if (isDz.value) {
      // 得众平台的数据结构
      submitData = {
        ...submitData,
        isp: addTaskConfigForm.value.isp,
        face_value: addTaskConfigForm.value.face_value,
        poll_interval_sec: addTaskConfigForm.value.poll_interval_sec,
        concurrency: addTaskConfigForm.value.concurrency,
        enabled: Boolean(addTaskConfigForm.value.enabled)
      };
    } else {
      // 闲赚侠等其他平台的数据结构
      submitData = {
        ...submitData,
        channel_id: addTaskConfigForm.value.channel_id,
        face_values: addTaskConfigForm.value.face_values,
        min_settle_amounts: addTaskConfigForm.value.min_settle_amounts,
        status: addTaskConfigForm.value.status
      };
    }

    // 根据平台类型调用不同的API
    if (isDz.value) {
      // 统一写入平台账号变体表，按单对象提交
      const payload: any = {
        platform_account_id: addTaskConfigForm.value.platform_account_id,
        isp: addTaskConfigForm.value.isp,
        face_value: addTaskConfigForm.value.face_value,
        poll_interval_sec: addTaskConfigForm.value.poll_interval_sec,
        concurrency: addTaskConfigForm.value.concurrency,
        enabled: Boolean(addTaskConfigForm.value.enabled)
      };
      const pidNum = Number(addTaskConfigForm.value.product_id);
      if (!Number.isNaN(pidNum) && pidNum > 0) {
        payload.product_id = pidNum;
      }
      await createTaskConfig(payload);
    } else {
      // 闲赚侠等其他平台使用原有的任务配置接口
      await createXianzhuanxiaTaskConfig([submitData]);
    }
    message.success('添加成功');
    showAddTaskConfigModal.value = false;
    fetchTaskConfigList();
  } catch (error: any) {
    message.error(error?.message || '添加失败');
  }
}

// 格式化日期时间
function formatDateTime(val: string) {
  if (!val) return '';
  const date = new Date(val);
  if (isNaN(date.getTime())) return val;
  const pad = (n: number) => n.toString().padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`;
}

// 批量新增配置相关状态和方法
const showChannelModal = ref(false);
const loadingChannels = ref(false);
const channels = ref<Channel[]>([]);
const selected = ref<{ [channelId: number]: number[] }>({});
const faceValues = ref<{ [channelId: number]: string }>({});
const minSettleAmounts = ref<{ [channelId: number]: string }>({});

// 省份选项
const provinceOptions = [
  '山东',
  '福建',
  '河北',
  '河南',
  '重庆',
  '湖北',
  '湖南',
  '海南',
  '江西',
  '黑龙江',
  '天津',
  '贵州',
  '陕西',
  '江苏',
  '安徽',
  '新疆',
  '西藏',
  '甘肃',
  '上海',
  '内蒙古',
  '辽宁',
  '广东',
  '青海',
  '北京',
  '广西',
  '山西',
  '四川',
  '云南',
  '浙江',
  '吉林',
  '宁夏',
  '香港',
  '澳门',
  '台湾'
];
const provinces = ref<{ [channelId: number]: string[] }>({});

function openChannelModal(account: string) {
  console.log('Opening channel modal for account:', account);
  showChannelModal.value = true;
  loadingChannels.value = true;
  getChannelList(account)
    .then(res => {
      const list = Array.isArray(res.data) ? res.data : [];
      channels.value = list;
      const selectedInit: { [channelId: number]: number[] } = {};
      const provincesInit: { [channelId: number]: string[] } = {};
      list.forEach((c: Channel) => {
        selectedInit[c.channelId] = [];
        provincesInit[c.channelId] = [];
      });
      selected.value = selectedInit;
      faceValues.value = {};
      minSettleAmounts.value = {};
      provinces.value = provincesInit;
    })
    .finally(() => {
      loadingChannels.value = false;
    });
}

function handleChannelChange(channelId: number, productIds: number[]) {
  selected.value[channelId] = productIds;
}

async function handleSaveChannelConfig() {
  // 批量新增仅用于闲赚侠平台，不与得众共用
  if (!isXianzhuanxia.value) {
    message.warning('当前平台不支持批量新增，请使用“新增配置”');
    showChannelModal.value = false;
    return;
  }
  if (!currentPlatformAccount.value) return;
  const payload = Object.entries(selected.value)
    .map(([cid, pids]) => {
      const productIds = pids as number[];
      return {
        platform_id: currentPlatformAccount.value?.platform_id || 0,
        platform_account_id: currentPlatformAccount.value?.id || 0,
        channel_id: Number(cid),
        channel_name: channels.value.find(c => c.channelId === Number(cid))?.channelName || '',
        face_values: faceValues.value[Number(cid)] || '',
        min_settle_amounts: minSettleAmounts.value[Number(cid)] || '',
        product_id: productIds.join(','),
        product_name: productIds
          .map(
            pid =>
              channels.value.find(c => c.channelId === Number(cid))?.productList.find(p => p.productId === pid)
                ?.productName || ''
          )
          .join(','),
        provinces: (provinces.value[Number(cid)] || []).join(',')
      };
    })
    .filter(item => item.product_id);
  if (!payload.length) {
    message.warning('请选择渠道及运营商');
    return;
  }
  try {
    // 闲赚侠平台批量新增配置
    await createXianzhuanxiaTaskConfig(payload);
    message.success('写入成功');
    showChannelModal.value = false;
    fetchTaskConfigList();
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

const showOrderStatsModal = ref(false);
const orderStats = ref({
  total_count: 0,
  success_count: 0,
  failed_count: 0,
  success_amount: 0,
  processing_count: 0
});

// 蜜蜂平台相关状态
// 移除showBeeProductModal变量，直接使用组件内置弹窗
const showBeeProvinceModal = ref(false);
const currentBeeAccount = ref<any>(null);

// 打开蜜蜂平台商品管理弹窗
function handleBeeProductManagement(row: any) {
  currentBeeAccount.value = row;
  // 直接调用组件的open方法
  if (beeProductManagementRef.value && row.id) {
    beeProductManagementRef.value.open(row.id);
  }
}

// 打开蜜蜂平台省份配置弹窗
function handleBeeProvinceConfig(row: any) {
  currentBeeAccount.value = row;
  showBeeProvinceModal.value = true;
}

// 查询平台余额
async function handleQueryBalance(row: PlatformAccount) {
  try {
    const res = await queryPlatformBalance(currentPlatformCode.value, row.id);
    const balance = res.data?.balance || '0';
    message.success(`账号 ${row.account_name} 的余额为：${balance}`);
  } catch (error: any) {
    message.error(error?.message || '查询余额失败');
  }
}

function handleViewOrderStatistics(row: any) {
  const customerId = row.customer_id || row.id;
  request({
    url: `/orders/statistics`,
    method: 'GET',
    params: { customer_id: customerId }
  })
    .then(res => {
      const stats = res.data?.data || res.data;
      orderStats.value = stats;
      showOrderStatsModal.value = true;
    })
    .catch(() => {
      message.error('获取订单统计失败');
    });
}

// 计算成功率和失败率
const successRate = computed(() => {
  const total = orderStats.value.total_count;
  return total > 0 ? `${((orderStats.value.success_count / total) * 100).toFixed(2)}%` : '0%';
});
const failedRate = computed(() => {
  const total = orderStats.value.total_count;
  return total > 0 ? `${((orderStats.value.failed_count / total) * 100).toFixed(2)}%` : '0%';
});

// 获取任务配置弹窗标题（已在文件前部实现，此处移除重复定义）

onMounted(() => {
  fetchPlatforms();
});
</script>

<template>
  <div class="min-h-500px flex-col-stretch gap-16px overflow-hidden lt-sm:overflow-auto">
    <!-- 搜索表单 -->
    <NCard>
      <NForm
        ref="searchFormRef"
        :model="searchForm"
        inline
        label-placement="left"
        label-width="auto"
        class="flex flex-wrap gap-16px"
      >
        <NGrid :cols="4" :x-gap="24">
          <NFormItemGi label="平台名称" path="name">
            <NInput v-model:value="searchForm.name" placeholder="请输入平台名称" />
          </NFormItemGi>
          <NFormItemGi label="平台代码" path="code">
            <NInput v-model:value="searchForm.code" placeholder="请输入平台代码" />
          </NFormItemGi>
          <NFormItemGi label="状态" path="status">
            <NSelect
              v-model:value="searchForm.status"
              :options="[
                { label: '启用', value: 1 },
                { label: '禁用', value: 0 }
              ]"
              placeholder="请选择状态"
              clearable
            />
          </NFormItemGi>
          <NFormItemGi>
            <NSpace>
              <NButton type="primary" @click="handleSearch(fetchPlatforms)">搜索</NButton>
              <NButton @click="handleReset">重置</NButton>
            </NSpace>
          </NFormItemGi>
        </NGrid>
      </NForm>
    </NCard>

    <!-- 数据表格 -->
    <NCard title="平台管理" :bordered="false" size="small" class="sm:flex-1-hidden card-wrapper">
      <template #header-extra>
        <NSpace>
          <NButton
            type="primary"
            @click="
              handleReset();
              showModal();
            "
          >
            新增平台
          </NButton>
        </NSpace>
      </template>
      <NDataTable
        :columns="columns"
        :data="data"
        :loading="loading"
        :pagination="pagination"
        :flex-height="!appStore.isMobile"
        :scroll-x="962"
        remote
        :row-key="row => row.id"
        class="sm:h-full"
        @update:page="onPageChange"
        @update:page-size="onPageSizeChange"
      />
    </NCard>

    <!-- 新增/编辑弹窗 -->
    <NModal
      v-model:show="visible"
      preset="dialog"
      :title="formModel.id ? '编辑平台' : '新增平台'"
      :style="{ width: '600px' }"
    >
      <NForm
        ref="formRef"
        :model="formModel"
        :rules="rules"
        label-placement="left"
        label-width="auto"
        require-mark-placement="right-hanging"
      >
        <NGrid :cols="2" :x-gap="24">
          <NFormItemGi label="平台名称" path="name">
            <NInput v-model:value="formModel.name" placeholder="请输入平台名称" />
          </NFormItemGi>
          <NFormItemGi label="平台代码" path="code">
            <NInput v-model:value="formModel.code" placeholder="请输入平台代码" />
          </NFormItemGi>
          <NFormItemGi label="API地址" path="api_url">
            <NInput v-model:value="formModel.api_url" placeholder="请输入API地址" />
          </NFormItemGi>
          <NFormItemGi label="描述" path="description">
            <NInput v-model:value="formModel.description" type="textarea" placeholder="请输入描述" />
          </NFormItemGi>
          <NFormItemGi label="状态" path="status">
            <NSwitch v-model:value="formModel.status" :checked-value="1" :unchecked-value="0" />
          </NFormItemGi>
        </NGrid>
      </NForm>
      <template #action>
        <NSpace>
          <NButton @click="hideModal">取消</NButton>
          <NButton type="primary" @click="handleFormSubmit">确定</NButton>
        </NSpace>
      </template>
    </NModal>

    <!-- 账号管理对话框 -->
    <NModal v-model:show="accountVisible" preset="dialog" title="平台账号管理" :style="{ width: '800px' }">
      <div class="flex flex-col gap-16px">
        <!-- 工具栏 -->
        <div class="flex justify-end gap-16px">
          <NButton type="primary" @click="accountFormRef?.add(accountData[0]?.platform_id)">新增账号</NButton>
          <NButton type="primary" @click="() => batchUpdatePushStatus(1)">批量开启推单</NButton>
          <NButton type="primary" @click="() => batchUpdatePushStatus(2)">批量关闭推单</NButton>
        </div>
        <!-- 账号列表 -->
        <NDataTable
          :columns="accountColumns"
          :data="accountData"
          :loading="accountLoading"
          :pagination="accountPagination"
          :flex-height="!appStore.isMobile"
          :scroll-x="962"
          v-model:checked-row-keys="selectedAccountIds"
          remote
          :row-key="row => row.id"
          class="sm:h-full"
          style="min-height: 300px"
          @update:page="onAccountPageChange"
          @update:page-size="onAccountPageSizeChange"
        />
      </div>
      <PlatformAccountForm ref="accountFormRef" @success="handleAccountSuccess" />
    </NModal>

    <!-- 绑定账号弹窗，单独放在外面 -->
    <NModal v-model:show="bindUserDialogVisible" preset="dialog" title="绑定本地账号" :style="{ width: '400px' }">
      <NForm>
        <NGrid :cols="1" :x-gap="24">
          <NFormItemGi label="本地账号">
            <NSelect
              v-model:value="selectedUserId"
              :options="userOptions"
              placeholder="请选择本地账号"
              filterable
              :loading="bindUserLoading"
              style="width: 100%"
            />
          </NFormItemGi>
        </NGrid>
      </NForm>
      <template #action>
        <NSpace>
          <NButton @click="bindUserDialogVisible = false">取消</NButton>
          <NButton type="primary" :loading="bindUserLoading" @click="submitBindUser">确定</NButton>
        </NSpace>
      </template>
    </NModal>

    <!-- 任务配置弹窗 -->
    <NModal v-model:show="showTaskConfigModal" :title="getTaskConfigModalTitle()" preset="dialog" style="width: 900px">
      <template #header>
        <div style="display: flex; align-items: center; width: 100%; box-sizing: border-box">
          <span style="flex: 1">{{ getTaskConfigModalTitle() }} - {{ currentPlatformAccount?.account_name }}</span>
          <NButton
            type="primary"
            @click="() => (isXianzhuanxia ? openChannelModal(currentPlatformAccount?.account_name || '') : handleAddTaskConfig())"
          >
            {{ isXianzhuanxia ? '增加配置' : '新增配置' }}
          </NButton>
          <NButton
            type="success"
            style="margin-left: 8px"
            :disabled="selectedTaskConfigKeys.length === 0"
            @click="batchSetTaskConfigStatus(1)"
          >
            批量开启
          </NButton>
          <NButton
            type="error"
            style="margin-left: 8px"
            :disabled="selectedTaskConfigKeys.length === 0"
            @click="batchSetTaskConfigStatus(0)"
          >
            批量关闭
          </NButton>
        </div>
      </template>
      <NDataTable
        v-model:checked-row-keys="selectedTaskConfigKeys"
        :columns="taskConfigColumns"
        :data="taskConfigList"
        :pagination="pagination"
        :loading="loading"
        :row-key="row => row.id"
        style="margin-top: 16px"
      />
    </NModal>

    <!-- 编辑任务配置弹窗 -->
    <NModal
      v-model:show="showEditTaskConfigModal"
      :title="getTaskConfigModalTitle()"
      preset="dialog"
      style="width: 500px"
    >
      <NForm
        :model="editTaskConfigForm"
        label-placement="left"
        label-width="auto"
        require-mark-placement="right-hanging"
      >
        <!-- 得众平台的字段 -->
        <template v-if="isDz">
          <NFormItem label="运营商" path="isp">
            <NSelect
              v-model:value="editTaskConfigForm.isp"
              :options="[
                { label: '移动', value: 1 },
                { label: '电信', value: 2 },
                { label: '联通', value: 3 }
              ]"
            />
          </NFormItem>
          <NFormItem label="面值" path="face_value">
            <NInputNumber v-model:value="editTaskConfigForm.face_value" :min="1" />
          </NFormItem>
          <NFormItem label="产品ID" path="product_id">
            <NInput v-model:value="editTaskConfigForm.product_id" placeholder="产品ID" />
          </NFormItem>
          <NFormItem label="状态" path="enabled">
            <NSelect
              v-model:value="editTaskConfigForm.enabled"
              :options="[
                { label: '启用', value: 1 },
                { label: '禁用', value: 0 }
              ]"
            />
          </NFormItem>
        </template>

        <!-- 闲赚侠等其他平台的字段 -->
        <template v-else>
          <NFormItem label="渠道ID" path="channel_id">
            <NInputNumber v-model:value="editTaskConfigForm.channel_id" :min="1" />
          </NFormItem>
          <NFormItem label="产品ID" path="product_id">
            <NInput v-model:value="editTaskConfigForm.product_id" placeholder="多个ID用逗号分隔" />
          </NFormItem>
          <NFormItem label="面值" path="face_values">
            <NInput v-model:value="editTaskConfigForm.face_values" placeholder="50,100,200" />
          </NFormItem>
          <NFormItem label="最低结算价" path="min_settle_amounts">
            <NInput v-model:value="editTaskConfigForm.min_settle_amounts" placeholder="49.5,99,198" />
          </NFormItem>
          <NFormItem label="状态" path="status">
            <NSelect
              v-model:value="editTaskConfigForm.status"
              :options="[
                { label: '启用', value: 1 },
                { label: '禁用', value: 0 }
              ]"
            />
          </NFormItem>
        </template>
      </NForm>
      <template #action>
        <NSpace>
          <NButton @click="showEditTaskConfigModal = false">取消</NButton>
          <NButton type="primary" @click="handleSaveTaskConfig">确定</NButton>
        </NSpace>
      </template>
    </NModal>

    <!-- 新增任务配置弹窗 -->
    <NModal
      v-model:show="showAddTaskConfigModal"
      :title="`新增${getTaskConfigModalTitle()}`"
      preset="dialog"
      style="width: 500px"
    >
      <NForm
        :model="addTaskConfigForm"
        label-placement="left"
        label-width="auto"
        require-mark-placement="right-hanging"
      >
        <!-- 得众平台的字段 -->
        <template v-if="isDz">
          <NFormItem label="运营商" path="isp">
            <NSelect
              v-model:value="addTaskConfigForm.isp"
              :options="[
                { label: '移动', value: 1 },
                { label: '电信', value: 2 },
                { label: '联通', value: 3 }
              ]"
            />
          </NFormItem>
          <NFormItem label="面值" path="face_value">
            <NInputNumber v-model:value="addTaskConfigForm.face_value" :min="1" />
          </NFormItem>
          <NFormItem label="产品ID" path="product_id">
            <NInput v-model:value="addTaskConfigForm.product_id" placeholder="产品ID" />
          </NFormItem>
          <NFormItem label="状态" path="enabled">
            <NSelect
              v-model:value="addTaskConfigForm.enabled"
              :options="[
                { label: '启用', value: 1 },
                { label: '禁用', value: 0 }
              ]"
            />
          </NFormItem>
        </template>

        <!-- 闲赚侠等其他平台的字段 -->
        <template v-else>
          <NFormItem label="渠道ID" path="channel_id">
            <NInputNumber v-model:value="addTaskConfigForm.channel_id" :min="1" />
          </NFormItem>
          <NFormItem label="产品ID" path="product_id">
            <NInput v-model:value="addTaskConfigForm.product_id" placeholder="多个ID用逗号分隔" />
          </NFormItem>
          <NFormItem label="面值" path="face_values">
            <NInput v-model:value="addTaskConfigForm.face_values" placeholder="50,100,200" />
          </NFormItem>
          <NFormItem label="最低结算价" path="min_settle_amounts">
            <NInput v-model:value="addTaskConfigForm.min_settle_amounts" placeholder="49.5,99,198" />
          </NFormItem>
          <NFormItem label="状态" path="status">
            <NSelect
              v-model:value="addTaskConfigForm.status"
              :options="[
                { label: '启用', value: 1 },
                { label: '禁用', value: 0 }
              ]"
            />
          </NFormItem>
        </template>
      </NForm>
      <template #action>
        <NSpace>
          <NButton @click="showAddTaskConfigModal = false">取消</NButton>
          <NButton type="primary" @click="handleSaveAddTaskConfig">确定</NButton>
        </NSpace>
      </template>
    </NModal>

    <!-- 批量新增配置弹窗 -->
    <NModal v-model:show="showChannelModal" title="批量新增配置" preset="dialog" style="width: 500px">
      <NSpin :show="loadingChannels">
        <div v-for="channel in channels" :key="channel.channelId" style="margin-bottom: 16px">
          <div style="font-weight: bold">{{ channel.channelName }}</div>
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
          <div class="flex items-center" style="margin-top: 8px">
            <span style="width: 90px">拉取面值</span>
            <div style="flex: 1">
              <NInput
                v-model:value="faceValues[channel.channelId]"
                size="small"
                placeholder="50,100,200,500,1000"
                style="width: 200px"
              />
              <div style="color: #888; font-size: 12px; margin-top: 2px">支持多个，逗号隔开，最多5个面值，不要重复</div>
            </div>
          </div>
          <div class="flex items-center" style="margin-top: 8px">
            <span style="width: 90px">最低结算价格</span>
            <NInput
              v-model:value="minSettleAmounts[channel.channelId]"
              size="small"
              placeholder="最低结算价格"
              style="width: 200px"
            />
            <div style="color: #888; font-size: 12px; margin-top: 2px">支持多个,逗号隔开,faceValues对应</div>
          </div>
          <div class="flex items-center" style="margin-top: 8px">
            <span style="width: 90px">省份</span>
            <NCheckboxGroup v-model:value="provinces[channel.channelId]" style="flex: 1; flex-wrap: wrap">
              <NCheckbox
                v-for="prov in provinceOptions"
                :key="prov"
                :value="prov"
                :label="prov"
                style="margin-right: 8px; margin-bottom: 4px"
              />
            </NCheckboxGroup>
            <div style="color: #888; font-size: 12px; margin-top: 2px">可多选，留空为全国</div>
          </div>
        </div>
        <div style="text-align: right">
          <NButton type="primary" @click="handleSaveChannelConfig">确定</NButton>
          <NButton style="margin-left: 8px" @click="showChannelModal = false">取消</NButton>
        </div>
      </NSpin>
    </NModal>

    <!-- 订单统计弹窗 -->
    <NModal v-model:show="showOrderStatsModal" preset="dialog" title="订单统计信息！" style="width: 480px">
      <NGrid :cols="2" :x-gap="24" :y-gap="24">
        <NGridItem>
          <NCard size="small" content-style="text-align:center;">
            <NStatistic
              label="总订单数"
              :value="orderStats.total_count"
              value-style="color: #409eff; font-size: 32px;"
            />
          </NCard>
        </NGridItem>
        <NGridItem>
          <NCard size="small" content-style="text-align:center;">
            <NStatistic
              label="成功订单"
              :value="orderStats.success_count"
              value-style="color: #67c23a; font-size: 32px;"
            />
          </NCard>
        </NGridItem>
        <NGridItem>
          <NCard size="small" content-style="text-align:center;">
            <NStatistic
              label="失败订单"
              :value="orderStats.failed_count"
              value-style="color: #f56c6c; font-size: 32px;"
            />
          </NCard>
        </NGridItem>
        <NGridItem>
          <NCard size="small" content-style="text-align:center;">
            <NStatistic
              label="充值中订单"
              :value="orderStats.processing_count"
              value-style="color: #409eff; font-size: 32px;"
            />
          </NCard>
        </NGridItem>
        <NGridItem>
          <NCard size="small" content-style="text-align:center;">
            <NStatistic
              label="今日交易额"
              :value="orderStats.success_amount"
              value-style="color: #e6a23c; font-size: 32px;"
            />
          </NCard>
        </NGridItem>
      </NGrid>
      <NGrid :cols="2" :x-gap="24" :y-gap="24" style="margin-top: 16px">
        <NGridItem>
          <NCard size="small" content-style="text-align:center;">
            <NStatistic label="成功率" :value="successRate" value-style="color: #67c23a; font-size: 28px;" />
          </NCard>
        </NGridItem>
        <NGridItem>
          <NCard size="small" content-style="text-align:center;">
            <NStatistic label="失败率" :value="failedRate" value-style="color: #f56c6c; font-size: 28px;" />
          </NCard>
        </NGridItem>
      </NGrid>
    </NModal>

    <!-- 蜜蜂平台商品管理组件 -->
    <BeeProductManagement ref="beeProductManagementRef" />

    <!-- 蜜蜂平台省份配置弹窗 -->
    <NModal
      v-model:show="showBeeProvinceModal"
      preset="card"
      title="蜜蜂平台省份配置"
      style="width: 1000px; max-height: 80vh"
      :closable="true"
      :mask-closable="false"
    >
      <ProductPriceForm
        v-if="currentBeeAccount"
        :account="currentBeeAccount"
        mode="province"
        @close="showBeeProvinceModal = false"
      />
    </NModal>
  </div>
</template>

<style scoped></style>