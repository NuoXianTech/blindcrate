'use client';

import {useCallback, useEffect, useState} from 'react';
import {toast} from 'sonner';
import {cn} from 'cn';
import {Table, TableBody, TableCell, TableHead, TableHeader, TableRow} from '@/components/ui/table';
import {useAuth} from '@/components/common/auth';
import {Pagination, TableEmpty, TableSkeleton} from '@/components/common/layout';
import services from '@/lib/services';
import type {AdminInventoryTransaction} from '@/lib/services/admin';
import {toErrorMessage} from '@/lib/services/core';
import {INVENTORY_BIZ_LABELS} from '@/lib/biz';
import {formatDateTime} from '@/lib/utils';
import {Actor, BizTypeSelect, IDFilterForm} from './TransactionFilters';
import {ALL, PAGE_SIZE, type IDFilters} from './constants';

/**
 * 全部用户的背包流水, 可按用户、操作人、物品和类型筛选
 */
export function InventoryTransactions({initialUserID}: { initialUserID?: number }) {
  const isAdmin = !!useAuth().user?.is_admin;
  const [ids, setIDs] = useState<IDFilters>({user_id: initialUserID});
  const [bizType, setBizType] = useState(ALL);
  const [current, setCurrent] = useState(1);
  const [data, setData] = useState<{ total: number; transactions: AdminInventoryTransaction[] } | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    if (!isAdmin) return;
    setLoading(true);
    try {
      setData(
          await services.admin.listInventoryTransactions({
            current,
            size: PAGE_SIZE,
            ...ids,
            biz_type: bizType === ALL ? undefined : bizType,
          }),
      );
    } catch (err) {
      toast.error(toErrorMessage(err));
    } finally {
      setLoading(false);
    }
  }, [current, ids, bizType, isAdmin]);

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
            {key: 'item_id', label: '物品 ID'},
          ]}
          values={ids}
          onSubmit={(next) => {
            setCurrent(1);
            setIDs(next);
          }}
        />
        <BizTypeSelect
          value={bizType}
          labels={INVENTORY_BIZ_LABELS}
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
              <TableHead>物品</TableHead>
              <TableHead className="text-right">变动</TableHead>
              <TableHead className="text-right">变动后数量</TableHead>
              <TableHead>类型</TableHead>
              <TableHead>操作人</TableHead>
              <TableHead>备注</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading && !data ? (
              <TableSkeleton cols={8} />
            ) : data?.transactions.length ? (
              data.transactions.map((t) => (
                <TableRow key={t.id}>
                  <TableCell className="text-muted-foreground">{formatDateTime(t.created_at)}</TableCell>
                  <TableCell>
                    <Actor id={t.user_id} username={t.username} />
                  </TableCell>
                  <TableCell>
                    <span className="flex flex-col">
                      <span>{t.item_name}</span>
                      <span className="text-xs text-muted-foreground">#{t.item_id}</span>
                    </span>
                  </TableCell>
                  <TableCell
                    className={cn('text-right font-mono tabular-nums', t.quantity > 0 ? 'text-quality-green' : 'text-destructive')}
                  >
                    {t.quantity > 0 ? `+${t.quantity}` : t.quantity}
                  </TableCell>
                  <TableCell className="text-right font-mono text-muted-foreground tabular-nums">{t.quantity_after}</TableCell>
                  <TableCell>{INVENTORY_BIZ_LABELS[t.biz_type] ?? t.biz_type}</TableCell>
                  <TableCell>
                    <Actor id={t.operator_id} username={t.operator_username} system />
                  </TableCell>
                  <TableCell className="max-w-64 truncate text-muted-foreground" title={t.remark || undefined}>
                    {t.remark || '-'}
                  </TableCell>
                </TableRow>
              ))
            ) : (
              <TableEmpty colSpan={8}>没有找到记录</TableEmpty>
            )}
          </TableBody>
        </Table>
      </div>

      <Pagination current={current} size={PAGE_SIZE} total={data?.total ?? 0} onChange={setCurrent} />
    </div>
  );
}
