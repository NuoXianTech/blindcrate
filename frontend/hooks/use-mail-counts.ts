'use client';

import {useEffect, useSyncExternalStore} from 'react';
import {useWsConnected, useWsEvent} from '@/hooks/use-ws';
import services from '@/lib/services';
import {MAILBOX_MESSAGES, type MailCounts} from '@/lib/services/mailbox';

/** 信箱的未读数, 全局共用一份, 由 useMailCountsSync 刷新 */
let counts: MailCounts | null = null;
const listeners = new Set<() => void>();
/** 进行中的请求; 请求期间又要求刷新时, 结束后再拉一次, 避免漏掉期间的新邮件 */
let inflight: Promise<void> | null = null;
let again = false;
/** 每次重置加一, 丢弃重置前发出的请求的结果 */
let generation = 0;

/** 收到推送后最多延迟的毫秒数, 全服邮件会推给所有在线用户, 错开请求避免同时打到后端 */
export const PUSH_JITTER_MS = 2000;

/** 随机延迟后执行, 返回取消函数 */
export function afterJitter(fn: () => void) {
  const timer = setTimeout(fn, Math.random() * PUSH_JITTER_MS);
  return () => clearTimeout(timer);
}

function emit() {
  for (const listener of listeners) listener();
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

/** 重新获取未读数; 失败时保留上次的数据 */
export function refreshMailCounts(): Promise<void> {
  if (inflight) {
    again = true;
    return inflight;
  }
  const current = generation;
  const request = (async () => {
    do {
      again = false;
      try {
        const data = await services.mailbox.counts();
        if (current !== generation) return;
        counts = data;
        emit();
      } catch {
        // 未读数只是提示, 失败时不打扰用户
      }
    } while (again && current === generation);
  })().finally(() => {
    if (inflight === request) inflight = null;
  });
  inflight = request;
  return request;
}

/** 退出登录时清空 */
function resetMailCounts() {
  generation++;
  again = false;
  inflight = null;
  if (counts === null) return;
  counts = null;
  emit();
}

/** 信箱的未读和附件未领取的封数, 还没有获取时为 null */
export function useMailCounts() {
  return useSyncExternalStore(subscribe, () => counts, () => null);
}

/**
 * 保持未读数为最新: 登录后获取一次, 收到新邮件推送、ws 重连和切回页面时刷新
 * 只在常驻的组件中调用一次, 其他地方用 useMailCounts 读取
 */
export function useMailCountsSync(userID: number | undefined) {
  const connected = useWsConnected();

  useEffect(() => {
    if (!userID) return;
    return resetMailCounts;
  }, [userID]);

  // 登录后获取一次; 断线期间的推送会丢失, 重连后补一次
  useEffect(() => {
    if (userID) void refreshMailCounts();
  }, [userID, connected]);

  useWsEvent(MAILBOX_MESSAGES.new, () => {
    if (userID) afterJitter(() => void refreshMailCounts());
  });

  useEffect(() => {
    if (!userID) return;
    const onVisible = () => {
      if (document.visibilityState === 'visible') void refreshMailCounts();
    };
    document.addEventListener('visibilitychange', onVisible);
    return () => document.removeEventListener('visibilitychange', onVisible);
  }, [userID]);
}
