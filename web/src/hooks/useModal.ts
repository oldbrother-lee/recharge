import { ref } from 'vue';

export function useModal() {
  const visible = ref(false);

  const showModal = () => {
    visible.value = true;
  };

  const hideModal = () => {
    visible.value = false;
  };

  return {
    visible,
    showModal,
    hideModal
  };
} 