'use client';

import {useCallback, useEffect, useRef, useState} from 'react';
import {ScrollText, Store} from 'lucide-react';
import {toast} from 'sonner';
import {Skeleton} from '@/components/ui/skeleton';
import {Tabs, TabsContent, TabsList, TabsTrigger} from '@/components/ui/tabs';
import {useAuth} from '@/components/common/auth';
import {RedeemButton} from '@/components/common/redeem';
import {useWsConnected, useWsEvent} from '@/hooks/use-ws';
import services from '@/lib/services';
import {toErrorMessage} from '@/lib/services/core';
import type {ShopItem} from '@/lib/services/shop';
import {WALLET_MESSAGES} from '@/lib/services/wallet';
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
  return user ? <ShopContent key={user.id} /> : null;
}

function ShopContent() {
  const [items, setItems] = useState<ShopItem[] | null>(null);
  const [tab, setTab] = useState('');
  const [buying, setBuying] = useState<ShopItem | null>(null);
  const [reading, setReading] = useState(false);
  const [readError, setReadError] = useState('');
  const readID = useRef(0);
  const connected = useWsConnected();
  const invalidateRead = useCallback(() => {
    readID.current++;
  }, []);

  const load = useCallback(async () => {
    const id = ++readID.current;
    setReading(true);
    setReadError('');
    try {
      const next = (await services.shop.listItems()).items;
      if (id !== readID.current) return null;
      setItems(next);
      setBuying((selected) => selected ? next.find((it) => it.code === selected.code) ?? {...selected, max_quantity: 0} : null);
      return next;
    } catch (err) {
      if (id === readID.current) {
        setReadError(toErrorMessage(err));
        toast.error(toErrorMessage(err));
      }
      return null;
    } finally {
      if (id === readID.current) setReading(false);
    }
  }, []);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 进入页面或重新连接时读取当前可购买份数
    void load();
    const onFocus = () => void load();
    window.addEventListener('focus', onFocus);
    return () => {
      invalidateRead();
      window.removeEventListener('focus', onFocus);
    };
  }, [load, connected, invalidateRead]);

  useWsEvent(WALLET_MESSAGES.balanceChanged, () => void load());
  useWsEvent('inventory:items_changed', () => void load());

  const groups = groupByTab(items ?? []);
  const single = groups.length <= 1;
  const tabs = single ? [GOODS_TAB] : groups.map(([name]) => `goods:${name}`);
  // 商品刷新后选中的分组可能已经没有商品, 此时回到第一个分组
  const active = tab === ORDERS_TAB || tabs.includes(tab) ? tab : tabs[0];

  const grid = (list: ShopItem[]) =>
    list.length ? (
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
        {list.map((item) => (
          <ShopItemCard key={item.code} item={item} onBuy={() => {
            setBuying(item);
            void load();
          }} />
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
      <div className="flex items-center justify-between gap-2">
        <h1 className="text-2xl font-bold">商城</h1>
        {/* 兑换到的货币和物品会影响商品的购买条件, 兑换后刷新商品 */}
        <RedeemButton onRedeemed={() => void load()} />
      </div>
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

      <BuyDialog item={buying} reading={reading} readError={readError} onClose={() => setBuying(null)} onRefresh={load} />
    </div>
  );
}
