import { request } from '@/utils/request';

/** 余额充值 */
export function rechargeBalance(data: {
user_id: number;
  amount: number;
  remark: string;
}) {
  return request.post('/api/v1/balance/recharge', data);
}

/** 余额扣款 */
export function deductBalance(data: {
    user_id: number;
  amount: number;
  style: number;
  remark: string;
}) {
  return request.post('/api/v1/balance/deduct', data);
}

/** 查询余额流水 */
export function getBalanceLogs(params: {
    user_id: number;
  page: number;
  pageSize: number;
}) {
  return request.get('/api/v1/balance/logs', { params });
} 