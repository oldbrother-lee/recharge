import { request } from '@/service/request';

/** 余额充值 */
export function rechargeBalance(data: { user_id: number; amount: number; remark: string }) {
  return request({ url: '/balance/recharge', method: 'post', data });
}

/** 余额扣减 */
export function deductBalance(data: { user_id: number; amount: number; remark: string }) {
  return request({ url: '/balance/deduct', method: 'post', data });
}

/** 获取余额流水（管理端） */
export function getBalanceLogs(params: { user_id: number; page?: number; page_size?: number }) {
  const page = params.page ?? 1;
  const pageSize = params.page_size ?? 20;
  return request({
    url: '/balance/logs',
    method: 'get',
    params: {
      user_id: params.user_id,
      offset: (page - 1) * pageSize,
      limit: pageSize
    }
  });
}

/** 查询平台余额 */
export function queryPlatformBalance(platform: string, accountId: number) {
  return request({
    url: `/platform-balance/${platform}`,
    method: 'get',
    params: { account_id: accountId }
  });
}
