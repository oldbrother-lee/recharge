<template>
  <div class="min-h-500px flex-col-stretch gap-16px overflow-hidden lt-sm:overflow-auto">
    <!-- 搜索表单 -->
    <NCard :bordered="false" size="small" class="mb-16px">
      <NForm
        ref="searchFormRef"
        :model="searchForm"
        label-placement="left"
        :label-width="80"
      >
        <NCollapse :default-expanded-names="[]">
          <NCollapseItem title="搜索条件" name="api-search">
            <NGrid responsive="screen" item-responsive :x-gap="24">
              <NFormItemGi span="24 s:12 m:6" label="接口名称" path="name">
                <NInput v-model:value="searchForm.name" placeholder="请输入接口名称" />
              </NFormItemGi>
              <NFormItemGi span="24 s:12 m:6" label="平台" path="platform_id">
                <NSelect
                  v-model:value="searchForm.platform_id"
                  :options="platformOptions"
                  placeholder="请选择平台"
                  clearable
                />
              </NFormItemGi>
              <NFormItemGi span="24 s:12 m:6" label="状态" path="status">
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
              <NFormItemGi span="24" class="pr-24px">
                <NSpace class="w-full" justify="end">
                  <NButton @click="handleReset">重置</NButton>
                  <NButton type="primary" ghost @click="handleSearch(fetchAPIs)">搜索</NButton>
                </NSpace>
              </NFormItemGi>
            </NGrid>
          </NCollapseItem>
        </NCollapse>
      </NForm>
    </NCard>

    <!-- 数据表格 -->
    <NCard :title="'接口管理'" :bordered="false" size="small" class="flex-1 card-wrapper">
      <template #header-extra>
        <NSpace>
          <NButton type="primary" @click="handleAdd()">
            新增接口
          </NButton>
        </NSpace>
      </template>
      <NDataTable
        :columns="columns"
        :data="data"
        :loading="loading"
        :pagination="pagination"
        :flex-height="true"
        :scroll-x="1200"
        remote
        :row-key="row => row.id"
        @update:page="onPageChange"
        @update:page-size="onPageSizeChange"
        class="min-h-400px"
        size="small"
      />
    </NCard>

    <!-- 新增/编辑弹窗 -->
    <NModal
      v-model:show="visible"
      preset="dialog"
      :title="formModel.id ? '编辑接口' : '新增接口'"
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
        <NFormItem label="接口名称" path="name">
          <NInput v-model:value="formModel.name" placeholder="请输入接口名称" />
        </NFormItem>
        <NFormItem label="平台" path="platform_id">
          <NSelect
            v-model:value="formModel.platform_id"
            :options="platformOptions"
            placeholder="请选择平台"
            @change="handlePlatformChange"
          />
        </NFormItem>
        <NFormItem label="账号ID" path="account_id">
          <NSelect
            v-model:value="formModel.account_id"
            :options="accountOptions"
            placeholder="请选择账号"
          />
        </NFormItem>
        <!-- <NFormItem label="商户ID" path="merchant_id">
          <NInput v-model:value="formModel.merchant_id" placeholder="商户id" />
        </NFormItem> -->
        <NFormItem label="接口地址" path="url">
          <NInput v-model:value="formModel.url" placeholder="请输入接口地址" />
        </NFormItem>
        <NFormItem label="回调地址" path="callback_url">
          <NInput v-model:value="formModel.callback_url" placeholder="回调地址" />
        </NFormItem>
        <!-- <NFormItem label="商户密钥" path="secret_key">
          <NInput v-model:value="formModel.secret_key" placeholder="商户密钥" />
        </NFormItem> -->
        <NFormItem label="请求方法" path="method">
          <NSelect
            v-model:value="formModel.method"
            :options="[
              { label: 'GET', value: 'GET' },
              { label: 'POST', value: 'POST' },
              { label: 'PUT', value: 'PUT' },
              { label: 'DELETE', value: 'DELETE' }
            ]"
            placeholder="请选择请求方法"
          />
        </NFormItem>
        <NFormItem label="描述" path="description">
          <NInput v-model:value="formModel.description" type="textarea" placeholder="请输入描述" />
        </NFormItem>
        <NFormItem label="状态" path="status">
          <NSwitch v-model:value="formModel.status" :checked-value="1" :unchecked-value="0" />
        </NFormItem>
      </NForm>
      <template #action>
        <NSpace>
          <NButton @click="hideModal">取消</NButton>
          <NButton type="primary" @click="handleFormSubmit">确定</NButton>
        </NSpace>
      </template>
    </NModal>

    <!-- 参数配置对话框 -->
    <NModal
      v-model:show="paramVisible"
      preset="dialog"
      title="套餐配置"
      :style="{ width: '1000px' }"
    >
      <div class="flex flex-col gap-16px">
        <!-- 工具栏 -->
        <div class="flex justify-end">
          <NButton type="primary" @click="paramFormRef?.add(apiId)">
            新增套餐
          </NButton>
        </div>
        <!-- 参数列表 -->
        <NDataTable
          :columns="paramColumns"
          :data="paramData"
          :loading="paramLoading"
          :pagination="paramPagination"
          :flex-height="!appStore.isMobile"
          :scroll-x="962"
          remote
          :row-key="row => row.id"
          @update:page="onParamPageChange"
          @update:page-size="onParamPageSizeChange"
          class="sm:h-full"
          style="min-height: 300px;"
        />
      </div>
      <PlatformAPIParamForm ref="paramFormRef" @success="handleParamSuccess" />
    </NModal>
  </div>
