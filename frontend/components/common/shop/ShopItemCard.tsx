import {Check, Clock, X} from 'lucide-react';
import {cn} from 'cn';
import {Badge} from '@/components/ui/badge';
import {Button} from '@/components/ui/button';
import {ItemImage} from '@/components/common/item';
import type {ShopItem} from '@/lib/services/shop';
import {formatDateTime} from '@/lib/utils';
import {GoodsList, PriceText} from './Lines';
import {LIMIT_LABELS, imageOf, unavailableReason} from './constants';

/**
 * 商城中的一件商品: 图片、名称、商品内容、购买条件、限购和库存、价格和购买按钮
 */
export function ShopItemCard({item, onBuy}: { item: ShopItem; onBuy: () => void }) {
  const reason = unavailableReason(item);
  return (
    <div className="flex flex-col overflow-hidden rounded-xl border bg-card">
      <div className="flex gap-3 p-4">
        <span className="flex size-16 shrink-0 items-center justify-center overflow-hidden rounded-lg border bg-background p-1.5">
          <ItemImage item={{name: item.name, image_url: imageOf(item)}} />
        </span>
        <div className="flex min-w-0 flex-1 flex-col gap-1">
          <h2 className="truncate font-bold" title={item.name}>
            {item.name}
          </h2>
          {item.description && <p className="line-clamp-2 text-sm text-muted-foreground">{item.description}</p>}
        </div>
      </div>

      <div className="flex flex-1 flex-col gap-3 px-4 pb-4">
        <GoodsList lines={item.goods} />

        {item.requirements.length > 0 && (
          <ul className="flex flex-col gap-1 text-sm">
            {item.requirements.map((r, i) => (
              <li key={i} className={cn('flex items-center gap-1.5', r.met ? 'text-muted-foreground' : 'text-destructive')}>
                {r.met ? <Check className="size-4 shrink-0" /> : <X className="size-4 shrink-0" />}
                {r.text}
              </li>
            ))}
          </ul>
        )}

        {(item.limits.length > 0 || item.stock !== null || item.ends_at) && (
          <div className="flex flex-wrap gap-1.5">
            {item.limits.map((l) => (
              <Badge key={l.type} variant={l.bought >= l.limit ? 'destructive' : 'outline'} className="tabular-nums">
                {LIMIT_LABELS[l.type]} {l.bought}/{l.limit}
              </Badge>
            ))}
            {item.stock !== null && (
              <Badge variant={item.stock === 0 ? 'destructive' : 'outline'} className="tabular-nums" title="全服剩余库存">
                剩余 {item.stock}
              </Badge>
            )}
            {item.ends_at && (
              <Badge variant="outline" className="tabular-nums">
                <Clock />
                {formatDateTime(item.ends_at)} 下架
              </Badge>
            )}
          </div>
        )}

        <div className="mt-auto flex items-end justify-between gap-3 border-t pt-3">
          <div className="flex min-w-0 flex-col gap-0.5">
            <PriceText lines={item.price} className="font-medium" />
            {item.original_price.length > 0 && (
              <PriceText lines={item.original_price} className="text-xs text-muted-foreground line-through" />
            )}
          </div>
          <Button size="sm" className="shrink-0" disabled={reason !== null} onClick={onBuy}>
            {reason ?? '购买'}
          </Button>
        </div>
      </div>
    </div>
  );
}
