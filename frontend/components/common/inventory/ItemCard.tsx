import {cn} from 'cn';
import {ItemImage} from '@/components/common/item';
import type {UserItem} from '@/lib/services/inventory';
import {qualityOf} from '@/lib/quality';

/**
 * 网格里的物品卡片: 品质色条 + 图片 + 数量
 */
export function ItemCard({item, onClick}: { item: UserItem; onClick: () => void }) {
  const q = qualityOf(item.quality);
  return (
    <button
      type="button"
      onClick={onClick}
      className="flex flex-col overflow-hidden rounded-xl border bg-card text-left transition-colors outline-none hover:bg-accent focus-visible:ring-[3px] focus-visible:ring-ring/50"
    >
      <span className={cn('h-1 shrink-0', q.bar)} />
      <span className="relative flex aspect-square items-center justify-center p-4">
        <ItemImage item={item} className={cn(item.is_placeholder && 'opacity-60')} />
        <span className="absolute right-2 bottom-2 rounded-md bg-background/80 px-1.5 py-0.5 font-mono text-xs tabular-nums">
          ×{item.quantity}
        </span>
      </span>
      <span className="flex flex-col gap-0.5 px-3 pb-3">
        <span className={cn('truncate font-medium', q.text)} title={item.name}>
          {item.name}
        </span>
        <span className="text-xs text-muted-foreground">
          {q.label} · {item.width}×{item.height}
        </span>
      </span>
    </button>
  );
}
