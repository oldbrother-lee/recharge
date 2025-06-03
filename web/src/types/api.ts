export interface ApiResponse<T = any> {
  code: number;
  data: T;
  message: string;
}

export interface PaginationParams {
  page: number;
  page_size: number;
}

export interface PaginatedResponse<T> {
  list: T[];
  total: number;
} 