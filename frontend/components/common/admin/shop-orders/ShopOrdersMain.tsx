'use client';

import {useCallback, useEffect, useState} from 'react';
import {Undo2} from 'lucide-react';
import {toast} from 'sonner';
import {Badge} from '@/components/ui/badge';
import {Button} from '@/components/ui/button';
import {Input} from '@/components/ui/input';
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select';
import {Table, TableBody, TableCell, TableHead, TableHeader, TableRow} from '@/components/ui/table';
import {bundleText} from '@/components/common/admin/editors';
import {Actor, IDFilterForm} from '@/components/common/admin/transactions/TransactionFilters';
import type {IDFilters} from '@/components/common/admin/transactions/constants';
import {useAuth} from '@/components/common/auth';
import {Pagination, TableEmpty, TableSkeleton} from '@/components/common/layout';
import {useAdminCatalog} from '@/hooks/use-admin-catalog';
import services from '@/lib/services';
import {SHOP_ORDER_STATUS, type AdminShopItem, type AdminShopOrder, type ListShopOrdersData} from '@/lib/services/admin';
import {toErrorMessage} from '@/lib/services/core';
import {formatDateTime} from '@/lib/utils';
import {RefundOrderDialog} from './RefundOrderDialog';
import {ALL, PAGE_SIZE, SHOP_ITEM_LIMIT, STATUS_LABELS, dayStart} from './constants';

/**
 * 商城订单: 按用户、商品、状态和日期筛选, 给已支付的订单退款
 */
