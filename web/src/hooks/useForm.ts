import { ref } from 'vue';
import type { FormRules } from 'naive-ui';

export function useForm() {
  const formRef = ref();
  const formModel = ref<Record<string, any>>({});
  const rules = ref<FormRules>({});

  const handleSubmit = async () => {
    await formRef.value?.validate();
  };

  const resetForm = () => {
    formModel.value = {};
    formRef.value?.restoreValidation();
  };

  return {
    formRef,
    formModel,
    rules,
    handleSubmit,
    resetForm
  };
} 