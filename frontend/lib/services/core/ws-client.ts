import {ApiError, UNAUTHORIZED_EVENT, apiClient} from './api-client';
import type {WsMessage} from './types';

/** 关闭码: 登录已失效或被强制下线, 不再重连 */
const CLOSE_UNAUTHORIZED = 4001;
/** 心跳间隔, 上一轮的 ping 到这一轮还没有回应就换一条新连接 */
const HEARTBEAT_INTERVAL = 20000;
/** 请求等待回复的最长时间 */
const REQUEST_TIMEOUT = 10000;
/** 重连等待从 1 秒开始翻倍, 最长 30 秒 */
const RECONNECT_MIN_DELAY = 1000;
const RECONNECT_MAX_DELAY = 30000;

type Listener = (data: unknown) => void;

interface PendingRequest {
  resolve: (data: unknown) => void;
  reject: (err: ApiError) => void;
  timer: ReturnType<typeof setTimeout>;
}

/** 同域的 /api 由 next.config.ts 的 rewrites 转发到后端, 握手时会自动带上 cookie */
function wsUrl() {
  const {protocol, host} = window.location;
  return `${protocol === 'https:' ? 'wss:' : 'ws:'}//${host}/api/v1/ws`;
}

/**
 * WebSocket 客户端, 全站共用一条连接, 登录后由 AuthProvider 建立
 * - 消息为 JSON {type, id, error_msg, data}, 推送不带 id, 请求的回复带回请求的 id
 * - 断线后自动重连; 登录失效或被强制下线时派发 UNAUTHORIZED_EVENT, 不再重连
 * - 推送不保证送达, 需要准确数据的页面在重连后重新拉取
 */
class WsClient {
  private socket: WebSocket | null = null;
  /** 是否需要保持连接: connect 后为 true, disconnect 或登录失效后为 false */
  private enabled = false;
  private opened = false;
  /** 收到任何消息都说明连接还活着, 心跳时检查 */
  private alive = false;
  private retries = 0;
  private seq = 0;
  private openTimer?: ReturnType<typeof setTimeout>;
  private heartbeatTimer?: ReturnType<typeof setInterval>;
  private readonly listeners = new Map<string, Set<Listener>>();
  private readonly statusListeners = new Set<() => void>();
  private readonly pending = new Map<string, PendingRequest>();

  /** 是否已连接 */
  get connected() {
    return this.opened;
  }

  /** 建立连接并保持, 断线后自动重连 */
  connect() {
    if (this.enabled) return;
    this.enabled = true;
    // 放到下一轮事件循环, 开发模式下 effect 连着执行 connect 和 disconnect 时不会白建一条连接
    this.scheduleOpen(0);
  }

  /** 断开连接, 不再重连 */
  disconnect() {
    this.enabled = false;
    this.retries = 0;
    clearTimeout(this.openTimer);
    this.close();
  }

  /** 订阅一类推送, 返回取消订阅的函数 */
  on<T = unknown>(type: string, listener: (data: T) => void): () => void {
    const fn = listener as Listener;
    const listeners = this.listeners.get(type) ?? new Set<Listener>();
    this.listeners.set(type, listeners);
    listeners.add(fn);
    return () => {
      listeners.delete(fn);
    };
  }

  /** 订阅连接状态变化, 返回取消订阅的函数 */
  onStatus(listener: () => void): () => void {
    this.statusListeners.add(listener);
    return () => {
      this.statusListeners.delete(listener);
    };
  }

  /**
   * 发送请求并等待回复, 返回回复的 data
   * 失败时抛出 ApiError, message 为后端返回的 error_msg; 等待中断线时服务端可能已经处理了请求
   */
  request<T = unknown>(type: string, data?: unknown): Promise<T> {
    return new Promise<T>((resolve, reject) => {
      const id = String(++this.seq);
      if (!this.send({type, id, data})) {
        reject(new ApiError('无法连接服务器, 请检查网络后重试'));
        return;
      }
      const timer = setTimeout(() => {
        this.pending.delete(id);
        reject(new ApiError('服务器响应超时, 请稍后重试'));
      }, REQUEST_TIMEOUT);
      this.pending.set(id, {resolve: (value) => resolve(value as T), reject, timer});
    });
  }

