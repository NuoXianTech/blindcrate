'use client';

import {useCallback, useEffect, useState} from 'react';
import {X} from 'lucide-react';
import {toast} from 'sonner';
import {cn} from 'cn';
import {Badge} from '@/components/ui/badge';
import {Table, TableBody, TableCell, TableHead, TableHeader, TableRow} from '@/components/ui/table';
import {useAuth} from '@/components/common/auth';
import {Pagination, TableEmpty, TableSkeleton} from '@/components/common/layout';
import services from '@/lib/services';
import {toErrorMessage} from '@/lib/services/core';
import type {InventoryTransaction} from '@/lib/services/inventory';
import {useLabels} from '@/hooks/use-labels';
import {formatDateTime} from '@/lib/utils';
import {RECORD_PAGE_SIZE, type ItemFilter} from './constants';

/**
 * 背包记录, 可只看某个物品
 */
export function RecordList({itemFilter, onClearFilter}: { itemFilter: ItemFilter | null; onClearFilter: () => void }) {
  const bizLabels = useLabels('inventoryBiz');
  const {user} = useAuth();
  const [current, setCurrent] = useState(1);
  const [data, setData] = useState<{ total: number; transactions: InventoryTransaction[] } | null>(null);
  const [loading, setLoading] = useState(true);
  const itemID = itemFilter?.id;

  // 切换物品筛选时回到第一页
  const [prevItemID, setPrevItemID] = useState(itemID);
  if (prevItemID !== itemID) {
    setPrevItemID(itemID);
    setCurrent(1);
  }

  const load = useCallback(async () => {
    if (!user) return;
    setLoading(true);
    try {
      setData(await services.inventory.listTransactions({current, size: RECORD_PAGE_SIZE, item_id: itemID}));
    } catch (err) {
      toast.error(toErrorMessage(err));
    } finally {
      setLoading(false);
    }
  }, [current, itemID, user]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 筛选条件变化时重新拉取
    void load();
  }, [load]);

  return (
    <div className="flex flex-col gap-4">
      {itemFilter && (
        <div className="flex items-center gap-2 text-sm">
          <span className="text-muted-foreground">只看</span>
          <Badge variant="secondary" className="gap-1 pr-1">
            {itemFilter.name}
            <button type="button" aria-label="取消筛选" className="rounded-full p-0.5 hover:bg-background/50" onClick={onClearFilter}>
              <X className="size-3" />
            </button>
          </Badge>
        </div>
      )}

      <div className="rounded-xl border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>时间</TableHead>
              <TableHead>物品</TableHead>
              <TableHead className="text-right">变动</TableHead>
              <TableHead className="text-right">变动后数量</TableHead>
              <TableHead>类型</TableHead>
              <TableHead>备注</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading && !data ? (
              <TableSkeleton cols={6} />
            ) : data?.transactions.length ? (
              data.transactions.map((t) => (
                <TableRow key={t.id}>
                  <TableCell className="text-muted-foreground">{formatDateTime(t.created_at)}</TableCell>
                  <TableCell>{t.item_name}</TableCell>
                  <TableCell
                    className={cn('text-right font-mono tabular-nums', t.quantity > 0 ? 'text-quality-green' : 'text-destructive')}
                  >
                    {t.quantity > 0 ? `+${t.quantity}` : t.quantity}
                  </TableCell>
                  <TableCell className="text-right font-mono text-muted-foreground tabular-nums">{t.quantity_after}</TableCell>
                  <TableCell>{bizLabels[t.biz_type] ?? t.biz_type}</TableCell>
                  <TableCell className="max-w-64 truncate text-muted-foreground" title={t.remark || undefined}>
                    {t.remark || '-'}
                  </TableCell>
                </TableRow>
              ))
            ) : (
              <TableEmpty colSpan={6}>还没有记录</TableEmpty>
            )}
          </TableBody>
        </Table>
      </div>

      <Pagination current={current} size={RECORD_PAGE_SIZE} total={data?.total ?? 0} onChange={setCurrent} />
    </div>
  );
}
