import {cn} from 'cn';
import {ItemImage} from '@/components/common/item';
import {useQualities} from '@/components/common/site';
import type {CodexEntry} from '@/lib/services/codex';
import {SILHOUETTE} from './constants';

/**
 * 图鉴中的一格: 收集过的显示图片和品质色, 没有收集的显示剪影和虚线边框
 */
export function CodexCard({entry, onClick}: { entry: CodexEntry; onClick: () => void }) {
  const q = useQualities().qualityOf(entry.quality);
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
          'flex flex-col overflow-hidden rounded-xl border bg-card text-left transition-colors outline-none hover:bg-accent focus-visible:ring-[3px] focus-visible:ring-ring/50',
          !entry.collected && 'border-dashed bg-card/40',
      )}
    >
      <span className={cn('h-1 shrink-0', entry.collected ? q.bar : 'bg-muted')} />
      <span className="flex aspect-square items-center justify-center p-3">
        <ItemImage item={entry} className={cn(!entry.collected && SILHOUETTE)} />
      </span>
      <span
        className={cn('truncate px-2 pb-2 text-center text-xs font-medium', entry.collected ? q.text : 'text-muted-foreground')}
        title={entry.name}
      >
        {entry.name}
      </span>
    </button>
  );
}
