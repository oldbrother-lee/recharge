import { request } from '@/service/request';
import type {
  BalanceDeductRequest,
  BalanceRechargeRequest,
  CreditSetRequest,
  User,
  UserListRequest,
  UserListResponse
} from '@/typings/api/user';

/** 获取用户列表 */
export const getUserList = (params: UserListRequest) => {
  return request({
    url: '/user/list',
    method: 'GET',
    params: {
      current: params.page,
      size: params.page_size,
      username: params.user_name,
      phone: params.phone,
      email: params.email,
      status: params.status,
      balance_min: params.balance_min,
      balance_max: params.balance_max
    }
  });
};

/** 余额充值 */
export const rechargeBalance = (data: BalanceRechargeRequest) => {
  return request({
    url: '/balance/recharge',
    method: 'POST',
    data
  });
};

/** 余额扣款 */
export const deductBalance = (data: BalanceDeductRequest) => {
  return request({
    url: '/balance/deduct',
    method: 'POST',
    data
  });
};

/** 设置授信额度 */
export const setUserCredit = (data: CreditSetRequest) => {
  return request({
    url: '/credit/set',
    method: 'POST',
    data
  });
};
