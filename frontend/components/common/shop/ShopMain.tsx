'use client';

import {useCallback, useEffect, useState} from 'react';
import {ScrollText, Store} from 'lucide-react';
import {toast} from 'sonner';
import {Skeleton} from '@/components/ui/skeleton';
import {Tabs, TabsContent, TabsList, TabsTrigger} from '@/components/ui/tabs';
import {useAuth} from '@/components/common/auth';
import services from '@/lib/services';
import {toErrorMessage} from '@/lib/services/core';
import type {ShopItem} from '@/lib/services/shop';
import {BuyDialog} from './BuyDialog';
import {OrderList} from './OrderList';
import {ShopItemCard} from './ShopItemCard';
import {DEFAULT_TAB} from './constants';

/** 购买记录的页签值, 商品页签的值带 "goods:" 前缀, 不会与之重复 */
const ORDERS_TAB = 'orders';

/** 只有一个分组时全部商品所在的页签 */
const GOODS_TAB = 'goods:';

/** 按页签分组, 保持后端的排序; 没有页签的商品归入 DEFAULT_TAB */
function groupByTab(items: ShopItem[]): [string, ShopItem[]][] {
  const groups = new Map<string, ShopItem[]>();
  for (const item of items) {
    const tab = item.tab || DEFAULT_TAB;
    groups.set(tab, [...(groups.get(tab) ?? []), item]);
  }
  return [...groups];
}

/**
 * 商城: 按页签分组的商品 + 购买记录; 只有一个分组时不再细分页签
 */
export function ShopMain() {
  const {user} = useAuth();
  const [items, setItems] = useState<ShopItem[] | null>(null);
  const [tab, setTab] = useState('');
  const [buying, setBuying] = useState<ShopItem | null>(null);

  const load = useCallback(async () => {
    if (!user) return;
    try {
      setItems((await services.shop.listItems()).items);
    } catch (err) {
      toast.error(toErrorMessage(err));
    }
  }, [user]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 进入页面时拉取商品
    void load();
  }, [load]);

  const groups = groupByTab(items ?? []);
  const single = groups.length <= 1;
  const tabs = single ? [GOODS_TAB] : groups.map(([name]) => `goods:${name}`);
  // 商品刷新后选中的分组可能已经没有商品, 此时回到第一个分组
  const active = tab === ORDERS_TAB || tabs.includes(tab) ? tab : tabs[0];

  const grid = (list: ShopItem[]) =>
    list.length ? (
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
        {list.map((item) => (
          <ShopItemCard key={item.code} item={item} onBuy={() => setBuying(item)} />
        ))}
      </div>
    ) : (
      <div className="flex h-48 flex-col items-center justify-center gap-2 rounded-xl border bg-card text-muted-foreground">
        <Store className="size-8" />
        <span className="text-sm">商城里还没有商品</span>
      </div>
    );

  return (
    <div className="flex flex-col gap-4">
      <h1 className="text-2xl font-bold">商城</h1>
      <Tabs value={active} onValueChange={setTab}>
        <TabsList className="h-auto flex-wrap">
          {single ? (
            <TabsTrigger value={GOODS_TAB}>
              <Store />
              商品
            </TabsTrigger>
          ) : (
            groups.map(([name]) => (
              <TabsTrigger key={name} value={`goods:${name}`}>
                {name}
              </TabsTrigger>
            ))
          )}
          <TabsTrigger value={ORDERS_TAB}>
            <ScrollText />
            购买记录
          </TabsTrigger>
        </TabsList>

        {items === null ? (
          <TabsContent value={GOODS_TAB} className="mt-2">
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
              {Array.from({length: 4}, (_, i) => (
                <Skeleton key={i} className="h-72 rounded-xl" />
              ))}
            </div>
          </TabsContent>
        ) : single ? (
          <TabsContent value={GOODS_TAB} className="mt-2">
            {grid(items)}
          </TabsContent>
        ) : (
          groups.map(([name, list]) => (
            <TabsContent key={name} value={`goods:${name}`} className="mt-2">
              {grid(list)}
            </TabsContent>
          ))
        )}
        <TabsContent value={ORDERS_TAB} className="mt-2">
          <OrderList />
        </TabsContent>
      </Tabs>

      <BuyDialog item={buying} onClose={() => setBuying(null)} onRefresh={() => void load()} />
    </div>
  );
}
