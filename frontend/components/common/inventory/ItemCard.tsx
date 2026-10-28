import {Check} from 'lucide-react';
import {cn} from 'cn';
import {ItemImage} from '@/components/common/item';
import {useQualities} from '@/components/common/site';
import type {UserItem} from '@/lib/services/inventory';

/**
 * 网格里的物品卡片: 品质色条 + 图片 + 数量; 多选时右上角显示勾选状态
 */
export function ItemCard({item, onClick, selected, disabled}: {
  item: UserItem;
  onClick: () => void;
  /** 多选模式下是否已选中, 不在多选模式时为 undefined */
  selected?: boolean;
  disabled?: boolean;
}) {
  const q = useQualities().qualityOf(item.quality);
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      aria-pressed={selected}
      className={cn(
          'relative flex flex-col overflow-hidden rounded-xl border bg-card text-left transition-colors outline-none hover:bg-accent focus-visible:ring-[3px] focus-visible:ring-ring/50',
          'disabled:pointer-events-none disabled:opacity-40',
          selected && 'border-primary ring-2 ring-primary/40',
      )}
    >
      {selected !== undefined && (
        <span
          className={cn(
              'absolute top-3 right-2 z-10 flex size-5 items-center justify-center rounded-full border bg-background/80',
              selected && 'border-primary bg-primary text-primary-foreground',
          )}
        >
          {selected && <Check className="size-3.5" />}
        </span>
      )}
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
          {q.name} · {item.width}×{item.height}
        </span>
      </span>
    </button>
  );
}
