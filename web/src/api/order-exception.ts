import { request } from '@/service/request';

export interface OrderException {
  id: number;
  order_id: string;
  exception_type: string;
  exception_reason: string;
  data: any;
  status: string;
  create_time: string;
  update_time: string;
  order?: {
    id: string;
    phone: string;
    amount: number;
    platform_name: string;
    status: string;
    create_time: string;
  };
}

export interface OrderExceptionListParams {
  page?: number;
  pageSize?: number;
  order_id?: string;
  exception_type?: string;
  status?: string;
  start_date?: string;
  end_date?: string;
}

export interface OrderExceptionListResponse {
  list: OrderException[];
  total: number;
}

export interface OrderExceptionStatistics {
  total_count: number;
  pending_count: number;
  processing_count: number;
  resolved_count: number;
  ignored_count: number;
  balance_verification_count: number;
}

/**
 * 获取订单异常列表
 */
export function fetchOrderExceptionList(params: OrderExceptionListParams) {
  return request<OrderExceptionListResponse>({
    url: '/order-exceptions',
    method: 'GET',
    params
  });
}

/**
 * 根据ID获取订单异常详情
 */
export function fetchOrderExceptionById(id: number) {
  return request<OrderException>({
    url: `/order-exceptions/${id}`,
    method: 'GET'
  });
}

/**
 * 根据订单ID获取异常记录
 */
export function fetchOrderExceptionsByOrderId(orderId: string) {
  return request<OrderException[]>({
    url: `/orders/${orderId}/exceptions`,
    method: 'GET'
  });
}

/**
 * 更新订单异常状态
 */
export function updateOrderExceptionStatus(id: number, status: string, remark?: string) {
  return request({
    url: `/order-exceptions/${id}/status`,
    method: 'PUT',
    data: { status, remark }
  });
}

/**
 * 获取待处理异常数量
 */
export function fetchPendingExceptionCount() {
  return request<{ count: number }>({
    url: '/order-exceptions/pending-count',
    method: 'GET'
  });
}

/**
 * 获取异常统计信息
 */
export function fetchOrderExceptionStatistics(startDate?: string, endDate?: string) {
  return request<OrderExceptionStatistics>({
    url: '/order-exceptions/statistics',
    method: 'GET',
    params: {
      start_date: startDate,
      end_date: endDate
    }
  });
}