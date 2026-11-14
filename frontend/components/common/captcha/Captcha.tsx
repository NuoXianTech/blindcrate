'use client';

import {useCallback, useEffect, useEffectEvent, useImperativeHandle, useRef, useState, type ReactNode, type Ref} from 'react';
import {cn} from 'cn';
import {useSite} from '@/components/common/site';
import {check} from '@/hooks/use-submit';

/** 需要人机验证的场景, 与后端 captcha 的场景一致, 每个场景在系统设置中单独开关 */
export type CaptchaScene = 'login' | 'register' | 'checkin' | 'shop';

/** always 始终显示组件; interaction-only 只在需要用户点击时显示, 多数情况下在后台自动完成 */
type CaptchaAppearance = 'always' | 'interaction-only';

/** Cloudflare Turnstile 的脚本, 由组件自己调用 render; 脚本和验证用的 iframe 都由浏览器直接从 Cloudflare 加载, 不经过后端 */
const TURNSTILE_SRC = 'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit';

/** 脚本超过这个时间还没有加载完时给出提示, 单位毫秒 */
const SLOW_AFTER = 8000;

/** 站点密钥或域名配置有误时的错误码, 重试也不会通过 */
const CONFIG_ERRORS = ['110100', '110110', '110200', '400020', '400070'];

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

/** 正在进行或已经完成的脚本加载, 页面上的所有组件共用; 失败后清空, 下一次调用重新加载 */
let loading: Promise<void> | null = null;

/**
 * 加载 Turnstile 的脚本, 完成后 window.turnstile 可用
 * 不用 next/script: 它不通知在脚本加载期间才挂载的组件, 加载失败后也不会再加载, 只能刷新页面
 */
function loadTurnstile(): Promise<void> {
  if (window.turnstile) return Promise.resolve();
  loading ??= new Promise<void>((resolve, reject) => {
    const script = document.createElement('script');
    const fail = () => {
      script.remove();
      loading = null;
      reject(new Error('turnstile script failed to load'));
    };
    script.src = TURNSTILE_SRC;
    script.async = true;
    // 脚本被浏览器插件等替换成空内容时也会触发 load
    script.onload = () => (window.turnstile ? resolve() : fail());
    script.onerror = fail;
    document.head.appendChild(script);
  });
  return loading;
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
 * Cloudflare Turnstile 人机验证组件; 脚本加载不出来或验证出错时在组件下方给出提示和重试, 不让提交按钮无故不可用
 */
export function Captcha({siteKey, onToken, appearance = 'always', className, ref}: CaptchaProps) {
  const box = useRef<HTMLDivElement>(null);
  const widgetId = useRef<string | null>(null);
  const [status, setStatus] = useState<'loading' | 'slow' | 'failed' | 'ready'>('loading');
  /** 每次重新加载脚本时加一 */
  const [attempt, setAttempt] = useState(0);
  /** Turnstile 给出的错误码 */
  const [error, setError] = useState('');
  const emit = useEffectEvent(onToken);

  useEffect(() => {
    let active = true;
    // 连接被拖住时浏览器要很久才报错, 先给出提示
    const timer = setTimeout(() => setStatus('slow'), SLOW_AFTER);
    void loadTurnstile()
        .then(() => 'ready' as const, () => 'failed' as const)
        .then((result) => {
          clearTimeout(timer);
          if (active) setStatus(result);
        });
    return () => {
      active = false;
      clearTimeout(timer);
    };
  }, [attempt]);

  useEffect(() => {
    if (status !== 'ready' || !box.current || !window.turnstile) return;
    const turnstile = window.turnstile;
    const id = turnstile.render(box.current, {
      'sitekey': siteKey,
      // 站点只有暗色主题
      'theme': 'dark',
      'size': 'flexible',
      'language': 'zh-cn',
      'appearance': appearance,
      'callback': (token: string) => {
        setError('');
        emit(token);
      },
      'expired-callback': () => emit(''),
      // interaction-only 时组件不显示, 出错只能靠下方的提示
      'error-callback': (code: string) => {
        setError(String(code));
        emit('');
      },
    });
    widgetId.current = id;
    return () => {
      widgetId.current = null;
      turnstile.remove(id);
      emit('');
    };
  }, [status, siteKey, appearance]);

  useImperativeHandle(ref, () => ({
    reset: () => {
      if (widgetId.current) window.turnstile?.reset(widgetId.current);
    },
  }), []);

  const retry = () => {
    if (status === 'failed') {
      setStatus('loading');
      setAttempt((n) => n + 1);
      return;
    }
    setError('');
    if (widgetId.current) window.turnstile?.reset(widgetId.current);
  };

  const misconfigured = CONFIG_ERRORS.includes(error);
  let problem = '';
  if (status === 'slow') problem = '人机验证加载较慢, 请稍候';
  else if (status === 'failed') problem = '人机验证加载失败, 请检查网络';
  else if (misconfigured) problem = `人机验证配置有误(${error}), 请联系管理员`;
  else if (error) problem = `人机验证没有完成(${error})`;

  return (
    <>
      <div ref={box} className={cn('w-full', className)} />
      {problem && (
        <p className={cn('text-xs', status === 'slow' ? 'text-muted-foreground' : 'text-destructive')}>
          {problem}
          {status !== 'slow' && !misconfigured && (
            <button type="button" className="ml-2 text-primary underline-offset-4 hover:underline" onClick={retry}>
              重试
            </button>
          )}
        </p>
      )}
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