export function ShopOrdersMain() {
  const isAdmin = !!useAuth().user?.is_admin;
  const [ids, setIDs] = useState<IDFilters>({});
  const [shopItem, setShopItem] = useState(ALL);
  const [status, setStatus] = useState(ALL);
  const [startDate, setStartDate] = useState('');
  const [endDate, setEndDate] = useState('');
  const [current, setCurrent] = useState(1);
  const [data, setData] = useState<ListShopOrdersData | null>(null);
  const [loading, setLoading] = useState(true);
  const [shopItems, setShopItems] = useState<AdminShopItem[]>([]);
  const [refunding, setRefunding] = useState<AdminShopOrder | null>(null);
  const catalog = useAdminCatalog();

  useEffect(() => {
    if (!isAdmin) return;
    services.admin.listShopItems({current: 1, size: SHOP_ITEM_LIMIT})
        .then((res) => setShopItems(res.items))
        .catch(() => {
          // 只影响商品筛选的选项
        });
  }, [isAdmin]);

  const load = useCallback(async () => {
    // 确认是管理员后再拉取, 否则 AdminGuard 会跳走
    if (!isAdmin) return;
    setLoading(true);
    try {
      setData(await services.admin.listShopOrders({
        current,
        size: PAGE_SIZE,
        user_id: ids.user_id,
        shop_item_id: shopItem === ALL ? undefined : Number(shopItem),
        status: status === ALL ? undefined : (status as AdminShopOrder['status']),
        start_time: dayStart(startDate),
        // 结束日期包含当天
        end_time: dayStart(endDate, 1),
      }));
    } catch (err) {
      toast.error(toErrorMessage(err));
    } finally {
      setLoading(false);
    }
  }, [current, ids, shopItem, status, startDate, endDate, isAdmin]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 翻页和筛选时重新拉取
    void load();
  }, [load]);

  const changeFilter = (set: (value: string) => void) => (value: string) => {
    set(value);
    setCurrent(1);
  };
  const textOf = (bundle: AdminShopOrder['price']) => bundleText(bundle, catalog.items, catalog.currencies).join(', ');

  return (
    <div className="flex flex-col gap-4">
      <h1 className="text-2xl font-bold">商城订单</h1>

      <div className="flex flex-col gap-2 lg:flex-row">
        <IDFilterForm
          fields={[{key: 'user_id', label: '用户 ID'}]}
          values={ids}
          onSubmit={(next) => {
            setCurrent(1);
            setIDs(next);
          }}
        />
        <Select value={shopItem} onValueChange={changeFilter(setShopItem)}>
          <SelectTrigger className="lg:w-44" aria-label="商品">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL}>全部商品</SelectItem>
            {shopItems.map((i) => (
              <SelectItem key={i.id} value={String(i.id)}>
                {i.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select value={status} onValueChange={changeFilter(setStatus)}>
          <SelectTrigger className="lg:w-32" aria-label="状态">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL}>全部状态</SelectItem>
            {Object.entries(STATUS_LABELS).map(([value, label]) => (
              <SelectItem key={value} value={value}>
                {label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <div className="flex items-center gap-2">
          <Input
            type="date"
            aria-label="开始日期"
            className="lg:w-40"
            value={startDate}
            onChange={(e) => changeFilter(setStartDate)(e.target.value)}
          />
          <span className="text-muted-foreground">至</span>
          <Input
            type="date"
            aria-label="结束日期"
            className="lg:w-40"
            value={endDate}
            onChange={(e) => changeFilter(setEndDate)(e.target.value)}
          />
        </div>
      </div>

      <div className="rounded-xl border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>订单</TableHead>
              <TableHead>用户</TableHead>
              <TableHead>商品</TableHead>
              <TableHead>支付</TableHead>
              <TableHead>得到</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>时间</TableHead>
              <TableHead className="w-0" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading && !data ? (
              <TableSkeleton cols={8} />
            ) : data?.orders.length ? (
              data.orders.map((o) => {
                const price = textOf(o.price);
                const goods = textOf(o.goods);
                return (
                  <TableRow key={o.id}>
                    <TableCell className="font-mono tabular-nums">#{o.id}</TableCell>
                    <TableCell>
                      <Actor id={o.user_id} username={o.username} />
                    </TableCell>
                    <TableCell>
                      <div className="flex flex-col">
                        <span>
                          {o.name} <span className="text-muted-foreground">×{o.quantity}</span>
                        </span>
                        <span className="font-mono text-xs text-muted-foreground">{o.item_code}</span>
                      </div>
                    </TableCell>
                    <TableCell className="max-w-48 truncate" title={price || undefined}>
                      {price || <span className="text-muted-foreground">免费</span>}
                    </TableCell>
                    <TableCell className="max-w-48 truncate" title={goods}>
                      {goods}
                    </TableCell>
                    <TableCell>
                      {o.status === SHOP_ORDER_STATUS.refunded ? (
                        <div className="flex flex-col gap-0.5">
                          <Badge variant="outline">{STATUS_LABELS[o.status]}</Badge>
                          <span className="text-xs text-muted-foreground">
                            {o.refunded_username ? `@${o.refunded_username}` : `#${o.refunded_by}`} · {formatDateTime(o.refunded_at)}
                          </span>
                        </div>
                      ) : (
                        <Badge variant="secondary">{STATUS_LABELS[o.status] ?? o.status}</Badge>
                      )}
                    </TableCell>
                    <TableCell className="text-muted-foreground tabular-nums">{formatDateTime(o.created_at)}</TableCell>
                    <TableCell>
                      {o.status === SHOP_ORDER_STATUS.paid && (
                        <Button
                          variant="ghost"
                          size="icon"
                          title="退款"
                          aria-label="退款"
                          className="hover:text-destructive"
                          onClick={() => setRefunding(o)}
                        >
                          <Undo2 />
                        </Button>
                      )}
                    </TableCell>
                  </TableRow>
                );
              })
            ) : (
              <TableEmpty colSpan={8}>没有找到订单</TableEmpty>
            )}
          </TableBody>
        </Table>
      </div>

      <Pagination current={current} size={PAGE_SIZE} total={data?.total ?? 0} onChange={setCurrent} />

      <RefundOrderDialog
        order={refunding}
        price={refunding ? textOf(refunding.price) : ''}
        goods={refunding ? textOf(refunding.goods) : ''}
        onClose={() => setRefunding(null)}
        onSaved={load}
      />
    </div>
  );
}
