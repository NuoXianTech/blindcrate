'use client';

import {useCallback, useEffect, useState} from 'react';
import {Backpack} from 'lucide-react';
import {toast} from 'sonner';
import {Button} from '@/components/ui/button';
import {Skeleton} from '@/components/ui/skeleton';
import {useAuth} from '@/components/common/auth';
import {Pagination} from '@/components/common/layout';
import services from '@/lib/services';
import {toErrorMessage} from '@/lib/services/core';
import type {UserItem} from '@/lib/services/inventory';
import type {BalanceItem} from '@/lib/services/wallet';
import {QUALITIES} from '@/lib/quality';
import {ItemCard} from './ItemCard';
import {ItemDetailDialog} from './ItemDetailDialog';
import {ITEM_PAGE_SIZE} from './constants';

/**
 * 物品网格, 可按品质筛选, 点击物品查看详情
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

  return (
    <div className="flex flex-col gap-4">
      <div className="hide-scrollbar flex gap-1 overflow-x-auto">
        <Button variant={quality === undefined ? 'secondary' : 'ghost'} size="sm" onClick={() => changeQuality(undefined)}>
          全部
        </Button>
        {QUALITIES.map((q) => (
          <Button key={q.value} variant={quality === q.value ? 'secondary' : 'ghost'} size="sm" onClick={() => changeQuality(q.value)}>
            <span className={q.text}>{q.label}</span>
          </Button>
        ))}
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
            <ItemCard key={item.item_id} item={item} onClick={() => setSelected(item)} />
          ))}
        </div>
      ) : (
        <div className="flex h-48 flex-col items-center justify-center gap-2 rounded-xl border bg-card text-muted-foreground">
          <Backpack className="size-8" />
          <span className="text-sm">{quality === undefined ? '背包里还没有物品' : '没有这个品质的物品'}</span>
        </div>
      )}

      <Pagination current={current} size={ITEM_PAGE_SIZE} total={data?.total ?? 0} onChange={setCurrent} />

      <ItemDetailDialog
        item={selected}
        currencies={currencies}
        onClose={() => setSelected(null)}
        onViewRecords={(item) => {
          setSelected(null);
          onViewRecords(item);
        }}
      />
    </div>
  );
}
