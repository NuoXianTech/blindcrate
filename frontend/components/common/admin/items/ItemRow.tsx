import {Pencil, Power, Trash2} from 'lucide-react';
import {cn} from 'cn';
import {Badge} from '@/components/ui/badge';
import {Button} from '@/components/ui/button';
import {TableCell, TableRow} from '@/components/ui/table';
import {ItemImage} from '@/components/common/item';
import {CurrencyIcon} from '@/components/common/wallet';
import {useQualities} from '@/components/common/site';
import type {AdminItem, Currency, ItemCategory} from '@/lib/services/admin';
import {formatAmount, formatDateTime} from '@/lib/utils';

/**
 * 物品行的Props接口
 */
interface ItemRowProps {
  item: AdminItem;
  /** 回收货币, 用于展示名称和小数位 */
  currency?: Currency;
  /** 物品类别, 未注册的类别按代码展示 */
  category?: ItemCategory;
  onEdit: () => void;
  onToggle: () => void;
  onDelete: () => void;
}

/**
 * 物品列表的一行: 物品信息和操作按钮
 */
export function ItemRow({item, currency, category, onEdit, onToggle, onDelete}: ItemRowProps) {
  const q = useQualities().qualityOf(item.quality);
  return (
    <TableRow>
      <TableCell>
        <div className="flex items-center gap-3">
          <span className="relative flex size-10 shrink-0 items-center justify-center overflow-hidden rounded-md border bg-background">
            <ItemImage item={item} />
            <span className={cn('absolute inset-x-0 top-0 h-0.5', q.bar)} />
          </span>
          <div className="flex min-w-0 flex-col">
            <span className={cn('font-medium', q.text)}>{item.name}</span>
            <span className="max-w-56 truncate text-xs text-muted-foreground" title={item.description || undefined}>
              <span className="font-mono">{item.code}</span>
              {item.description && ` · ${item.description}`}
            </span>
          </div>
        </div>
      </TableCell>
      <TableCell>
        <span className={q.text}>{q.name}</span>
      </TableCell>
      <TableCell>
        <Badge variant="outline">{category?.label ?? item.category}</Badge>
      </TableCell>
      <TableCell className="text-right">
        {item.sell_price > 0 ? (
          <span className="inline-flex items-center gap-1.5" title={currency?.name ?? item.sell_currency}>
            <CurrencyIcon currency={item.sell_currency} className="size-4" />
            <span className="font-mono tabular-nums">{formatAmount(item.sell_price, currency?.decimals ?? 0)}</span>
          </span>
        ) : (
          <span className="text-muted-foreground">不可出售</span>
        )}
      </TableCell>
      <TableCell className="text-muted-foreground">
        {item.width}×{item.height}
      </TableCell>
      <TableCell className="text-right font-mono tabular-nums">{item.holder_count}</TableCell>
      <TableCell>
        {item.is_enabled ? <Badge variant="secondary">上架中</Badge> : <Badge variant="outline">已下架</Badge>}
      </TableCell>
      <TableCell className="text-muted-foreground">{formatDateTime(item.updated_at)}</TableCell>
      <TableCell>
        <div className="flex justify-end gap-1">
          <Button variant="ghost" size="icon" title="编辑" aria-label="编辑" onClick={onEdit}>
            <Pencil />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            title={item.is_enabled ? '下架' : '上架'}
            aria-label={item.is_enabled ? '下架' : '上架'}
            className={item.is_enabled ? undefined : 'text-destructive'}
            onClick={onToggle}
          >
            <Power />
          </Button>
          <Button variant="ghost" size="icon" title="删除" aria-label="删除" className="hover:text-destructive" onClick={onDelete}>
            <Trash2 />
          </Button>
        </div>
      </TableCell>
    </TableRow>
  );
}
