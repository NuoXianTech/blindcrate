'use client';

import {useState} from 'react';
import {BookOpen} from 'lucide-react';
import {Button} from '@/components/ui/button';
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select';
import {useQualities} from '@/components/common/site';
import type {CodexEntry} from '@/lib/services/codex';
import {CodexCard} from './CodexCard';
import {COLLECTED_FILTERS, type CollectedFilter} from './constants';

/**
 * 藏品网格: 按品质和收集状态筛选, 品质按钮上显示该品质的收集进度; 点击藏品查看详情
 */
export function CodexGrid({items, onView}: { items: CodexEntry[]; onView: (entry: CodexEntry) => void }) {
  const {qualities} = useQualities();
  const [quality, setQuality] = useState<number | undefined>();
  const [filter, setFilter] = useState<CollectedFilter>('all');

  // 只列出图鉴中有藏品的品质
  const groups = qualities
      .map((q) => {
        const list = items.filter((e) => e.quality === q.level);
        return {quality: q, total: list.length, collected: list.filter((e) => e.collected).length};
      })
      .filter((g) => g.total > 0);
  const shown = items.filter(
      (e) => (quality === undefined || e.quality === quality) && (filter === 'all' || e.collected === (filter === 'collected')),
  );

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center gap-2">
        <div className="hide-scrollbar flex min-w-0 flex-1 gap-1 overflow-x-auto">
          <Button variant={quality === undefined ? 'secondary' : 'ghost'} size="sm" onClick={() => setQuality(undefined)}>
            全部
          </Button>
          {groups.map((g) => (
            <Button
              key={g.quality.level}
              variant={quality === g.quality.level ? 'secondary' : 'ghost'}
              size="sm"
              onClick={() => setQuality(g.quality.level)}
            >
              <span className={g.quality.text}>{g.quality.name}</span>
              <span className="font-mono text-xs text-muted-foreground tabular-nums">
                {g.collected}/{g.total}
              </span>
            </Button>
          ))}
        </div>
        <Select value={filter} onValueChange={(value) => setFilter(value as CollectedFilter)}>
          <SelectTrigger size="sm" className="w-24 shrink-0" aria-label="收集状态">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {COLLECTED_FILTERS.map((f) => (
              <SelectItem key={f.value} value={f.value}>
                {f.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {shown.length ? (
        <div className="grid grid-cols-3 gap-3 sm:grid-cols-4 md:grid-cols-6 lg:grid-cols-8">
          {shown.map((entry) => (
            <CodexCard key={entry.item_id} entry={entry} onClick={() => onView(entry)} />
          ))}
        </div>
      ) : (
        <div className="flex h-48 flex-col items-center justify-center gap-2 rounded-xl border bg-card text-muted-foreground">
          <BookOpen className="size-8" />
          <span className="text-sm">{items.length ? '没有符合条件的藏品' : '图鉴里还没有藏品'}</span>
        </div>
      )}
    </div>
  );
}
