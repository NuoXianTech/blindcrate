import axios, {AxiosError} from 'axios';
import type {ApiResponse} from './types';

/**
 * 未登录(接口返回 401 或 WebSocket 登录失效)时派发的事件, 由登录状态相关组件监听后跳转登录
 * WebSocket 派发时 detail 为断开原因, 如被强制下线
 */
export const UNAUTHORIZED_EVENT = 'blindcrate:unauthorized';

/** 请求失败的错误, message 为后端返回的 error_msg, 没有时为按失败原因生成的提示 */
export class ApiError extends Error {
  status?: number;
  /** 后端随错误返回的 data, 如导入配置包冲突时每一项的结果 */
  data?: unknown;

  constructor(message: string, status?: number, data?: unknown) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.data = data;
  }
}

/**
 * API 客户端
 * 请求同域的 /api, 由 next.config.ts 的 rewrites 转发到后端, cookie 会自动带上
 */
export const apiClient = axios.create({
  baseURL: '/api/v1',
  timeout: 15000,
  withCredentials: true,
  headers: {'Content-Type': 'application/json'},
});

apiClient.interceptors.response.use(
    (response) => response,
    async (error: AxiosError<ApiResponse | Blob>) => {
      const status = error.response?.status;
      if (status === 401 && typeof window !== 'undefined') {
        window.dispatchEvent(new Event(UNAUTHORIZED_EVENT));
      }

      let body = error.response?.data;
      // 下载文件的请求失败时响应体也是 Blob, 读出其中的 JSON
      if (body instanceof Blob) {
        body = await body.text().then((text) => JSON.parse(text) as ApiResponse).catch(() => undefined);
      }
      let message = body?.error_msg;
      if (!message) {
        if (error.code === 'ECONNABORTED') {
          message = '服务器响应超时, 请稍后重试';
        } else if (!error.response) {
          message = '无法连接服务器, 请检查网络后重试';
        } else if (status === 429) {
          message = '操作太频繁, 请稍后再试';
        } else if (status && status >= 500) {
          message = '服务器暂时不可用, 请稍后重试';
        } else {
          message = `请求没有成功 (HTTP ${status}), 请刷新页面后重试`;
        }
      }
      return Promise.reject(new ApiError(message, status, body?.data));
    },
);

/** 把任意错误转成给用户看的提示 */
export function toErrorMessage(err: unknown): string {
  return err instanceof ApiError ? err.message : '操作没有完成, 请刷新页面后重试';
}
