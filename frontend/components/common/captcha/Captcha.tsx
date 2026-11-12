'use client';

import {useCallback, useEffect, useEffectEvent, useImperativeHandle, useRef, useState, type ReactNode, type Ref} from 'react';
import Script from 'next/script';
import {cn} from 'cn';
import {useSite} from '@/components/common/site';
import {check} from '@/hooks/use-submit';

/** 需要人机验证的场景, 与后端 captcha 的场景一致, 每个场景在系统设置中单独开关 */
export type CaptchaScene = 'login' | 'register' | 'checkin' | 'shop';

/** always 始终显示组件; interaction-only 只在需要用户点击时显示, 多数情况下在后台自动完成 */
type CaptchaAppearance = 'always' | 'interaction-only';

/** Cloudflare Turnstile 的脚本, 由组件自己调用 render */
const TURNSTILE_SRC = 'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit';

interface TurnstileApi {
  render: (container: HTMLElement, options: Record<string, unknown>) => string;
  reset: (widgetId: string) => void;
  remove: (widgetId: string) => void;
}

declare global {
  interface Window {
    turnstile?: TurnstileApi;
  }
}

export interface CaptchaHandle {
  /** 令牌只能用一次, 用过后重置, 组件会重新验证并给出新的令牌 */
  reset: () => void;
}

interface CaptchaProps {
  siteKey: string;
  /** 验证通过时给出令牌, 令牌过期、出错或重置时给出空字符串 */
  onToken: (token: string) => void;
  appearance?: CaptchaAppearance;
  className?: string;
  ref?: Ref<CaptchaHandle>;
}

/**
 * Cloudflare Turnstile 人机验证组件
 */
export function Captcha({siteKey, onToken, appearance = 'always', className, ref}: CaptchaProps) {
  const box = useRef<HTMLDivElement>(null);
  const widgetId = useRef<string | null>(null);
  const [ready, setReady] = useState(false);
  const emit = useEffectEvent(onToken);

  useEffect(() => {
    if (!ready || !box.current || !window.turnstile) return;
    const turnstile = window.turnstile;
    const id = turnstile.render(box.current, {
      'sitekey': siteKey,
      // 站点只有暗色主题
      'theme': 'dark',
      'size': 'flexible',
      'language': 'zh-cn',
      'appearance': appearance,
      'callback': (token: string) => emit(token),
      'expired-callback': () => emit(''),
      'error-callback': () => emit(''),
    });
    widgetId.current = id;
    return () => {
      widgetId.current = null;
      turnstile.remove(id);
      emit('');
    };
  }, [ready, siteKey, appearance]);

  useImperativeHandle(ref, () => ({
    reset: () => {
      if (widgetId.current) window.turnstile?.reset(widgetId.current);
    },
  }), []);

  return (
    <>
      <div ref={box} className={cn('w-full', className)} />
      {/* 脚本只加载一次, onReady 在每次挂载时都会调用 */}
      <Script src={TURNSTILE_SRC} onReady={() => setReady(true)} />
    </>
  );
}

/**
 * 某个场景的人机验证, 用法:
 *   const captcha = useCaptcha('login');
 *   await services.auth.login(body, captcha.take());
 *   ... {captcha.widget} ...
 * 场景没有开启时 widget 为 null, take 返回 undefined, ready 始终为 true
 */
export function useCaptcha(scene: CaptchaScene, options?: { appearance?: CaptchaAppearance; className?: string }): {
  /** 可以提交: 没有开启或已拿到令牌 */
  ready: boolean;
  /** 取出令牌用于一次请求并重置组件; 还没有拿到令牌时中断提交, 需要在 useSubmit 的 submit 中调用 */
  take: () => string | undefined;
  widget: ReactNode;
} {
  const {captcha} = useSite();
  // 没有填写站点密钥时后端也不验证
  const enabled = captcha[`${scene}_enabled`] && captcha.site_key !== '';
  const [token, setToken] = useState('');
  const handle = useRef<CaptchaHandle>(null);

  const take = useCallback(() => {
    if (!enabled) return undefined;
    check(token ? null : '人机验证还没有完成, 请稍后再试');
    setToken('');
    handle.current?.reset();
    return token;
  }, [enabled, token]);

  const widget = enabled ? (
    <Captcha ref={handle} siteKey={captcha.site_key} onToken={setToken} appearance={options?.appearance} className={options?.className} />
  ) : null;
  return {ready: !enabled || token !== '', take, widget};
}