</template>

<script setup lang="tsx">
import { ref, onMounted } from 'vue';
import { useTable } from '@/hooks/useTable';
import { useModal } from '@/hooks/useModal';
import { useForm } from '@/hooks/useForm';
import { useMessage } from 'naive-ui';
import { request } from '@/service/request';
import type { DataTableColumns } from 'naive-ui';
import { NButton, NPopconfirm, NCard, NForm, NFormItem, NFormItemGi, NSpace, NInput, NSelect, NSwitch, NModal, NDataTable, NTag, NCollapse, NCollapseItem, NGrid } from 'naive-ui';
import { useAppStore } from '@/store/modules/app';
import PlatformAPIParamForm from './components/PlatformAPIParamForm.vue';

interface PlatformAPI {
  id: number;
  name: string;
  code: string;
  platform_id: number;
  api_url: string;
  method: string;
  description: string;
  status: number;
  created_at: string;
}

interface PlatformAPIParam {
  id: number;
  api_id: number;
  name: string;
  key: string;
  value: string;
  type: string;
  required: number;
  description: string;
  status: number;
  created_at: string;
}

const appStore = useAppStore();
const message = useMessage();
const { loading, data, pagination, handlePageChange, handlePageSizeChange, handleSearch } = useTable<PlatformAPI>();
const { visible, showModal, hideModal } = useModal();
const { formRef, formModel, rules, handleSubmit, resetForm } = useForm();

// 平台选项
const platformOptions = ref<{ label: string; value: number; code: string }[]>([]);
const accountOptions = ref<{ label: string; value: number }[]>([]);
// 参数相关状态
const apiId = ref(0);
const paramVisible = ref(false);
const paramFormRef = ref();
const paramData = ref<PlatformAPIParam[]>([]);
const paramLoading = ref(false);
const paramPagination = ref({
  page: 1,
  pageSize: 10,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 30, 40],
  onChange: (page: number) => {
    paramPagination.value.page = page;
    fetchAPIParams(paramData.value[0]?.api_id);
  },
  onUpdatePageSize: (pageSize: number) => {
    paramPagination.value.pageSize = pageSize;
    paramPagination.value.page = 1;
    fetchAPIParams(paramData.value[0]?.api_id);
  }
});

