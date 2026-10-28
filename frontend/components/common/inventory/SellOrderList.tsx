'use client';

import {useCallback, useEffect, useState} from 'react';
import {toast} from 'sonner';
import {Table, TableBody, TableCell, TableHead, TableHeader, TableRow} from '@/components/ui/table';
import {useAuth} from '@/components/common/auth';
import {Pagination, TableEmpty, TableSkeleton} from '@/components/common/layout';
import {CurrencyIcon} from '@/components/common/wallet';
import services from '@/lib/services';
import {toErrorMessage} from '@/lib/services/core';
import type {ListSellOrdersData} from '@/lib/services/inventory';
import type {BalanceItem} from '@/lib/services/wallet';
import {formatAmount, formatDateTime} from '@/lib/utils';
import {SELL_ORDER_PAGE_SIZE} from './constants';
import {formatFeeRate} from './sell';

/**
 * 出售记录: 每次出售一行, 显示卖出的物品、手续费和实得
 */
export function SellOrderList({currencies}: { currencies: BalanceItem[] }) {
  const {user} = useAuth();
  const [current, setCurrent] = useState(1);
  const [data, setData] = useState<ListSellOrdersData | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    if (!user) return;
    setLoading(true);
    try {
      setData(await services.inventory.listSellOrders({current, size: SELL_ORDER_PAGE_SIZE}));
    } catch (err) {
      toast.error(toErrorMessage(err));
    } finally {
      setLoading(false);
    }
  }, [current, user]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 翻页时重新拉取
    void load();
  }, [load]);

  const decimalsOf = (currency: string) => currencies.find((c) => c.currency === currency)?.decimals ?? 0;

  return (
    <div className="flex flex-col gap-4">
      <div className="rounded-xl border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>时间</TableHead>
              <TableHead>物品</TableHead>
              <TableHead className="text-right">手续费</TableHead>
              <TableHead className="text-right">实得</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading && !data ? (
              <TableSkeleton cols={4} />
            ) : data?.orders.length ? (
              data.orders.map((order) => (
                <TableRow key={order.id} className="align-top">
                  <TableCell className="text-muted-foreground">{formatDateTime(order.created_at)}</TableCell>
                  <TableCell>
                    <ul className="flex flex-col gap-1">
                      {order.items.map((i) => (
                        <li key={i.id} className="flex items-center gap-1.5">
                          <span className="max-w-48 truncate" title={i.item_name}>
                            {i.item_name}
                          </span>
                          <span className="font-mono text-muted-foreground tabular-nums">×{i.quantity}</span>
                          <span className="flex items-center gap-1 text-xs text-muted-foreground">
                            <CurrencyIcon currency={i.sell_currency} className="size-3.5" />
                            <span className="font-mono tabular-nums">{formatAmount(i.amount, decimalsOf(i.sell_currency))}</span>
                          </span>
                        </li>
                      ))}
                    </ul>
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex flex-col items-end gap-1">
                      {order.incomes.map((i) => (
                        <span key={i.currency} className="flex items-center gap-1 text-muted-foreground">
                          <CurrencyIcon currency={i.currency} className="size-4" />
                          <span className="font-mono tabular-nums">{formatAmount(i.fee, decimalsOf(i.currency))}</span>
                        </span>
                      ))}
                      <span className="text-xs text-muted-foreground">费率 {formatFeeRate(order.fee_rate)}</span>
                    </div>
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex flex-col items-end gap-1">
                      {order.incomes.map((i) => (
                        <span key={i.currency} className="flex items-center gap-1 font-medium text-quality-green">
                          <CurrencyIcon currency={i.currency} className="size-4" />
                          <span className="font-mono tabular-nums">
                            {formatAmount(i.amount - i.fee, decimalsOf(i.currency), {signed: true})}
                          </span>
                        </span>
                      ))}
                    </div>
                  </TableCell>
                </TableRow>
              ))
            ) : (
              <TableEmpty colSpan={4}>还没有出售过物品</TableEmpty>
            )}
          </TableBody>
        </Table>
      </div>

      <Pagination current={current} size={SELL_ORDER_PAGE_SIZE} total={data?.total ?? 0} onChange={setCurrent} />
    </div>
  );
}