  private scheduleOpen(delay: number) {
    clearTimeout(this.openTimer);
    this.openTimer = setTimeout(() => this.open(), delay);
  }

  private scheduleReconnect() {
    if (!this.enabled) return;
    // 指数退避加随机抖动, 服务重启后客户端不会同时涌进来
    const delay = Math.min(RECONNECT_MAX_DELAY, RECONNECT_MIN_DELAY * 2 ** this.retries);
    this.retries++;
    this.scheduleOpen(delay / 2 + Math.random() * delay / 2);
  }

  private open() {
    const socket = new WebSocket(wsUrl());
    this.socket = socket;
    socket.onopen = () => {
      this.alive = true;
      this.heartbeatTimer = setInterval(() => this.heartbeat(), HEARTBEAT_INTERVAL);
      this.setOpened(true);
    };
    socket.onmessage = (event) => this.handle(event.data);
    socket.onclose = (event) => {
      const opened = this.opened;
      this.close();
      if (event.code === CLOSE_UNAUTHORIZED) {
        this.enabled = false;
        window.dispatchEvent(new CustomEvent(UNAUTHORIZED_EVENT, {detail: event.reason}));
        return;
      }
      if (!opened) {
        // 握手失败时拿不到状态码, 查一次登录状态
        apiClient.get('/auth/user-info').catch(() => {
          // 返回 401 时 api-client 已经派发了 UNAUTHORIZED_EVENT, 其他错误继续重连即可
        });
      }
      this.scheduleReconnect();
    };
  }

  /** 关闭当前连接, 结束心跳并让等待中的请求失败 */
  private close() {
    clearInterval(this.heartbeatTimer);
    const socket = this.socket;
    if (socket) {
      this.socket = null;
      // 不再处理这条连接的事件, 换新连接时不用等它的关闭握手
      socket.onopen = null;
      socket.onmessage = null;
      socket.onclose = null;
      socket.close();
    }
    for (const request of this.pending.values()) {
      clearTimeout(request.timer);
      request.reject(new ApiError('连接已断开, 请稍后重试'));
    }
    this.pending.clear();
    this.setOpened(false);
  }

  private heartbeat() {
    if (!this.alive) {
      // 上一轮的 ping 没有回应, 连接可能已经断了而浏览器还没发现
      this.close();
      this.scheduleReconnect();
      return;
    }
    this.alive = false;
    this.send({type: 'ping'});
  }

  private send(msg: WsMessage): boolean {
    const socket = this.socket;
    if (!socket || socket.readyState !== WebSocket.OPEN) return false;
    socket.send(JSON.stringify(msg));
    return true;
  }

  private handle(raw: unknown) {
    this.alive = true;
    // 收到消息才算连接可用, 握手后马上被关闭(如连接数过多)时重连等待会继续变长
    this.retries = 0;
    if (typeof raw !== 'string') return;
    let msg: WsMessage;
    try {
      msg = JSON.parse(raw);
    } catch {
      return;
    }

    // 请求的回复
    if (msg.id) {
      const request = this.pending.get(msg.id);
      if (!request) return;
      this.pending.delete(msg.id);
      clearTimeout(request.timer);
      if (msg.error_msg) {
        request.reject(new ApiError(msg.error_msg));
      } else {
        request.resolve(msg.data);
      }
      return;
    }

    // 推送
    const listeners = this.listeners.get(msg.type);
    if (!listeners) return;
    for (const listener of [...listeners]) {
      try {
        listener(msg.data);
      } catch (err) {
        // 一个订阅出错不影响其他订阅
        reportError(err);
      }
    }
  }

  private setOpened(opened: boolean) {
    if (this.opened === opened) return;
    this.opened = opened;
    this.statusListeners.forEach((listener) => listener());
  }
}

/** 全站共用的 WebSocket 客户端 */
export const wsClient = new WsClient();