// 表格列定义
const columns: DataTableColumns<PlatformAPI> = [
  {
    type: 'selection',
    align: 'center',
    width: 48
  },
  {
    key: 'name',
    title: '接口名称',
    align: 'center',
    width: 120
  },
  {
    key: 'platform_id',
    title: '平台',
    align: 'center',
    width: 60,
    render(row: PlatformAPI) {
      const platform = platformOptions.value.find(p => p.value === row.platform_id);
      return platform?.label || row.platform_id;
    }
  },
  {
    key: 'url',
    title: '接口地址',
    align: 'center',
    width: 200
  },
  {
    key: 'callback_url',
    title: '回调地址',
    align: 'center',
    width: 200
  },
  {
    key: 'status',
    title: '状态',
    align: 'center',
    width: 80,
    render(row: PlatformAPI) {
      return row.status === 1 ? (
        <NTag type="success" size="small">启用</NTag>
      ) : (
        <NTag type="error" size="small">禁用</NTag>
      );
    }
  },
  {    key: 'operate',    title: '操作',    align: 'center',    width: 200,    render(row: PlatformAPI) {      return (        <div class="operation-buttons">          <NButton type="primary" ghost size="small" onClick={() => handleEdit(row)} class="op-btn">
            编辑
          </NButton>          <NButton type="info" ghost size="small" onClick={() => showParamDialog(row)} class="op-btn">
            套餐配置
          </NButton>          <NPopconfirm onPositiveClick={() => handleDelete(row)}>
            {{
              default: () => '确认删除？',
              trigger: () => (
                <NButton type="error" ghost size="small" class="op-btn">
                  删除
                </NButton>
              )
            }}
          </NPopconfirm>        </div>      );    }  }
];

// 参数表格列定义
const paramColumns: DataTableColumns<PlatformAPIParam> = [
  {
    key: 'name',
    title: '套餐名称',
    align: 'center',
    width: 120
  },
  {
    key: 'cost',
    title: '成本',
    align: 'center',
    width: 60
  },
  {
    key: 'product_id',
    title: '产品ID',
    align: 'center',
    width: 90
  },

  {
    key: 'par_value',
    title: '面值',
    align: 'center',
    width: 80,
  },
  {
    key: 'price',
    title: '价格',
    align: 'center',
    width: 80,
  },
  {
    key: 'allow_provinces',
    title: '允许省份',
    align: 'center',
    width: 120,
  },
  {
    key: 'forbid_provinces',
    title: '禁止省份',
    align: 'center',
    width: 120,
  },
  {    key: 'operate',    title: '操作',    align: 'center',    width: 120,    render(row: PlatformAPIParam) {      return (        <div class="param-operation-buttons">          <NButton type="primary" ghost size="small" onClick={() => paramFormRef.value?.edit(row)} class="param-op-btn">
            编辑
          </NButton>          <NPopconfirm onPositiveClick={() => handleDeleteParam(row)}>
            {{
              default: () => '确认删除？',
              trigger: () => (
                <NButton type="error" ghost size="small" class="param-op-btn">
                  删除
                </NButton>
              )
            }}
          </NPopconfirm>        </div>      );    }  }
];

// 搜索表单
const searchForm = ref({
  name: '',
  platform_id: null as number | null,
  status: null as number | null
});

// 获取平台列表
const fetchPlatforms = async () => {
  try {
    const res = await request({
      url: '/platform/list',
      method: 'GET',
      params: {
        page: 1,
        page_size: 100
      }
    });
    if (res.data) {
      platformOptions.value = res.data.list.map((item: any) => ({
        label: item.name,
        value: item.id,
        code: item.code
      }));
    }
  } catch (error) {
    message.error('获取平台列表失败');
  }
};

// 获取账号列表
const fetchAccounts = async (platformId: number) => {
  try {
    const res = await request({
      url: '/platform/account/list',
      method: 'GET',
      params: {
        platform_id: platformId,
        page: 1,
        page_size: 100
      }
    }); 
    if (res.data) {
      accountOptions.value = res.data.items.map((item: any) => ({
        label: item.account_name,
        value: item.id
      }));
    }
  } catch (error) {
    message.error('获取账号列表失败');
  }
};

// 处理平台选择
const handlePlatformChange = (platformId: number) => {
  // 获取选中平台的code
  const selectedPlatform = platformOptions.value.find(p => p.value === platformId);
  if (selectedPlatform) {
    formModel.value.code = selectedPlatform.code;
  }
  // 获取账号列表
  fetchAccounts(platformId);
};

// 获取接口列表
const fetchAPIs = async () => {
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
      url: '/platform/api',
      method: 'GET',
      params
    });
    if (res.data) {
      data.value = res.data.list;
      pagination.value.itemCount = res.data.total;
    }
  } catch (error) {
    message.error('获取接口列表失败');
  } finally {
    loading.value = false;
  }
};

