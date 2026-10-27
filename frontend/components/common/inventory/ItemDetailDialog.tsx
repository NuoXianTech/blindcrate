'use client';

import {ScrollText} from 'lucide-react';
import {cn} from 'cn';
import {Badge} from '@/components/ui/badge';
import {Button} from '@/components/ui/button';
import {Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle} from '@/components/ui/dialog';
import {ItemImage} from '@/components/common/item';
import {CurrencyIcon} from '@/components/common/wallet';
import type {UserItem} from '@/lib/services/inventory';
import type {BalanceItem} from '@/lib/services/wallet';
import {qualityOf} from '@/lib/quality';
import {formatAmount, formatDateTime} from '@/lib/utils';

/**
 * 物品详情对话框的Props接口
 */
interface ItemDetailDialogProps {
  /** 要查看的物品, null 时关闭 */
  item: UserItem | null;
  /** 货币列表, 用于展示售价的货币名称和小数位 */
  currencies: BalanceItem[];
  /** 关闭回调 */
  onClose: () => void;
  /** 查看该物品的记录 */
  onViewRecords: (item: UserItem) => void;
}

/**
 * 物品详情对话框
 */
export function ItemDetailDialog({item, currencies, onClose, onViewRecords}: ItemDetailDialogProps) {
  const q = qualityOf(item?.quality ?? 1);
  const currency = currencies.find((c) => c.currency === item?.sell_currency);

  return (
    <Dialog open={!!item} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-md">
        {item && (
          <>
            <DialogHeader className="flex-row items-center gap-4 text-left">
              <span className="flex size-16 shrink-0 items-center justify-center overflow-hidden rounded-lg border bg-card">
                <ItemImage item={item} />
              </span>
              <div className="flex min-w-0 flex-col gap-1">
                <DialogTitle className={cn('truncate', q.text)}>{item.name}</DialogTitle>
                <DialogDescription className="flex flex-wrap items-center gap-2">
                  <span>{q.label}品质</span>
                  {item.is_placeholder ? (
                    <Badge variant="outline">已移除</Badge>
                  ) : (
                    !item.is_enabled && <Badge variant="outline">已下架</Badge>
                  )}
                </DialogDescription>
              </div>
            </DialogHeader>

            {item.description && <p className="text-sm whitespace-pre-wrap text-muted-foreground">{item.description}</p>}
            {item.is_placeholder && <p className="text-sm text-muted-foreground">这个物品已被官方移除, 已有的仍保留在背包中。</p>}

            <dl className="grid grid-cols-2 gap-3 rounded-lg border bg-card p-4 text-sm">
              <Detail label="数量">
                <span className="font-mono tabular-nums">{item.quantity}</span>
              </Detail>
              <Detail label="尺寸">
                {item.width}×{item.height} 格
              </Detail>
              <Detail label="出售价格">
                {item.sell_price > 0 ? (
                  <span className="flex items-center gap-1.5">
                    <CurrencyIcon currency={item.sell_currency} className="size-4" />
                    <span className="font-mono tabular-nums">{formatAmount(item.sell_price, currency?.decimals ?? 0)}</span>
                    <span className="text-muted-foreground">{currency?.name ?? item.sell_currency}</span>
                  </span>
                ) : (
                  <span className="text-muted-foreground">不可出售</span>
                )}
              </Detail>
              <Detail label="首次获得">{formatDateTime(item.created_at)}</Detail>
            </dl>

            <DialogFooter>
              <Button variant="outline" onClick={() => onViewRecords(item)}>
                <ScrollText />
                查看记录
              </Button>
            </DialogFooter>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}

/**
 * 详情里的一项
 */
function Detail({label, children}: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-1">
      <dt className="text-xs text-muted-foreground">{label}</dt>
      <dd>{children}</dd>
    </div>
  );
}
