/** 后端统一响应体 */
export interface ApiResponse<T = unknown> {
  error_msg: string;
  data: T;
}

/** 分页查询参数 */
export interface PageParams {
  current: number;
  size: number;
}
