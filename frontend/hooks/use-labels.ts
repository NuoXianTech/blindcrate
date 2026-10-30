'use client';

import {useEffect, useState} from 'react';
import services from '@/lib/services';
import type {CodeLabel} from '@/lib/services/core';

/** 代码 -> 名称, 按后端登记的顺序 */
export type Labels = Record<string, string>;

/** 名称的来源, 都由后端各模块登记 */
const loaders = {
  /** 余额流水的业务类型 */
  walletBiz: () => services.wallet.listBizTypes().then((d) => d.biz_types),
  /** 背包流水的业务类型 */
  inventoryBiz: () => services.inventory.listBizTypes().then((d) => d.biz_types),
  /** 操作日志的操作类型, 只有管理员可以读取 */
  auditActions: () => services.admin.listAuditActions().then((d) => d.actions),
} satisfies Record<string, () => Promise<CodeLabel[]>>;

export type LabelSource = keyof typeof loaders;

/** 每种名称在页面会话中只拉取一次, 失败时下次使用再重试 */
const cache = new Map<LabelSource, Promise<Labels>>();

function load(source: LabelSource) {
  let pending = cache.get(source);
  if (!pending) {
    pending = loaders[source]().then((list) => Object.fromEntries(list.map((t) => [t.code, t.label])));
    pending.catch(() => cache.delete(source));
    cache.set(source, pending);
  }
  return pending;
}

/** 后端登记的代码名称, 加载完成前为空, 调用方应在没有名称时显示代码本身 */
export function useLabels(source: LabelSource): Labels {
  const [labels, setLabels] = useState<Labels>({});

  useEffect(() => {
    let alive = true;
    load(source)
        .then((next) => {
          if (alive) setLabels(next);
        })
        .catch(() => {
          // 名称只用于展示, 失败时显示代码
        });
    return () => {
      alive = false;
    };
  }, [source]);

  return labels;
}
