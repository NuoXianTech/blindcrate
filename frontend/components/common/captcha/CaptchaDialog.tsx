'use client';

import {useCallback, useEffect, useLayoutEffect, useRef, useState, type ReactNode} from 'react';
import {Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle} from '@/components/ui/dialog';
import {useSite} from '@/components/common/site';
import {CancelledError} from '@/hooks/use-submit';
import services from '@/lib/services';
import {ApiError} from '@/lib/services/core';
import {Captcha, loadTurnstile, type CaptchaScene} from './Captcha';

/**
 * 人机验证弹窗的Props接口
 */
interface CaptchaDialogProps {
  siteKey: string;
  open: boolean;
  /** 验证通过时给出令牌, 由调用方关闭弹窗 */
  onVerified: (token: string) => void;
  /** 没有完成验证就关闭弹窗时调用 */
  onCancel: () => void;
}

/**
 * 人机验证弹窗: 打开后开始一次新的验证, 多数情况下自动通过, 需要时由用户点击
 */
export function CaptchaDialog({siteKey, open, onVerified, onCancel}: CaptchaDialogProps) {
  return (
    <Dialog open={open} onOpenChange={(next) => !next && onCancel()}>
      {/* 宽度跟随验证组件: 组件最窄 300px, 窄屏上不能被弹窗的最大宽度挤出去 */}
      <DialogContent className="w-fit max-w-none">
        <DialogHeader>
          <DialogTitle>人机验证</DialogTitle>
          <DialogDescription>验证通过后自动继续</DialogDescription>
        </DialogHeader>
        {/* 与验证组件一样大, 组件出现前后弹窗的大小不变 */}
        <div className="flex min-h-[65px] w-[300px] flex-col gap-2">
          {/* 每次打开都是新的组件和令牌 */}
          {open && <Captcha siteKey={siteKey} onToken={(token) => token && onVerified(token)} />}
        </div>
      </DialogContent>
    </Dialog>
  );
}

/** 正在等待结果的一次验证 */
interface Verification {
  siteKey: string;
  resolve: (token: string) => void;
  reject: (reason: Error) => void;
}

/**
 * 弹窗形式的人机验证, 用在点一下按钮就执行的操作上(签到、购买等): 点击后弹出验证, 通过后自动关闭并继续, 用法:
 *   const captcha = useCaptchaDialog('checkin');
 *   await services.checkin.checkin(await captcha.verify());
 *   ... {captcha.dialog} ...
 * 每次操作先读取最新配置; 场景没有开启时不弹窗, verify 返回 undefined
 */
export function useCaptchaDialog(scene: CaptchaScene): {
  /** 读取最新配置后弹出验证并等待令牌; 取消时中断提交且不提示, 需要在 useSubmit 的 submit 中调用 */
  verify: () => Promise<string | undefined>;
  dialog: ReactNode;
} {
  const {captcha} = useSite();
  // 没有填写站点密钥时后端也不验证
  const enabled = captcha[`${scene}_enabled`] && captcha.site_key !== '';
  /** 有值时弹窗打开 */
  const [waiting, setWaiting] = useState<Verification | null>(null);
  const verification = useRef<Verification | null>(null);
  const requestId = useRef(0);

  // 缓存中的开关仅用于预加载, 是否验证在点击时重新读取。
  useEffect(() => {
    if (!enabled) return;
    void loadTurnstile().catch(() => {
      // 弹窗打开时会重新加载并给出提示
    });
  }, [enabled]);

  useLayoutEffect(() => () => {
    // 页面可能被 Activity 隐藏并保留状态, 取消等待时也要关闭弹窗。
    requestId.current++;
    verification.current?.reject(new CancelledError());
    verification.current = null;
    setWaiting(null);
  }, []);

  const verify = useCallback(async (): Promise<string | undefined> => {
    const id = ++requestId.current;
    verification.current?.reject(new CancelledError());
    verification.current = null;
    setWaiting(null);
    let latest;
    try {
      latest = await services.settings.getPublic();
    } catch (err) {
      if (id !== requestId.current) throw new CancelledError();
      throw err;
    }
    if (id !== requestId.current) throw new CancelledError();
    const current = latest.captcha;
    if (!current) throw new ApiError('无法读取人机验证配置, 请稍后再试');
    if (!current[`${scene}_enabled`] || !current.site_key) return undefined;
    return new Promise((resolve, reject) => {
      const next = {siteKey: current.site_key, resolve, reject};
      verification.current = next;
      setWaiting(next);
    });
  }, [scene]);

  const dialog = (
    <CaptchaDialog
      siteKey={waiting?.siteKey ?? ''}
      open={!!waiting}
      onVerified={(token) => {
        if (!waiting || verification.current !== waiting) return;
        verification.current = null;
        waiting.resolve(token);
        setWaiting(null);
      }}
      onCancel={() => {
        if (!waiting || verification.current !== waiting) return;
        verification.current = null;
        waiting.reject(new CancelledError());
        setWaiting(null);
      }}
    />
  );
  return {verify, dialog};
}
