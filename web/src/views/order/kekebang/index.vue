<script setup lang="ts">
import OrderList from '../components/OrderList.vue';

const kekebang = 'kekebang';
</script>

<template>
  <OrderList :platform_code="kekebang" />
</template> 