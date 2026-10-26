import type { AxiosRequestConfig } from "axios";
import { apiClient } from "./api-client";
import type { ApiResponse } from "./types";

/**
 * 服务基类, 各模块的 service 继承它
 * 自动剥掉响应外壳, 直接返回 data
 */
export class BaseService {
  /** 模块路径前缀, 如 "/auth" */
  protected static readonly basePath: string = "";

  protected static url(path: string): string {
    return `${this.basePath}${path}`;
  }

  protected static async get<T>(path: string, config?: AxiosRequestConfig): Promise<T> {
    const res = await apiClient.get<ApiResponse<T>>(this.url(path), config);
    return res.data.data;
  }

  protected static async post<T>(path: string, body?: unknown, config?: AxiosRequestConfig): Promise<T> {
    const res = await apiClient.post<ApiResponse<T>>(this.url(path), body, config);
    return res.data.data;
  }

  protected static async put<T>(path: string, body?: unknown, config?: AxiosRequestConfig): Promise<T> {
    const res = await apiClient.put<ApiResponse<T>>(this.url(path), body, config);
    return res.data.data;
  }

  protected static async patch<T>(path: string, body?: unknown, config?: AxiosRequestConfig): Promise<T> {
    const res = await apiClient.patch<ApiResponse<T>>(this.url(path), body, config);
    return res.data.data;
  }

  protected static async delete<T>(path: string, config?: AxiosRequestConfig): Promise<T> {
    const res = await apiClient.delete<ApiResponse<T>>(this.url(path), config);
    return res.data.data;
  }
}
