export interface OrderException {
  id: number;
  order_id: string;
  exception_type: string;
  exception_reason: string;
  data: string;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface BalanceVerificationExceptionData {
  expected_balance: number;
  actual_balance: number;
  phone: string;
  amount: number;
  platform_name: string;
}

export interface OrderExceptionListParams {
  page: number;
  page_size: number;
  order_id?: string;
  exception_type?: string;
  status?: string;
  start_date?: string;
  end_date?: string;
}

export interface OrderExceptionListResponse {
  list: OrderException[];
  total: number;
  page: number;
  page_size: number;
}

export interface OrderExceptionStatistics {
  total_count: number;
  pending_count: number;
  processing_count: number;
  resolved_count: number;
  ignored_count: number;
}

export interface UpdateOrderExceptionStatusRequest {
  status: string;
}
