<template>
  <n-modal v-model:show="visible" title="分配角色" preset="dialog" :mask-closable="false">
    <n-spin :show="loading">
      <n-checkbox-group v-model:value="checkedRoles">
        <n-space vertical>
          <n-checkbox v-for="role in allRoles" :key="role.id" :value="role.id">
            {{ role.name }}
          </n-checkbox>
        </n-space>
      </n-checkbox-group>
    </n-spin>
    <template #action>
      <n-space>
        <n-button @click="close">取消</n-button>
        <n-button type="primary" :loading="saving" @click="handleSave">保存</n-button>
      </n-space>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from 'vue';
import { NModal, NButton, NCheckbox, NCheckboxGroup, NSpace, NSpin, useMessage } from 'naive-ui';
import { request } from '@/service/request';

const props = defineProps<{ visible: boolean; user: any }>();
const emit = defineEmits(['update:visible', 'success']);

const visible = ref(props.visible);
watch(() => props.visible, v => (visible.value = v));
watch(visible, v => emit('update:visible', v));

const user = ref(props.user);
watch(() => props.user, v => (user.value = v));

const allRoles = ref<any[]>([]);
const checkedRoles = ref<number[]>([]);
const loading = ref(false);
const saving = ref(false);
const message = useMessage();

async function fetchRoles() {
  loading.value = true;
  try {
    // 获取所有角色
    const res = await request({ url: '/roles', method: 'GET' });
    console.log(res.data, "res++++++++")
    allRoles.value = res.data.list || [];
    // 获取用户已分配角色
    if (user.value?.id) {
      const res2 = await request({ url: `/users/${user.value.id}/roles`, method: 'GET' });
      checkedRoles.value = (res2.data || []).map((r: any) => r.id);
    }
  } catch (e) {
    message.error('获取角色信息失败');
  } finally {
    loading.value = false;
  }
}

watch(
  () => visible.value,
  v => {
    if (v && user.value?.id) fetchRoles();
  }
);

function close() {
  visible.value = false;
}

async function handleSave() {
  if (!user.value?.id) return;
  saving.value = true;
  try {
    await request({
      url: `/users/${user.value.id}/roles`,
      method: 'POST',
      data: checkedRoles.value
    });
    message.success('分配成功');
    emit('success');
    close();
  } catch (e) {
    message.error('分配失败');
  } finally {
    saving.value = false;
  }
}
</script> 