// 编辑接口
const handleEdit = (row: PlatformAPI) => {
  formModel.value = { ...row };
  formModel.value.extra_params = "";
  showModal();
};

// 删除接口
const handleDelete = async (row: PlatformAPI) => {
  try {
    await request({
      url: `/platform/api/${row.id}`,
      method: 'DELETE'
    });
    message.success('删除成功');
    fetchAPIs();
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
        url: `/platform/api/${formModel.value.id}`,
        method: 'PUT',
        data: formModel.value
      });
      message.success('更新成功');
    } else {
      await request({
        url: '/platform/api',
        method: 'POST',
        data: formModel.value
      });
      message.success('创建成功');
    }
    hideModal();
    resetForm();
    fetchAPIs();
  } catch (error) {
    message.error('操作失败');
  }
};

// 重置搜索表单
const handleReset = () => {
  searchForm.value = {
    name: '',
    platform_id: null,
    status: null
  };
  fetchAPIs();
};

// 新增接口
const handleAdd = () => {
  resetForm();
  showModal();
};

// 添加这些处理函数
const onPageChange = (page: number) => {
  pagination.value.page = page;
  fetchAPIs();
};

const onPageSizeChange = (pageSize: number) => {
  pagination.value.pageSize = pageSize;
  pagination.value.page = 1;
  fetchAPIs();
};

// 获取接口参数列表
const fetchAPIParams = async (apiId: number) => {
  try {
    paramLoading.value = true;
    const { page, pageSize } = paramPagination.value;
    
    const res = await request({
      url: '/platform/api/params',
      method: 'GET',
      params: {
        api_id: apiId,
        page,
        page_size: pageSize
      }
    });
    if (res.data) {
      paramData.value = res.data.list;
      paramPagination.value.itemCount = res.data.total;
    }
  } catch (error) {
    message.error('获取接口参数列表失败');
  } finally {
    paramLoading.value = false;
  }
};

// 删除接口参数
const handleDeleteParam = async (row: PlatformAPIParam) => {
  try {
    await request({
      url: `/platform/api/params/${row.id}`,
      method: 'DELETE'
    });
    message.success('删除成功');
    fetchAPIParams(row.api_id);
  } catch (error) {
    message.error('删除失败');
  }
};

// 显示参数配置对话框
const showParamDialog = (api: PlatformAPI) => {
  paramVisible.value = true;
  // 重置分页
  paramPagination.value.page = 1;
  // 获取参数列表
  fetchAPIParams(api.id);
  apiId.value = api.id;
};

// 参数分页变化
const onParamPageChange = (page: number) => {
  paramPagination.value.page = page;
  if (paramData.value.length > 0) {
    fetchAPIParams(paramData.value[0].api_id);
  }
};

const onParamPageSizeChange = (pageSize: number) => {
  paramPagination.value.pageSize = pageSize;
  paramPagination.value.page = 1;
  if (paramData.value.length > 0) {
    fetchAPIParams(paramData.value[0].api_id);
  }
};

// 参数表单提交成功
const handleParamSuccess = () => {
  fetchAPIParams(paramData.value[0]?.api_id);
};

onMounted(() => {
  fetchPlatforms();
  fetchAPIs();
});
</script>

<style scoped>
.min-h-500px {
  min-height: 500px;
}
.flex-col-stretch {
  display: flex;
  flex-direction: column;
}
.gap-16px {
  gap: 16px;
}
.mb-16px {
  margin-bottom: 16px;
}
.pr-24px {
  padding-right: 24px;
}
.w-full {
  width: 100%;
}
.lt-sm\:overflow-auto {
  @media (max-width: 640px) {
    overflow: auto;
  }
}
.overflow-hidden {
  overflow: hidden;
}
.flex-wrap {
  flex-wrap: wrap;
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
}
.sm\:h-full {
  @media (min-width: 640px) {
    height: 100%;
  }
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
.min-h-400px {
  min-height: 400px;
}

/* 移动端操作按钮优化 */
.operation-buttons {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-wrap: wrap;
  gap: 8px;
}

.op-btn {
  min-width: auto;
  padding: 0 8px;
}

.param-operation-buttons {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-wrap: wrap;
  gap: 8px;
}

.param-op-btn {
  min-width: auto;
  padding: 0 8px;
}

/* 统一按钮样式，不区分设备类型 */
</style>