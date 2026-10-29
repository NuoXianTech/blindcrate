import {Copy, Pencil, Power, Trash2} from 'lucide-react';
import {Badge} from '@/components/ui/badge';
import {Button} from '@/components/ui/button';
import {TableCell, TableRow} from '@/components/ui/table';
import {bundleText} from '@/components/common/admin/editors';
import type {AdminItem, AdminShopItem, Currency} from '@/lib/services/admin';
import {formatDateTime} from '@/lib/utils';
import {SALE_STATUS_LABELS, saleStatus} from './constants';

/**
 * 商品行的Props接口
 */
interface ShopItemRowProps {
  item: AdminShopItem;
  items: AdminItem[];
  currencies: Currency[];
  /** 列表加载的时间, 用于判断销售状态 */
  now: number;
  onEdit: () => void;
  onCopy: () => void;
  onToggle: () => void;
  onDelete: () => void;
}

/**
 * 商品列表的一行: 名称、页签、商品内容、价格、库存和销量、上架状态和操作按钮
 */
export function ShopItemRow({item, items, currencies, now, onEdit, onCopy, onToggle, onDelete}: ShopItemRowProps) {
  const goods = bundleText(item.goods, items, currencies);
  const price = bundleText(item.price, items, currencies);
  const original = bundleText(item.original_price, items, currencies);
  const status = saleStatus(item, now);

  return (
    <TableRow>
      <TableCell>
        <div className="flex min-w-0 flex-col">
          <span className="font-medium">{item.name}</span>
          <span className="font-mono text-xs text-muted-foreground">{item.code}</span>
        </div>
      </TableCell>
      <TableCell>{item.tab || <span className="text-muted-foreground">-</span>}</TableCell>
      <TableCell className="max-w-48 truncate" title={goods.join(', ')}>
        {goods.join(', ')}
      </TableCell>
      <TableCell className="max-w-48" title={[...price, ...(original.length ? [`原价 ${original.join(', ')}`] : [])].join(', ') || undefined}>
        <div className="flex flex-col">
          <span className="truncate">{price.length ? price.join(', ') : <span className="text-muted-foreground">免费</span>}</span>
          {original.length > 0 && <span className="truncate text-xs text-muted-foreground line-through">{original.join(', ')}</span>}
        </div>
      </TableCell>
      <TableCell className="text-right font-mono tabular-nums">
        <div className="flex flex-col">
          <span title="已售出">{item.sold}</span>
          <span className="text-xs text-muted-foreground" title="剩余库存">
            {item.stock === null ? '不限' : `余 ${item.stock}`}
          </span>
        </div>
      </TableCell>
      <TableCell>
        <Badge variant={status === 'selling' ? 'secondary' : 'outline'}>{SALE_STATUS_LABELS[status]}</Badge>
      </TableCell>
      <TableCell className="text-xs text-muted-foreground tabular-nums">
        {item.starts_at || item.ends_at ? (
          <div className="flex flex-col">
            <span>{item.starts_at ? formatDateTime(item.starts_at) : '不限'} 起</span>
            <span>{item.ends_at ? formatDateTime(item.ends_at) : '不限'} 止</span>
          </div>
        ) : (
          '不限'
        )}
      </TableCell>
      <TableCell>
        <div className="flex justify-end gap-1">
          <Button variant="ghost" size="icon" title="编辑" aria-label="编辑" onClick={onEdit}>
            <Pencil />
          </Button>
          <Button variant="ghost" size="icon" title="复制为新商品" aria-label="复制为新商品" onClick={onCopy}>
            <Copy />
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
