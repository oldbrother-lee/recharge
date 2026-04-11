import { request } from '@/service/request';

export interface AgentManualOrderPayload {
  mobile: string;
  product_id: number;
  out_trade_num: string;
  isp?: number;
  remark?: string;
}

/** 代理商手动下单（扣款与外部 API 一致） */
export function createAgentManualOrder(data: AgentManualOrderPayload) {
  return request({
    url: '/order/manual',
    method: 'POST',
    data
  });
}
