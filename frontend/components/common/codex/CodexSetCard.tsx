import {CircleCheck, LoaderCircle} from 'lucide-react';
import {cn} from 'cn';
import {Button} from '@/components/ui/button';
import {ItemImage} from '@/components/common/item';
import {PriceText} from '@/components/common/reward';
import type {CodexEntry, CodexSet} from '@/lib/services/codex';
import {ProgressBar} from './ProgressBar';
import {SET_THUMBNAILS, SILHOUETTE} from './constants';

/**
 * 收集组卡片的Props接口
 */
interface CodexSetCardProps {
  set: CodexSet;
  /** 图鉴中的藏品, 按物品 ID */
  entries: Map<number, CodexEntry>;
  /** 正在领取这个收集组 */
  claiming: boolean;
  /** 有收集组正在领取时其他按钮也不能点 */
  disabled: boolean;
  onClaim: () => void;
  /** 点击缩略图查看藏品详情 */
  onView: (entry: CodexEntry) => void;
}

/** 领取按钮上的文字 */
function buttonText(set: CodexSet) {
  if (set.claimed) return '已领取';
  if (set.claimable) return '领取';
  return '未集齐';
}

/**
 * 一个收集组: 名称、说明、进度条、组内藏品的缩略图、奖励和领取按钮; 没有收集的藏品显示为剪影
 */
export function CodexSetCard({set, entries, claiming, disabled, onClaim, onView}: CodexSetCardProps) {
  const members = set.item_ids.flatMap((id) => entries.get(id) ?? []);
  const shown = members.slice(0, SET_THUMBNAILS);
  const total = set.item_ids.length;

  return (
    <div className={cn('flex flex-col gap-3 rounded-xl border bg-card p-4 sm:flex-row sm:items-center', set.claimed && 'opacity-60')}>
      <div className="flex min-w-0 flex-1 flex-col gap-2">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="flex min-w-0 items-center gap-1.5 font-bold">
            {set.claimed && <CircleCheck className="size-4 shrink-0 text-quality-green" />}
            <span className="truncate" title={set.name}>
              {set.name}
            </span>
          </h2>
          {set.description && <p className="text-sm text-muted-foreground">{set.description}</p>}
        </div>

        <div className="flex items-center gap-3">
          <ProgressBar value={set.collected} max={total} label={`${set.name}进度`} className="flex-1" />
          <span className="shrink-0 font-mono text-sm tabular-nums">
            {set.collected}/{total}
          </span>
        </div>

        <div className="flex flex-wrap items-center gap-1.5">
          {shown.map((e) => (
            <button
              key={e.item_id}
              type="button"
              title={e.name}
              aria-label={e.name}
              onClick={() => onView(e)}
              className={cn(
                  'flex size-9 items-center justify-center overflow-hidden rounded-md border bg-background p-0.5 transition-colors outline-none hover:bg-accent focus-visible:ring-[3px] focus-visible:ring-ring/50',
                  !e.collected && 'border-dashed',
              )}
            >
              <ItemImage item={e} className={cn(!e.collected && SILHOUETTE)} />
            </button>
          ))}
          {members.length > shown.length && (
            <span className="px-1 text-xs text-muted-foreground tabular-nums">+{members.length - shown.length}</span>
          )}
        </div>
      </div>

      <div className="flex items-center justify-between gap-3 border-t pt-3 sm:w-56 sm:flex-col sm:items-end sm:border-t-0 sm:border-l sm:pt-0 sm:pl-4">
        <PriceText lines={set.reward} className="text-sm sm:justify-end" />
        <Button size="sm" className="shrink-0" disabled={!set.claimable || disabled} onClick={onClaim}>
          {claiming && <LoaderCircle className="animate-spin" />}
          {buttonText(set)}
        </Button>
      </div>
    </div>
  );
}
