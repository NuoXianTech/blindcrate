'use client';

import {useCallback, useEffect, useState} from 'react';
import {toast} from 'sonner';
import {cn} from 'cn';
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select';
import {Table, TableBody, TableCell, TableHead, TableHeader, TableRow} from '@/components/ui/table';
import {useAuth} from '@/components/common/auth';
import {Pagination, TableEmpty, TableSkeleton} from '@/components/common/layout';
import {CurrencyIcon} from '@/components/common/wallet';
import services from '@/lib/services';
import type {AdminWalletTransaction, Currency} from '@/lib/services/admin';
import {toErrorMessage} from '@/lib/services/core';
import {WALLET_BIZ_LABELS} from '@/lib/biz';
import {formatAmount, formatDateTime} from '@/lib/utils';
import {Actor, BizTypeSelect, IDFilterForm} from './TransactionFilters';
import {ALL, PAGE_SIZE, type IDFilters} from './constants';

/**
 * 全部用户的余额流水, 可按用户、操作人、货币和类型筛选
 */
export function WalletTransactions({initialUserID}: { initialUserID?: number }) {
  const isAdmin = !!useAuth().user?.is_admin;
  const [ids, setIDs] = useState<IDFilters>({user_id: initialUserID});
  const [currency, setCurrency] = useState(ALL);
  const [bizType, setBizType] = useState(ALL);
  const [current, setCurrent] = useState(1);
  const [data, setData] = useState<{ total: number; transactions: AdminWalletTransaction[] } | null>(null);
  const [loading, setLoading] = useState(true);
  const [currencies, setCurrencies] = useState<Currency[]>([]);

  useEffect(() => {
    if (!isAdmin) return;
    services.admin.listCurrencies()
        .then((data) => setCurrencies(data.currencies))
        .catch(() => {
        // 只影响货币名称和小数位, 失败时显示货币代码
        });
  }, [isAdmin]);

  const load = useCallback(async () => {
    // 确认是管理员后再拉取, 否则 AdminGuard 会跳走
    if (!isAdmin) return;
    setLoading(true);
    try {
      setData(
          await services.admin.listWalletTransactions({
            current,
            size: PAGE_SIZE,
            ...ids,
            currency: currency === ALL ? undefined : currency,
            biz_type: bizType === ALL ? undefined : bizType,
          }),
      );
    } catch (err) {
      toast.error(toErrorMessage(err));
    } finally {
      setLoading(false);
    }
  }, [current, ids, currency, bizType, isAdmin]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 筛选条件变化时重新拉取
    void load();
  }, [load]);

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-2 sm:flex-row">
        <IDFilterForm
          fields={[
            {key: 'user_id', label: '用户 ID'},
            {key: 'operator_id', label: '操作人 ID'},
          ]}
          values={ids}
          onSubmit={(next) => {
            setCurrent(1);
            setIDs(next);
          }}
        />
        <Select
          value={currency}
          onValueChange={(value) => {
            setCurrent(1);
            setCurrency(value);
          }}
        >
          <SelectTrigger className="sm:w-32">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL}>全部货币</SelectItem>
            {currencies.map((c) => (
              <SelectItem key={c.code} value={c.code}>
                <CurrencyIcon currency={c.code} />
                {c.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <BizTypeSelect
          value={bizType}
          labels={WALLET_BIZ_LABELS}
          onChange={(value) => {
            setCurrent(1);
            setBizType(value);
          }}
        />
      </div>

      <div className="rounded-xl border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>时间</TableHead>
              <TableHead>用户</TableHead>
              <TableHead>货币</TableHead>
              <TableHead className="text-right">变动</TableHead>
              <TableHead className="text-right">变动后余额</TableHead>
              <TableHead>类型</TableHead>
              <TableHead>操作人</TableHead>
              <TableHead>备注</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading && !data ? (
              <TableSkeleton cols={8} />
            ) : data?.transactions.length ? (
              data.transactions.map((t) => {
                const c = currencies.find((c) => c.code === t.currency);
                const decimals = c?.decimals ?? 0;
                return (
                  <TableRow key={t.id}>
                    <TableCell className="text-muted-foreground">{formatDateTime(t.created_at)}</TableCell>
                    <TableCell>
                      <Actor id={t.user_id} username={t.username} />
                    </TableCell>
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
                    <TableCell>{WALLET_BIZ_LABELS[t.biz_type] ?? t.biz_type}</TableCell>
                    <TableCell>
                      <Actor id={t.operator_id} username={t.operator_username} system />
                    </TableCell>
                    <TableCell className="max-w-64 truncate text-muted-foreground" title={t.remark || undefined}>
                      {t.remark || '-'}
                    </TableCell>
                  </TableRow>
                );
              })
            ) : (
              <TableEmpty colSpan={8}>没有找到流水</TableEmpty>
            )}
          </TableBody>
        </Table>
      </div>

      <Pagination current={current} size={PAGE_SIZE} total={data?.total ?? 0} onChange={setCurrent} />
    </div>
  );
}
