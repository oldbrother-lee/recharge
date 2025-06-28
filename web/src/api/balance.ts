import { request } from '@/service/request';

/** 余额充值 */
export function rechargeBalance(data: {
user_id: number;
  amount: number;
  remark: string;
}) {
  return request({ url: '/api/v1/balance/recharge', method: 'post', data });
}

/** 余额扣减 */
export function deductBalance(data: {
  user_id: number;
  amount: number;
  remark: string;
}) {
  return request({ url: '/api/v1/balance/deduct', method: 'post', data });
}

/** 获取余额日志 */
export function getBalanceLogs(params: {
  user_id?: number;
  page?: number;
  page_size?: number;
  start_time?: string;
  end_time?: string;
}) {
  return request({ url: '/api/v1/balance/logs', method: 'get', params });
}