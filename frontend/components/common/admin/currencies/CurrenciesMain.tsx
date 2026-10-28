'use client';

import {useCallback, useEffect, useState} from 'react';
import {Pencil, Plus, Power} from 'lucide-react';
import {toast} from 'sonner';
import {Badge} from '@/components/ui/badge';
import {Button} from '@/components/ui/button';
import {Table, TableBody, TableCell, TableHead, TableHeader, TableRow} from '@/components/ui/table';
import {useAuth} from '@/components/common/auth';
import {TableEmpty, TableSkeleton} from '@/components/common/layout';
import {useSite} from '@/components/common/site';
import {CurrencyIcon} from '@/components/common/wallet';
import services from '@/lib/services';
import type {Currency} from '@/lib/services/admin';
import {toErrorMessage} from '@/lib/services/core';
import {formatAmount, formatDateTime} from '@/lib/utils';
import {toRequest} from './constants';
import {CurrencyDialog} from './CurrencyDialog';

/**
 * 货币管理: 列出全部货币, 新建、编辑、启用和停用
 */
export function CurrenciesMain() {
  const isAdmin = !!useAuth().user?.is_admin;
  const {economy} = useSite();
  const [currencies, setCurrencies] = useState<Currency[] | null>(null);

  // null: 关闭; "new": 新建; 货币: 编辑
  const [editing, setEditing] = useState<Currency | 'new' | null>(null);

  const load = useCallback(async () => {
    // 确认是管理员后再拉取, 否则 AdminGuard 会跳走
    if (!isAdmin) return;
    try {
      setCurrencies((await services.admin.listCurrencies()).currencies);
    } catch (err) {
      toast.error(toErrorMessage(err));
    }
  }, [isAdmin]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 挂载和保存后重新拉取
    void load();
  }, [load]);

  // 启用和停用也走修改接口, 需提交全部字段
  const toggleEnabled = async (currency: Currency) => {
    try {
      await services.admin.updateCurrency(currency.code, {...toRequest(currency), is_enabled: !currency.is_enabled});
      toast.success(currency.is_enabled ? `已停用 ${currency.name}` : `已启用 ${currency.name}`);
      await load();
    } catch (err) {
      toast.error(toErrorMessage(err));
    }
  };

  const nextSortOrder = Math.max(0, ...(currencies ?? []).map((c) => c.sort_order)) + 10;

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between gap-2">
        <div className="flex flex-col gap-1">
          <h1 className="text-2xl font-bold">货币管理</h1>
          <p className="text-sm text-muted-foreground">金额按最小单位的整数存储, 小数位只影响展示; 基础货币在系统设置中选择</p>
        </div>
        <Button onClick={() => setEditing('new')}>
          <Plus />
          新建货币
        </Button>
      </div>

      <div className="rounded-xl border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>货币</TableHead>
              <TableHead className="text-right">小数位</TableHead>
              <TableHead className="text-right">排序</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>更新时间</TableHead>
              <TableHead className="text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {!currencies ? (
              <TableSkeleton cols={6} />
            ) : currencies.length ? (
              currencies.map((c) => (
                <CurrencyRow
                  key={c.code}
                  currency={c}
                  isBase={c.code === economy.base_currency}
                  onEdit={() => setEditing(c)}
                  onToggle={() => toggleEnabled(c)}
                />
              ))
            ) : (
              <TableEmpty colSpan={6}>还没有货币</TableEmpty>
            )}
          </TableBody>
        </Table>
      </div>

      <CurrencyDialog
        currency={editing}
        isBase={editing !== 'new' && editing?.code === economy.base_currency}
        nextSortOrder={nextSortOrder}
        onClose={() => setEditing(null)}
        onSaved={load}
      />
    </div>
  );
}

/**
 * 货币列表的一行, 基础货币不能停用
 */
function CurrencyRow({currency: c, isBase, onEdit, onToggle}: {
  currency: Currency;
  isBase: boolean;
  onEdit: () => void;
  onToggle: () => void;
}) {
  return (
    <TableRow>
      <TableCell>
        <div className="flex items-center gap-2">
          <CurrencyIcon currency={c.code} className="size-5" />
          <div className="flex min-w-0 flex-col">
            <span className="font-medium">{c.name}</span>
            <span className="font-mono text-xs text-muted-foreground">{c.code}</span>
          </div>
        </div>
      </TableCell>
      <TableCell className="text-right font-mono tabular-nums">
        <span className="flex flex-col items-end">
          {c.decimals}
          <span className="text-xs text-muted-foreground">如 {formatAmount(12345, c.decimals)}</span>
        </span>
      </TableCell>
      <TableCell className="text-right font-mono tabular-nums">{c.sort_order}</TableCell>
      <TableCell>
        <span className="flex items-center gap-1">
          {c.is_enabled ? <Badge variant="secondary">启用中</Badge> : <Badge variant="outline">已停用</Badge>}
          {isBase && <Badge>基础货币</Badge>}
        </span>
      </TableCell>
      <TableCell className="text-muted-foreground">{formatDateTime(c.updated_at)}</TableCell>
      <TableCell>
        <div className="flex justify-end gap-1">
          <Button variant="ghost" size="icon" title="编辑" aria-label="编辑" onClick={onEdit}>
            <Pencil />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            title={isBase && c.is_enabled ? '基础货币不能停用' : c.is_enabled ? '停用' : '启用'}
            aria-label={c.is_enabled ? '停用' : '启用'}
            className={c.is_enabled ? undefined : 'text-destructive'}
            disabled={isBase && c.is_enabled}
            onClick={onToggle}
          >
            <Power />
          </Button>
        </div>
      </TableCell>
    </TableRow>
  );
}
