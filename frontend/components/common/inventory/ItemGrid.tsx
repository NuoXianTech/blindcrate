'use client';

import {useCallback, useEffect, useState} from 'react';
import {Backpack, HandCoins, ListChecks} from 'lucide-react';
import {toast} from 'sonner';
import {Button} from '@/components/ui/button';
import {Skeleton} from '@/components/ui/skeleton';
import {useAuth} from '@/components/common/auth';
import {Pagination} from '@/components/common/layout';
import {CurrencyIcon} from '@/components/common/wallet';
import services from '@/lib/services';
import {toErrorMessage} from '@/lib/services/core';
import type {UserItem} from '@/lib/services/inventory';
import type {BalanceItem} from '@/lib/services/wallet';
import {QUALITIES} from '@/lib/quality';
import {formatAmount} from '@/lib/utils';
import {ItemCard} from './ItemCard';
import {ItemDetailDialog} from './ItemDetailDialog';
import {SellDialog} from './SellDialog';
import {ITEM_PAGE_SIZE, MAX_SELL_LINES, MAX_SELL_QUANTITY} from './constants';
import {sellTotals, type SellLine} from './sell';

/**
 * 物品网格, 可按品质筛选, 点击物品查看详情; 多选模式下可以批量出售
 */
export function ItemGrid({currencies, onViewRecords}: {
  currencies: BalanceItem[];
  onViewRecords: (item: UserItem) => void;
}) {
  const {user} = useAuth();
  const [quality, setQuality] = useState<number | undefined>();
  const [current, setCurrent] = useState(1);
  const [data, setData] = useState<{ total: number; items: UserItem[] } | null>(null);
  const [loading, setLoading] = useState(true);
  const [selected, setSelected] = useState<UserItem | null>(null);
  // 多选模式下选中的物品, 翻页和切换品质时保留
  const [picking, setPicking] = useState(false);
  const [picked, setPicked] = useState<Map<number, UserItem>>(new Map());
  const [selling, setSelling] = useState<SellLine[] | null>(null);

  const load = useCallback(async () => {
    if (!user) return;
    setLoading(true);
    try {
      setData(await services.inventory.listItems({current, size: ITEM_PAGE_SIZE, quality}));
    } catch (err) {
      toast.error(toErrorMessage(err));
    } finally {
      setLoading(false);
    }
  }, [current, quality, user]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 筛选条件变化时重新拉取
    void load();
  }, [load]);

  const changeQuality = (value: number | undefined) => {
    setCurrent(1);
    setQuality(value);
  };

  const togglePicking = () => {
    setPicking(!picking);
    setPicked(new Map());
  };

  const togglePicked = (item: UserItem) => {
    const next = new Map(picked);
    if (next.has(item.item_id)) {
      next.delete(item.item_id);
    } else if (next.size >= MAX_SELL_LINES) {
      toast.error(`一次最多出售 ${MAX_SELL_LINES} 种物品`);
      return;
    } else {
      next.set(item.item_id, item);
    }
    setPicked(next);
  };

  // 本页可出售的物品全部选中, 已全部选中时取消
  const pageSellable = data?.items.filter((item) => item.sell_price > 0) ?? [];
  const pageAllPicked = pageSellable.length > 0 && pageSellable.every((item) => picked.has(item.item_id));
  const togglePage = () => {
    const next = new Map(picked);
    for (const item of pageSellable) {
      if (pageAllPicked) next.delete(item.item_id);
      else if (next.size < MAX_SELL_LINES) next.set(item.item_id, item);
    }
    setPicked(next);
  };

  // 批量出售默认卖出全部数量, 可在确认框中调整
  const pickedLines: SellLine[] = [...picked.values()].map((item) => ({item, quantity: Math.min(item.quantity, MAX_SELL_QUANTITY)}));
  const pickedTotals = sellTotals(pickedLines, 0);

  const handleSold = () => {
    setPicking(false);
    setPicked(new Map());
    void load();
  };

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center gap-2">
        <div className="hide-scrollbar flex min-w-0 flex-1 gap-1 overflow-x-auto">
          <Button variant={quality === undefined ? 'secondary' : 'ghost'} size="sm" onClick={() => changeQuality(undefined)}>
            全部
          </Button>
          {QUALITIES.map((q) => (
            <Button key={q.value} variant={quality === q.value ? 'secondary' : 'ghost'} size="sm" onClick={() => changeQuality(q.value)}>
              <span className={q.text}>{q.label}</span>
            </Button>
          ))}
        </div>
        <Button variant={picking ? 'secondary' : 'outline'} size="sm" className="shrink-0" onClick={togglePicking}>
          <ListChecks />
          {picking ? '退出多选' : '批量出售'}
        </Button>
      </div>

      {loading && !data ? (
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4 lg:grid-cols-6">
          {Array.from({length: 6}, (_, i) => (
            <Skeleton key={i} className="aspect-[3/4] rounded-xl" />
          ))}
        </div>
      ) : data?.items.length ? (
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4 lg:grid-cols-6">
          {data.items.map((item) => (
            <ItemCard
              key={item.item_id}
              item={item}
              selected={picking ? picked.has(item.item_id) : undefined}
              disabled={picking && item.sell_price <= 0}
              onClick={() => (picking ? togglePicked(item) : setSelected(item))}
            />
          ))}
        </div>
      ) : (
        <div className="flex h-48 flex-col items-center justify-center gap-2 rounded-xl border bg-card text-muted-foreground">
          <Backpack className="size-8" />
          <span className="text-sm">{quality === undefined ? '背包里还没有物品' : '没有这个品质的物品'}</span>
        </div>
      )}

      <Pagination current={current} size={ITEM_PAGE_SIZE} total={data?.total ?? 0} onChange={setCurrent} />

      {picking && (
        <div className="sticky bottom-24 z-10 flex flex-wrap items-center gap-3 rounded-xl border bg-card/95 px-4 py-3 shadow-lg backdrop-blur">
          <span className="text-sm font-medium">已选 {picked.size} 种</span>
          {pickedTotals.map((t) => (
            <span key={t.currency} className="flex items-center gap-1 text-sm text-muted-foreground">
              <CurrencyIcon currency={t.currency} className="size-4" />
              <span className="font-mono tabular-nums">
                {formatAmount(t.amount, currencies.find((c) => c.currency === t.currency)?.decimals ?? 0)}
              </span>
            </span>
          ))}
          <div className="ml-auto flex gap-2">
            <Button variant="ghost" size="sm" onClick={togglePage} disabled={!pageSellable.length}>
              {pageAllPicked ? '取消本页' : '全选本页'}
            </Button>
            <Button size="sm" onClick={() => setSelling(pickedLines)} disabled={!picked.size}>
              <HandCoins />
              出售
            </Button>
          </div>
        </div>
      )}

      <ItemDetailDialog
        item={selected}
        currencies={currencies}
        onClose={() => setSelected(null)}
        onViewRecords={(item) => {
          setSelected(null);
          onViewRecords(item);
        }}
        onSell={(item) => {
          setSelected(null);
          setSelling([{item, quantity: 1}]);
        }}
      />
      <SellDialog lines={selling} currencies={currencies} onClose={() => setSelling(null)} onSold={handleSold} />
    </div>
  );
}
