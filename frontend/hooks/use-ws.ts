'use client';

import {useEffect, useEffectEvent, useSyncExternalStore} from 'react';
import {wsClient} from '@/lib/services/core';

const subscribeStatus = (onChange: () => void) => wsClient.onStatus(onChange);
const getConnected = () => wsClient.connected;
const getServerConnected = () => false;

/**
 * 订阅 WebSocket 推送, 组件卸载时自动取消
 * handler 每次都能拿到最新的 props 和 state, 不需要 useCallback
 * 用法: useWsEvent<Data>('<app>:<事件>', (data) => { ... })
 */
export function useWsEvent<T = unknown>(type: string, handler: (data: T) => void) {
  const onEvent = useEffectEvent(handler);
  useEffect(() => wsClient.on<T>(type, (data) => onEvent(data)), [type]);
}

/** WebSocket 是否已连接; 推送不保证送达, 需要准确数据的页面在它变回 true 时重新拉取 */
export function useWsConnected() {
  return useSyncExternalStore(subscribeStatus, getConnected, getServerConnected);
}
