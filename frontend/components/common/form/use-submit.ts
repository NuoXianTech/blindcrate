"use client";

import { useCallback, useRef, useState } from "react";
import { toast } from "sonner";
import { toErrorMessage } from "@/lib/services/core";

/** 校验未通过, 提示文案直接展示给用户 */
class FormError extends Error {}

/** 校验结果为错误文案时中断提交, 用法: check(checkUsername(name)) */
export function check(message: string | null | undefined): void {
  if (message) throw new FormError(message);
}

/**
 * 表单提交: 管理提交中状态、防止重复提交, 校验失败和请求失败都以右下角提示框展示
 * 用法: <form onSubmit={submit(async () => { check(...); await api(); })}>
 */
export function useSubmit() {
  const [pending, setPending] = useState(false);
  // 同一次渲染内的连续触发也要拦住, state 来不及更新
  const running = useRef(false);

  const submit = useCallback(
    (action: () => Promise<void>) => async (e?: React.FormEvent) => {
      e?.preventDefault();
      if (running.current) return;
      running.current = true;
      setPending(true);
      try {
        await action();
      } catch (err) {
        toast.error(err instanceof FormError ? err.message : toErrorMessage(err));
      } finally {
        running.current = false;
        setPending(false);
      }
    },
    [],
  );

  return { pending, submit };
}
