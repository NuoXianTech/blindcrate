/**
 * API基础响应格式
 * @template T - 响应数据类型
 */
export interface ApiResponse<T = unknown> {
  /** 错误信息，成功时为空字符串 */
  error_msg: string;
  /** 响应数据 */
  data: T;
}

/** 分页查询参数 */
export interface PageParams {
  current: number;
  size: number;
}

/** WebSocket 消息, 推送不带 id, 请求的回复带回请求的 id */
export interface WsMessage<T = unknown> {
  /** 格式为 <app>:<事件>, ping、pong、error 为内置类型 */
  type: string;
  id?: string;
  /** 处理失败的原因, 成功时没有 */
  error_msg?: string;
  data?: T;
}

/** 后端登记的代码和名称, 如流水的业务类型、操作日志的操作类型 */
export interface CodeLabel {
  code: string;
  label: string;
}
