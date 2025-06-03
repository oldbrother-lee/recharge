export interface User {
  id: number;
  username: string;
  nickname: string;
  email: string;
  phone: string;
  status: number;
  created_at: string;
  balance: number;
  credit_limit: number;
  credit_used: number;
}

export interface UserListRequest {
  page: number;
  page_size: number;
  user_name?: string;
  phone?: string;
  email?: string;
  status?: number;
  balance_min?: number;
  balance_max?: number;
}

export interface UserListResponse {
  records: User[];
  total: number;
  current: number;
  size: number;
}

export interface BalanceRechargeRequest {
  user_id: number;
  amount: number;
  remark: string;
}

export interface BalanceDeductRequest {
  user_id: number;
  amount: number;
  style: number;
  remark: string;
}

export interface CreditSetRequest {
  user_id: number;
  creditLimit: number;
  remark: string;
 
} 