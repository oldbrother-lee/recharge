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
        <NFormItem label="手机号" path="phone">
          <NInput v-model:value="searchForm.phone" placeholder="请输入手机号" style="min-width: 200px" />
        </NFormItem>
        <NFormItem label="省份" path="province">
          <NInput v-model:value="searchForm.province" placeholder="请输入省份" style="min-width: 200px" />
        </NFormItem>
        <NFormItem label="城市" path="city">
          <NInput v-model:value="searchForm.city" placeholder="请输入城市" style="min-width: 200px" />
        </NFormItem>
        <NFormItem label="运营商" path="isp">
          <NSelect
            v-model:value="searchForm.isp"
            :options="[
              { label: '移动', value: '移动' },
              { label: '联通', value: '联通' },
              { label: '电信', value: '电信' }
            ]"
            placeholder="请选择运营商"
            clearable
            style="min-width: 100px"
          />
        </NFormItem>
        <NFormItem>
          <NSpace>
            <NButton type="primary" @click="handleSearch(fetchPhoneLocations)">
              搜索
            </NButton>
            <NButton @click="handleReset">重置</NButton>
          </NSpace>
        </NFormItem>
      </NForm>
    </NCard>

    <!-- 数据表格 -->
    <NCard :title="'手机归属地管理'" :bordered="false" size="small" class="sm:flex-1-hidden card-wrapper">
      <template #header-extra>
        <NSpace>
          <NButton type="primary" @click="handleReset(); showModal()">
            新增归属地
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
      :title="formModel.id ? '编辑归属地' : '新增归属地'"
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
        <NFormItem label="手机号" path="phone_number">
          <NInput v-model:value="formModel.phone_number" placeholder="请输入手机号" style="min-width: 200px" />
        </NFormItem>
        <NFormItem label="省份" path="province">
          <NInput v-model:value="formModel.province" placeholder="请输入省份" style="min-width: 200px" />
        </NFormItem>
        <NFormItem label="城市" path="city">
          <NInput v-model:value="formModel.city" placeholder="请输入城市" style="min-width: 200px" />
        </NFormItem>
        <NFormItem label="运营商" path="isp">
          <NSelect
            v-model:value="formModel.isp"
            :options="[
              { label: '移动', value: '1' },
              { label: '联通', value: '3' },
              { label: '电信', value: '2' }
            ]"
            placeholder="请选择运营商"
            style="min-width: 200px"
          />
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
import { NButton, NPopconfirm, NCard, NForm, NFormItem, NSpace, NInput, NSelect } from 'naive-ui';
import { useAppStore } from '@/store/modules/app';

const appStore = useAppStore();
const message = useMessage();
const { loading, data, pagination, handlePageChange, handlePageSizeChange, handleSearch } = useTable();
const { visible, showModal, hideModal } = useModal();
const { formRef, formModel, rules, handleSubmit, resetForm } = useForm();

// 表格列定义
const columns: DataTableColumns = [
  {
    type: 'selection',
    align: 'center',
    width: 48
  },
  {
    key: 'phone_number',
    title: '手机号',
    align: 'center',
    width: 120
  },
  {
    key: 'province',
    title: '省份',
    align: 'center',
    width: 100
  },
  {
    key: 'city',
    title: '城市',
    align: 'center',
    width: 100
  },
  {
    key: 'isp',
    title: '运营商',
    align: 'center',
    width: 80
  },
  {
    key: 'created_at',
    title: '创建时间',
    align: 'center',
    width: 180,
    render(row) {
      return new Date(row.created_at).toLocaleString();
    }
  },
  {
    key: 'operate',
    title: '操作',
    align: 'center',
    width: 130,
    render(row) {
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
  phone: '',
  province: '',
  city: '',
  isp: ''
});

// 获取手机归属地列表
const fetchPhoneLocations = async () => {
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
      url: '/phone-locations',
      method: 'GET',
      params
    });
    if (res.data) {
      data.value = res.data.items;
      pagination.value.itemCount = res.data.total;
    }
  } catch (error) {
    message.error('获取手机归属地列表失败');
  } finally {
    loading.value = false;
  }
};

// 编辑归属地
const handleEdit = (row) => {
  formModel.value = { ...row };
  showModal();
};

// 删除归属地
const handleDelete = async (row) => {
  try {
    await request({
      url: `/phone-locations/${row.id}`,
      method: 'DELETE'
    });
    message.success('删除成功');
    fetchPhoneLocations();
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
        url: `/phone-locations/${formModel.value.id}`,
        method: 'PUT',
        data: formModel.value
      });
      message.success('更新成功');
    } else {
      await request({
        url: '/phone-locations',
        method: 'POST',
        data: formModel.value
      });
      message.success('创建成功');
    }
    hideModal();
    fetchPhoneLocations();
  } catch (error) {
    message.error('操作失败');
  }
};

// 重置搜索表单
const handleReset = () => {
  searchForm.value = {
    phone: '',
    province: '',
    city: '',
    isp: ''
  };
  fetchPhoneLocations();
};

onMounted(() => {
  fetchPhoneLocations();
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