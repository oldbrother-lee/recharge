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
                <NButton type="primary" @click="handleSearch(fetchPlatforms)">
                  搜索
                </NButton>
                <NButton @click="handleReset">重置</NButton>
              </NSpace>
            </NFormItemGi>
          </NGrid>
        </NForm>
      </NCard>
  
      <!-- 数据表格 -->
      <NCard :title="'平台管理'" :bordered="false" size="small" class="sm:flex-1-hidden card-wrapper">
        <template #header-extra>
          <NSpace>
            <NButton type="primary" @click="handleReset(); showModal()">
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
          @update:page="onPageChange"
          @update:page-size="onPageSizeChange"
          class="sm:h-full"
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
      <NModal
        v-model:show="accountVisible"
        preset="dialog"
        title="平台账号管理"
        :style="{ width: '800px' }"
      >
        <div class="flex flex-col gap-16px">
          <!-- 工具栏 -->
          <div class="flex gap-16px justify-end">
            <NButton type="primary" @click="accountFormRef?.add(accountData[0]?.platform_id)">
              新增账号
            </NButton>
            <NButton type="primary" @click="() => batchUpdatePushStatus(1)">
              批量开启推单
            </NButton>
            <NButton type="primary" @click="() => batchUpdatePushStatus(2)">
              批量关闭推单
            </NButton>
          </div>
          <!-- 账号列表 -->
          <NDataTable
            :columns="accountColumns"
            :data="accountData"
            :loading="accountLoading"
            :pagination="accountPagination"
            :flex-height="!appStore.isMobile"
            :scroll-x="962"
            remote
            :row-key="row => row.id"
            @update:page="onAccountPageChange"
            @update:page-size="onAccountPageSizeChange"
            class="sm:h-full"
            style="min-height: 300px;"
            v-model:checked-row-keys="selectedAccountIds"
          />
        </div>
        <PlatformAccountForm ref="accountFormRef" @success="handleAccountSuccess" />
      </NModal>

      <!-- 绑定账号弹窗，单独放在外面 -->
      <NModal
        v-model:show="bindUserDialogVisible"
        preset="dialog"
        title="绑定本地账号"
        :style="{ width: '400px' }"
      >
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
      <n-modal v-model:show="showTaskConfigModal" title="拉取订单配置" preset="dialog" style="width: 900px;">
        <template #header>
          <div style="display: flex; align-items: center; width: 100%; box-sizing: border-box;">
            <span style="flex: 1;">拉取订单配置 - {{ currentPlatformAccount?.account_name }}</span>
            <NButton type="primary" @click="openChannelModal(currentPlatformAccount.account_name)">增加配置</NButton>
            <NButton
              type="success"
              style="margin-left: 8px;"
              :disabled="selectedTaskConfigKeys.length === 0"
              @click="batchSetTaskConfigStatus(1)"
            >批量开启</NButton>
            <NButton
              type="error"
              style="margin-left: 8px;"
              :disabled="selectedTaskConfigKeys.length === 0"
              @click="batchSetTaskConfigStatus(0)"
            >批量关闭</NButton>
          </div>
        </template>
        <n-data-table
          :columns="taskConfigColumns"
          :data="taskConfigList"
          :pagination="pagination"
          :loading="loading"
          :row-key="row => row.id"
          v-model:checked-row-keys="selectedTaskConfigKeys"
          style="margin-top: 16px;"
        />
      </n-modal>

      <!-- 编辑任务配置弹窗 -->
      <n-modal v-model:show="showEditTaskConfigModal" title="编辑配置" preset="dialog" style="width: 500px;">
        <n-form
          :model="editTaskConfigForm"
          label-placement="left"
          label-width="auto"
          require-mark-placement="right-hanging"
        >
          <n-form-item label="渠道ID" path="channel_id">
            <n-input-number v-model:value="editTaskConfigForm.channel_id" :min="1" />
          </n-form-item>
          <n-form-item label="产品ID" path="product_id">
            <n-input v-model:value="editTaskConfigForm.product_id" placeholder="多个ID用逗号分隔" />
          </n-form-item>
          <n-form-item label="面值" path="face_values">
            <n-input v-model:value="editTaskConfigForm.face_values" placeholder="50,100,200" />
          </n-form-item>
          <n-form-item label="最低结算价" path="min_settle_amounts">
            <n-input v-model:value="editTaskConfigForm.min_settle_amounts" placeholder="49.5,99,198" />
          </n-form-item>
          <n-form-item label="状态" path="status">
            <n-select
              v-model:value="editTaskConfigForm.status"
              :options="[
                { label: '启用', value: 1 },
                { label: '禁用', value: 0 }
              ]"
            />
          </n-form-item>
        </n-form>
        <template #action>
          <n-space>
            <n-button @click="showEditTaskConfigModal = false">取消</n-button>
            <n-button type="primary" @click="handleSaveTaskConfig">确定</n-button>
          </n-space>
        </template>
      </n-modal>

      <!-- 新增任务配置弹窗 -->
      <n-modal v-model:show="showAddTaskConfigModal" title="新增配置" preset="dialog" style="width: 500px;">
        <n-form
          :model="addTaskConfigForm"
          label-placement="left"
          label-width="auto"
          require-mark-placement="right-hanging"
        >
          <n-form-item label="渠道ID" path="channel_id">
            <n-input-number v-model:value="addTaskConfigForm.channel_id" :min="1" />
          </n-form-item>
          <n-form-item label="产品ID" path="product_id">
            <n-input v-model:value="addTaskConfigForm.product_id" placeholder="多个ID用逗号分隔" />
          </n-form-item>
          <n-form-item label="面值" path="face_values">
            <n-input v-model:value="addTaskConfigForm.face_values" placeholder="50,100,200" />
          </n-form-item>
          <n-form-item label="最低结算价" path="min_settle_amounts">
            <n-input v-model:value="addTaskConfigForm.min_settle_amounts" placeholder="49.5,99,198" />
          </n-form-item>
          <n-form-item label="状态" path="status">
            <n-select
              v-model:value="addTaskConfigForm.status"
              :options="[
                { label: '启用', value: 1 },
                { label: '禁用', value: 0 }
              ]"
            />
          </n-form-item>
        </n-form>
        <template #action>
          <n-space>
            <n-button @click="showAddTaskConfigModal = false">取消</n-button>
            <n-button type="primary" @click="handleSaveAddTaskConfig">确定</n-button>
          </n-space>
        </template>
      </n-modal>

      <!-- 批量新增配置弹窗 -->
      <NModal
        v-model:show="showChannelModal"
        title="批量新增配置"
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
            <div class="flex items-center" style="margin-top: 8px;">
              <span style="width: 90px;">省份</span>
              <NCheckboxGroup
                v-model:value="provinces[channel.channelId]"
                style="flex: 1; flex-wrap: wrap;"
              >
                <NCheckbox
                  v-for="prov in provinceOptions"
                  :key="prov"
                  :value="prov"
                  :label="prov"
                  style="margin-right: 8px; margin-bottom: 4px;"
                />
              </NCheckboxGroup>
              <div style="color: #888; font-size: 12px; margin-top: 2px;">
                可多选，留空为全国
              </div>
            </div>
          </div>
          <div style="text-align: right;">
            <NButton type="primary" @click="handleSaveChannelConfig">确定</NButton>
            <NButton @click="showChannelModal = false" style="margin-left: 8px;">取消</NButton>
          </div>
        </NSpin>
      </NModal>

      <!-- 订单统计弹窗 -->
      <NModal v-model:show="showOrderStatsModal" preset="dialog" title="订单统计信息！" style="width: 480px;">
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
        <NGrid :cols="2" :x-gap="24" :y-gap="24" style="margin-top: 16px;">
          <NGridItem>
            <NCard size="small" content-style="text-align:center;">
              <NStatistic
                label="成功率"
                :value="successRate"
                value-style="color: #67c23a; font-size: 28px;"
              />
            </NCard>
          </NGridItem>
          <NGridItem>
            <NCard size="small" content-style="text-align:center;">
              <NStatistic
                label="失败率"
                :value="failedRate"
                value-style="color: #f56c6c; font-size: 28px;"
              />
            </NCard>
          </NGridItem>
        </NGrid>
      </NModal>
    </div>
  </template>
  
  <script setup lang="tsx">
  import { ref, h, watch, onMounted, computed } from 'vue';
  import { useTable } from '@/hooks/useTable';
  import { useModal } from '@/hooks/useModal';
  import { useForm } from '@/hooks/useForm';
  import { useMessage } from 'naive-ui';
  import { request } from '@/service/request';
  import type { DataTableColumns } from 'naive-ui';
  import { NButton, NPopconfirm, NCard, NForm, NFormItem, NSpace, NInput, NSelect, NSwitch, NModal, NDataTable, NGrid, NFormItemGi, NTag, NGridItem, NStatistic } from 'naive-ui';
  import { useAppStore } from '@/store/modules/app';
  import PlatformAccountForm from './components/PlatformAccountForm.vue';
  import { getChannelList } from '@/api/platform';
  import { getTaskConfigList, deleteTaskConfig, updateTaskConfig, createTaskConfig } from '@/api/taskConfig';
  import type { ApiResponse } from '@/types/api';
 
  
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
    description: string;
    status: number;
    created_at: string;
    bind_user_id?: number;
    bind_user_name?: string;
    push_status: number;
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
  
  // 添加 computed 属性
  const isXianzhuanxia = computed(() => {
    console.log('Computing isXianzhuanxia:', currentPlatformCode.value);
    return currentPlatformCode.value === 'xianzhuanxia';
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
    { label: "admin", value: 1 },
    { label: "test2", value: 3 }
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
      await handleSubmit();
      if (formModel.value.id) {
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
        if (row.push_status === 1) return '开启';
        if (row.push_status === 2) return '关闭';
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
      width: 300,
      render(row: PlatformAccount) {
        return (
          <div class="flex-center gap-8px">
            <NButton type="primary" ghost size="small" onClick={() => handleViewOrderStatistics(row)}>查看订单</NButton>
            <NButton type="primary" ghost size="small" onClick={() => accountFormRef.value?.edit(row)}>
              编辑
            </NButton>
            {isXianzhuanxia.value && (
              <NButton 
                type="primary" 
                ghost 
                size="small" 
                onClick={() => {
                  console.log('Platform code:', currentPlatformCode.value);
                  console.log('Is equal to xianzhuanxia:', isXianzhuanxia.value);
                  handleTaskConfig(row);
                }}
              >
                配置拉取订单
              </NButton>
            )}
            <NButton type="info" ghost size="small" onClick={() => handleBindUser(row)}>
              {row.bind_user_id ? '更换绑定' : '绑定账号'}
            </NButton>
            <NPopconfirm onPositiveClick={() => handleDeleteAccount(row)}>
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
    currentPlatformAccount.value = row;
    showTaskConfigModal.value = true;
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
      const res = await getTaskConfigList(params) as ApiResponse<TaskConfigListData>;
      console.log("rrrrrrr",res)
      if (res.data) {
        console.log(res.data.list,"gggggg");
        taskConfigList.value = res.data.list;
        pagination.value.itemCount = res.data.total;
      }
    } catch (error) {
      console.error('获取任务配置列表失败:', error);
    }
  };
  
  // 任务配置表格列定义
  const taskConfigColumns: DataTableColumns<any> = [
    { type: 'selection', width: 40 },
    { title: 'ID', key: 'id', width: 60 },
    { title: '渠道ID', key: 'channel_id', width: 80 },
    { title: '渠道', key: 'channel_name', width: 120 },
    { title: '产品', key: 'product_name', width: 120 },
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
          h(NButton, { size: 'small', type: 'primary', onClick: () => handleEditTaskConfig(row) }, { default: () => '编辑' }),
          h(NPopconfirm, {
            onPositiveClick: () => handleDeleteTaskConfig(row)
          }, {
            default: () => '确认删除该配置吗？',
            trigger: () => h(NButton, { size: 'small', type: 'error' }, { default: () => '删除' })
          })
        ]);
      }
    }
  ];
  
  // 编辑任务配置
  const showEditTaskConfigModal = ref(false);
  const editTaskConfigForm = ref({
    id: 0,
    platform_id: 0,
    platform_account_id: 0,
    channel_id: 0,
    product_id: '',
    face_values: '',
    min_settle_amounts: '',
    status: 1
  });
  
  function handleEditTaskConfig(row: any) {
    editTaskConfigForm.value = {
      id: row.id,
      platform_id: row.platform_id,
      platform_account_id: row.platform_account_id,
      channel_id: row.channel_id,
      product_id: row.product_id,
      face_values: row.face_values,
      min_settle_amounts: row.min_settle_amounts,
      status: row.status
    };
    showEditTaskConfigModal.value = true;
  }
  
  // 保存编辑的任务配置
  async function handleSaveTaskConfig() {
    try {
      await updateTaskConfig(editTaskConfigForm.value);
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
      await deleteTaskConfig(row.id);
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
        await updateTaskConfig({ id, status });
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
    channel_id: 0,
    product_id: '',
    face_values: '',
    min_settle_amounts: '',
    status: 1
  });
  
  function handleAddTaskConfig() {
    if (!currentPlatformAccount.value) return;
    
    addTaskConfigForm.value = {
      platform_id: currentPlatformAccount.value.platform_id,
      platform_account_id: currentPlatformAccount.value.id,
      channel_id: 0,
      product_id: '',
      face_values: '',
      min_settle_amounts: '',
      status: 1
    };
    showAddTaskConfigModal.value = true;
  }
  
  // 保存新增的任务配置
  async function handleSaveAddTaskConfig() {
    try {
      await createTaskConfig(addTaskConfigForm.value);
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
    '山东','福建','河北','河南','重庆','湖北','湖南','海南','江西','黑龙江','天津','贵州','陕西','江苏','安徽','新疆','西藏','甘肃','上海','内蒙古','辽宁','广东','青海','北京','广西','山西','四川','云南','浙江','吉林','宁夏','香港','澳门','台湾'
  ];
  const provinces = ref<{ [channelId: number]: string[] }>({});
  
  function openChannelModal(account: string) {
    console.log('Opening channel modal for account:', account);
    showChannelModal.value = true;
    loadingChannels.value = true;
    getChannelList(account).then(res => {
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
    }).finally(() => {
      loadingChannels.value = false;
    });
  }
  
  function handleChannelChange(channelId: number, productIds: number[]) {
    selected.value[channelId] = productIds;
  }
  
  async function handleSaveChannelConfig() {
    if (!currentPlatformAccount.value) return;
    const payload = Object.entries(selected.value)
      .map(([cid, pids]) => {
        const productIds = (pids as number[]);
        return {
          platform_id: currentPlatformAccount.value.platform_id,
          platform_account_id: currentPlatformAccount.value.id,
          channel_id: Number(cid),
          channel_name: channels.value.find(c => c.channelId === Number(cid))?.channelName || '',
          face_values: faceValues.value[Number(cid)] || '',
          min_settle_amounts: minSettleAmounts.value[Number(cid)] || '',
          product_id: productIds.join(','),
          product_name: productIds
            .map(pid => channels.value.find(c => c.channelId === Number(cid))?.productList.find(p => p.productId === pid)?.productName || '')
            .join(','),
          provinces: (provinces.value[Number(cid)] || []).join(',')
        }
      })
      .filter(item => item.product_id);
    if (!payload.length) {
      message.warning('请选择渠道及运营商');
      return;
    }
    try {
      await createTaskConfig(payload);
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
  
  const showOrderStatsModal = ref(false)
  const orderStats = ref({
    total_count: 0,
    success_count: 0,
    failed_count: 0,
    success_amount: 0,
    processing_count: 0
  })

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
    return total > 0 ? ((orderStats.value.success_count / total) * 100).toFixed(2) + '%' : '0%';
  });
  const failedRate = computed(() => {
    const total = orderStats.value.total_count;
    return total > 0 ? ((orderStats.value.failed_count / total) * 100).toFixed(2) + '%' : '0%';
  });
  
  onMounted(() => {
    fetchPlatforms();
  });
  </script>
  
  <style scoped>

  </style>