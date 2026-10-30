'use client';

import {useCallback, useEffect, useState} from 'react';
import {toast} from 'sonner';
import {Badge} from '@/components/ui/badge';
import {Table, TableBody, TableCell, TableHead, TableHeader, TableRow} from '@/components/ui/table';
import {useAuth} from '@/components/common/auth';
import {Pagination, TableEmpty, TableSkeleton} from '@/components/common/layout';
import services from '@/lib/services';
import {toErrorMessage} from '@/lib/services/core';
import {SHOP_ORDER_STATUS, type ListShopOrdersData} from '@/lib/services/shop';
import {formatDateTime} from '@/lib/utils';
import {PriceText, RewardList} from '@/components/common/reward';
import {ORDER_PAGE_SIZE} from './constants';

/**
 * 购买记录: 每次购买一行, 显示支付的价格、得到的商品和是否已退款
 */
export function OrderList() {
  const {user} = useAuth();
  const [current, setCurrent] = useState(1);
  const [data, setData] = useState<ListShopOrdersData | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    if (!user) return;
    setLoading(true);
    try {
      setData(await services.shop.listOrders({current, size: ORDER_PAGE_SIZE}));
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

  return (
    <div className="flex flex-col gap-4">
      <div className="rounded-xl border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>时间</TableHead>
              <TableHead>商品</TableHead>
              <TableHead>支付</TableHead>
              <TableHead>得到</TableHead>
              <TableHead>状态</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading && !data ? (
              <TableSkeleton cols={5} />
            ) : data?.orders.length ? (
              data.orders.map((order) => (
                <TableRow key={order.id} className="align-top">
                  <TableCell className="text-muted-foreground">{formatDateTime(order.created_at)}</TableCell>
                  <TableCell>
                    <span className="flex items-center gap-1.5">
                      <span className="max-w-48 truncate" title={order.name}>
                        {order.name}
                      </span>
                      <span className="font-mono text-muted-foreground tabular-nums">×{order.quantity}</span>
                    </span>
                  </TableCell>
                  <TableCell>
                    <PriceText lines={order.price} />
                  </TableCell>
                  <TableCell>
                    <RewardList lines={order.goods} className="min-w-40" />
                  </TableCell>
                  <TableCell>
                    {order.status === SHOP_ORDER_STATUS.refunded ? (
                      <Badge variant="secondary" title={order.refunded_at ? `${formatDateTime(order.refunded_at)} 退款` : undefined}>
                        已退款
                      </Badge>
                    ) : (
                      <Badge variant="outline">已支付</Badge>
                    )}
                  </TableCell>
                </TableRow>
              ))
            ) : (
              <TableEmpty colSpan={5}>还没有购买过商品</TableEmpty>
            )}
          </TableBody>
        </Table>
      </div>

      <Pagination current={current} size={ORDER_PAGE_SIZE} total={data?.total ?? 0} onChange={setCurrent} />
    </div>
  );
}
