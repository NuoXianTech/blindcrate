'use client';

import {useEffect, useState} from 'react';
import {toast} from 'sonner';
import services from '@/lib/services';
import type {AdminItem, Currency} from '@/lib/services/admin';
import {toErrorMessage} from '@/lib/services/core';

/** 物品列表接口的分页上限 */
const ITEM_PAGE_SIZE = 100;
/** 最多拉取的页数, 避免物品过多时请求不停 */
const MAX_ITEM_PAGES = 20;

/** 分页拉取全部物品, 含已下架的 */
async function listAllItems() {
  const items: AdminItem[] = [];
  for (let current = 1; current <= MAX_ITEM_PAGES; current++) {
    const data = await services.admin.listItems({current, size: ITEM_PAGE_SIZE});
    items.push(...data.items);
    if (items.length >= data.total || data.items.length < ITEM_PAGE_SIZE) break;
  }
  return items;
}

/**
 * 后台编辑器用到的物品和货币目录, 挂载时拉取一次
 * 物品含已下架的, 用于显示已引用物品的名称; 可选项由调用方按 is_enabled 过滤
 */
export function useAdminCatalog() {
  const [items, setItems] = useState<AdminItem[]>([]);
  const [currencies, setCurrencies] = useState<Currency[]>([]);
  const [loaded, setLoaded] = useState(false);

  useEffect(() => {
    Promise.all([listAllItems(), services.admin.listCurrencies()])
        .then(([items, data]) => {
          setItems(items);
          setCurrencies(data.currencies);
          setLoaded(true);
        })
        .catch((err) => toast.error(toErrorMessage(err)));
  }, []);

  return {items, currencies, loaded};
}
