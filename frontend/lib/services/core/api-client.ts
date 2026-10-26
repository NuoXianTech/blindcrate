import axios, { AxiosError } from "axios";
import type { ApiResponse } from "./types";

/** 未登录(401)时派发的事件, 由登录状态相关组件监听后跳转登录 */
export const UNAUTHORIZED_EVENT = "blindcrate:unauthorized";

/** 请求失败的错误, message 为后端返回的 error_msg */
export class ApiError extends Error {
  status?: number;

  constructor(message: string, status?: number) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

/**
 * API 客户端
 * 请求同域的 /api, 由 next.config.ts 的 rewrites 转发到后端, cookie 会自动带上
 */
export const apiClient = axios.create({
  baseURL: "/api/v1",
  timeout: 15000,
  withCredentials: true,
  headers: { "Content-Type": "application/json" },
});

apiClient.interceptors.response.use(
  (response) => response,
  (error: AxiosError<ApiResponse>) => {
    const status = error.response?.status;
    if (status === 401 && typeof window !== "undefined") {
      window.dispatchEvent(new Event(UNAUTHORIZED_EVENT));
    }

    let message = error.response?.data?.error_msg;
    if (!message) {
      if (error.code === "ECONNABORTED") {
        message = "请求超时, 请稍后再试";
      } else if (!error.response) {
        message = "网络连接失败, 请检查网络";
      } else {
        message = "请求失败, 请稍后再试";
      }
    }
    return Promise.reject(new ApiError(message, status));
  },
);
