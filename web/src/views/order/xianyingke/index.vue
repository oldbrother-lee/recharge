<script setup lang="ts">
import OrderList from '../components/OrderList.vue';

// 闲赢客平台代码，用于后端筛选不同来源的订单
const xianyinke = 'xianyinke';
</script>

<template>
  <OrderList :platform_code="xianyinke" />
</template>
