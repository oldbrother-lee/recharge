import { request } from '@/service/request';

export function getOperatorStatistics() {
  return request({
    url: '/statistics/order/operator',
    method: 'GET'
  });
} 