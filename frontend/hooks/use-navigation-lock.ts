'use client';

import {useEffect, useSyncExternalStore} from 'react';

/** 正在锁定导航的组件数, 全局共用一份, 由 useLockNavigation 增减 */
let holders = 0;
const listeners = new Set<() => void>();

function subscribe(listener: () => void) {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

function update(delta: number) {
  holders += delta;
  for (const listener of listeners) listener();
}

/** 是否锁定了导航; 锁定时 dock、余额栏等离开当前页面的入口不可用 */
export function useNavigationLocked() {
  return useSyncExternalStore(subscribe, () => holders > 0, () => false);
}

/**
 * locked 为 true 时锁定导航, 变为 false 或组件隐藏、卸载时解除, 如大厅匹配期间不能离开大厅
 * 只锁住页面上的入口, 拦不住浏览器的前进后退
 */
export function useLockNavigation(locked: boolean) {
  useEffect(() => {
    if (!locked) return;
    update(1);
    return () => update(-1);
  }, [locked]);
}
