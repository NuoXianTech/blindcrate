'use client';

import {useCallback, useEffect, useState} from 'react';
import {toast} from 'sonner';
import {cn} from 'cn';
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select';
import {Table, TableBody, TableCell, TableHead, TableHeader, TableRow} from '@/components/ui/table';
import {useAuth} from '@/components/common/auth';
import {Pagination, TableEmpty, TableSkeleton} from '@/components/common/layout';
import services from '@/lib/services';
import {toErrorMessage} from '@/lib/services/core';
import type {BalanceItem, WalletTransaction} from '@/lib/services/wallet';
import {useLabels} from '@/hooks/use-labels';
import {formatAmount, formatDateTime} from '@/lib/utils';
import {CurrencyIcon} from './CurrencyIcon';
import {ALL, PAGE_SIZE} from './constants';

/**
 * 当前用户的余额流水, 可按货币筛选
 */
export function TransactionList({balances}: { balances: BalanceItem[] }) {
  const bizLabels = useLabels('walletBiz');
  const {user} = useAuth();
  const [currency, setCurrency] = useState<string>(ALL);
  const [current, setCurrent] = useState(1);
  const [data, setData] = useState<{ total: number; transactions: WalletTransaction[] } | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    if (!user) return;
    setLoading(true);
    try {
      setData(
          await services.wallet.listTransactions({
            current,
            size: PAGE_SIZE,
            currency: currency === ALL ? undefined : currency,
          }),
      );
    } catch (err) {
      toast.error(toErrorMessage(err));
    } finally {
      setLoading(false);
    }
  }, [current, currency, user]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 筛选条件变化时重新拉取
    void load();
  }, [load]);

  // 流水里可能有已停用且余额为 0 的货币, 余额列表里没有时直接显示代码
  const currencyOf = (code: string) => balances.find((b) => b.currency === code);

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between gap-2">
        <h2 className="text-lg font-semibold">余额流水</h2>
        <Select
          value={currency}
          onValueChange={(value) => {
            setCurrent(1);
            setCurrency(value);
          }}
        >
          <SelectTrigger className="w-32">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL}>全部货币</SelectItem>
            {balances.map((b) => (
              <SelectItem key={b.currency} value={b.currency}>
                {b.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="rounded-xl border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>时间</TableHead>
              <TableHead>货币</TableHead>
              <TableHead className="text-right">变动</TableHead>
              <TableHead className="text-right">变动后余额</TableHead>
              <TableHead>类型</TableHead>
              <TableHead>备注</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading && !data ? (
              <TableSkeleton cols={6} />
            ) : data?.transactions.length ? (
              data.transactions.map((t) => {
                const c = currencyOf(t.currency);
                const decimals = c?.decimals ?? 0;
                return (
                  <TableRow key={t.id}>
                    <TableCell className="text-muted-foreground">{formatDateTime(t.created_at)}</TableCell>
                    <TableCell>
                      <span className="flex items-center gap-1.5">
                        <CurrencyIcon currency={t.currency} className="size-4" />
                        {c?.name ?? t.currency}
                      </span>
                    </TableCell>
                    <TableCell className="text-right font-mono tabular-nums">
                      {t.amount !== 0 && (
                        <div className={cn(t.amount > 0 ? 'text-quality-green' : 'text-destructive')}>
                          {formatAmount(t.amount, decimals, {signed: true})}
                        </div>
                      )}
                      {t.frozen_amount !== 0 && (
                        <div className="text-xs text-muted-foreground">冻结 {formatAmount(t.frozen_amount, decimals, {signed: true})}</div>
                      )}
                    </TableCell>
                    <TableCell className="text-right font-mono text-muted-foreground tabular-nums">
                      {formatAmount(t.balance_after, decimals)}
                    </TableCell>
                    <TableCell>{bizLabels[t.biz_type] ?? t.biz_type}</TableCell>
                    <TableCell className="max-w-64 truncate text-muted-foreground" title={t.remark || undefined}>
                      {t.remark || '-'}
                    </TableCell>
                  </TableRow>
                );
              })
            ) : (
              <TableEmpty colSpan={6}>还没有流水</TableEmpty>
            )}
          </TableBody>
        </Table>
      </div>

      <Pagination current={current} size={PAGE_SIZE} total={data?.total ?? 0} onChange={setCurrent} />
    </div>
  );
}
