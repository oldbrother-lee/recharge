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
          <NFormItem label="类型名称" path="type_name">
            <NInput v-model:value="searchForm.type_name" placeholder="请输入类型名称" style="min-width: 200px" />
          </NFormItem>
          <NFormItem label="分类" path="typec_id">
            <NSelect
              v-model:value="searchForm.typec_id"
              :options="categoryOptions"
              placeholder="请选择分类"
              clearable
              style="min-width: 200px"
            />
          </NFormItem>
          <NFormItem label="状态" path="status">
            <NSelect
              v-model:value="searchForm.status"
              :options="[
                { label: '启用', value: 1 },
                { label: '禁用', value: 0 }
              ]"
              placeholder="请选择状态"
              clearable
              style="min-width: 100px"
            />
          </NFormItem>
          <NFormItem>
            <NSpace>
              <NButton type="primary" @click="handleSearch(fetchTypes)">
                搜索
              </NButton>
              <NButton @click="handleReset">重置</NButton>
            </NSpace>
          </NFormItem>
        </NForm>
      </NCard>
  
      <!-- 数据表格 -->
      <NCard :title="'产品类型管理'" :bordered="false" size="small" class="sm:flex-1-hidden card-wrapper">
        <template #header-extra>
          <NSpace>
            <NButton type="primary" @click="handleReset(); showModal()">
              新增类型
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
          @update:page="handlePageChange"
          @update:page-size="handlePageSizeChange"
          class="sm:h-full"
        />
      </NCard>
  
      <!-- 新增/编辑弹窗 -->
      <NModal
        v-model:show="visible"
        preset="dialog"
        :title="formModel.id ? '编辑类型' : '新增类型'"
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
          <NFormItem label="类型名称" path="type_name">
            <NInput v-model:value="formModel.type_name" placeholder="请输入类型名称" style="min-width: 200px" />
          </NFormItem>
          <NFormItem label="分类" path="typec_id">
            <NSelect
              v-model:value="formModel.typec_id"
              :options="categoryOptions"
              placeholder="请选择分类"
              style="min-width: 200px"
            />
          </NFormItem>
          <NFormItem label="状态" path="status">
            <NSwitch v-model:value="formModel.status" :checked-value="1" :unchecked-value="0" />
          </NFormItem>
          <NFormItem label="排序" path="sort">
            <NInputNumber v-model:value="formModel.sort" placeholder="请输入排序" style="min-width: 200px" />
          </NFormItem>
          <NFormItem label="账户类型" path="account_type">
            <NSelect
              v-model:value="formModel.account_type"
              :options="[
                { label: '充值', value: 1 },
                { label: '授信', value: 2 }
              ]"
              placeholder="请选择账户类型"
              style="min-width: 200px"
            />
          </NFormItem>
          <NFormItem label="提示文档" path="tishi_doc">
            <NInput
              v-model:value="formModel.tishi_doc"
              type="textarea"
              placeholder="请输入提示文档"
              style="min-width: 200px"
            />
          </NFormItem>
          <NFormItem label="图标" path="icon">
            <NInput v-model:value="formModel.icon" placeholder="请输入图标" style="min-width: 200px" />
          </NFormItem>
        </NForm>
        <template #action>
          <NSpace>
            <NButton @click="hideModal">取消</NButton>
            <NButton type="primary" @click="handleFormSubmit">确定</NButton>
          </NSpace>
        </template>
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
  import { NButton, NPopconfirm, NCard, NForm, NFormItem, NSpace, NInput, NSelect, NSwitch, NInputNumber,NTag } from 'naive-ui';
  import { useAppStore } from '@/store/modules/app';
  
  interface ProductType {
    id: number;
    type_name: string;
    typec_id: number;
    status: number;
    sort: number;
    account_type: string;
    tishi_doc: string;
    icon: string;
    created_at: string;
  }
  
  interface Category {
    id: number;
    name: string;
  }
  
  const appStore = useAppStore();
  const message = useMessage();
  const { loading, data, pagination, handlePageChange, handlePageSizeChange, handleSearch } = useTable<ProductType>();
  const { visible, showModal, hideModal } = useModal();
  const { formRef, formModel, rules, handleSubmit, resetForm } = useForm();
  
  // 分类选项
  const categoryOptions = ref<{ label: string; value: number }[]>([]);
  
  // 获取分类列表
  const fetchCategories = async () => {
    try {
      const res = await request({
        url: '/product-type/categories',
        method: 'GET',
        params: {
          page: 1,
          page_size: 10000
        }
      });
      if (res.data) {
        categoryOptions.value = res.data.items.map((item: Category) => ({
          label: item.name,
          value: item.id
        }));
      }
    } catch (error) {
      message.error('获取分类列表失败');
    }
  };
  
  // 表格列定义
  const columns: DataTableColumns<ProductType> = [
    {
      type: 'selection',
      align: 'center',
      width: 48
    },
    {
      key: 'type_name',
      title: '类型名称',
      align: 'center',
      width: 120
    },
    {
      key: 'typec_id',
      title: '分类',
      align: 'center',
      width: 120,
      render(row: ProductType) {
        const category = categoryOptions.value.find(item => item.value === row.typec_id);
        return category ? category.label : '-';
      }
    },
    {
      key: 'status',
      title: '状态',
      align: 'center',
      width: 80,
      render(row: ProductType) {
        return row.status === 1 ? (
        <NTag type="success" size="small">启用</NTag>
      ) : (
        <NTag type="error" size="small">禁用</NTag>
      );
      }
    },
    {
      key: 'sort',
      title: '排序',
      align: 'center',
      width: 80
    },
    {
      key: 'created_at',
      title: '创建时间',
      align: 'center',
      width: 180,
      render(row: ProductType) {
        return new Date(row.created_at).toLocaleString();
      }
    },
    {
      key: 'operate',
      title: '操作',
      align: 'center',
      width: 130,
      render(row: ProductType) {
        return (
          <div class="flex-center gap-8px">
            <NButton type="primary" ghost size="small" onClick={() => handleEdit(row)}>
              编辑
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
    type_name: '',
    typec_id: null as number | null,
    status: null as number | null
  });
  
  // 获取产品类型列表
  const fetchTypes = async () => {
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
        url: '/product-type/list',
        method: 'GET',
        params
      });
      if (res.data) {
        data.value = res.data.items;
        pagination.value.itemCount = res.data.total;
      }
    } catch (error) {
      message.error('获取产品类型列表失败');
    } finally {
      loading.value = false;
    }
  };
  
  // 编辑类型
  const handleEdit = (row: ProductType) => {
    formModel.value = { ...row };
    showModal();
  };
  
  // 删除类型
  const handleDelete = async (row: ProductType) => {
    try {
      await request({
        url: `/product-type/${row.id}`,
        method: 'DELETE'
      });
      message.success('删除成功');
      fetchTypes();
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
          url: `/product-type/${formModel.value.id}`,
          method: 'PUT',
          data: formModel.value
        });
        message.success('更新成功');
      } else {
        await request({
          url: '/product-type',
          method: 'POST',
          data: formModel.value
        });
        message.success('创建成功');
      }
      hideModal();
      fetchTypes();
    } catch (error) {
      message.error('操作失败');
    }
  };
  
  // 重置搜索表单
  const handleReset = () => {
    searchForm.value = {
      type_name: '',
      typec_id: null,
      status: null
    };
    fetchTypes();
  };
  
  onMounted(() => {
    fetchCategories();
    fetchTypes();
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
  .flex-center {
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .gap-8px {
    gap: 8px;
  }
  </style>