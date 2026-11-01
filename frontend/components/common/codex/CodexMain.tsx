'use client';

import {useCallback, useEffect, useMemo, useState} from 'react';
import {Gem, Layers} from 'lucide-react';
import {toast} from 'sonner';
import {Badge} from '@/components/ui/badge';
import {Skeleton} from '@/components/ui/skeleton';
import {Tabs, TabsContent, TabsList, TabsTrigger} from '@/components/ui/tabs';
import {useAuth} from '@/components/common/auth';
import {RewardResultDialog} from '@/components/common/reward';
import services from '@/lib/services';
import type {ClaimCodexSetResult, CodexData, CodexEntry, CodexSet} from '@/lib/services/codex';
import {toErrorMessage} from '@/lib/services/core';
import {CodexEntryDialog} from './CodexEntryDialog';
import {CodexGrid} from './CodexGrid';
import {CodexSetCard} from './CodexSetCard';
import {ProgressBar} from './ProgressBar';
import {setRank} from './constants';

/**
 * 图鉴: 总收集进度, 藏品和收集组两个页签; 进入时有收集组可以领取就选中收集组
 * 收集情况取自背包, 不推送, 重新进入页面时刷新
 */
export function CodexMain() {
  const {user} = useAuth();
  const [data, setData] = useState<CodexData | null>(null);
  const [tab, setTab] = useState('');
  const [viewing, setViewing] = useState<CodexEntry | null>(null);
  const [claiming, setClaiming] = useState<string | null>(null);
  const [result, setResult] = useState<ClaimCodexSetResult | null>(null);

  const load = useCallback(async () => {
    if (!user) return;
    try {
      const next = await services.codex.getCodex();
      setData(next);
      // 只在第一次加载时自动选择, 之后刷新不切换页签
      setTab((prev) => prev || (next.claimable > 0 ? 'sets' : 'items'));
    } catch (err) {
      toast.error(toErrorMessage(err));
    }
  }, [user]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 进入页面时拉取图鉴
    void load();
  }, [load]);

  const claim = async (set: CodexSet) => {
    setClaiming(set.code);
    try {
      setResult(await services.codex.claimSet(set.code));
    } catch (err) {
      toast.error(toErrorMessage(err));
    } finally {
      setClaiming(null);
      void load();
    }
  };

  const entries = useMemo(() => new Map(data?.items.map((e) => [e.item_id, e])), [data]);
  const sets = useMemo(() => [...(data?.sets ?? [])].sort((a, b) => setRank(a) - setRank(b)), [data]);
  const total = data?.items.length ?? 0;

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h1 className="text-2xl font-bold">图鉴</h1>
        {!!data?.claimable && <Badge>{data.claimable} 个收集奖励待领取</Badge>}
      </div>

      {!data ? (
        <div className="flex flex-col gap-3">
          <Skeleton className="h-20 rounded-xl" />
          <Skeleton className="h-9 w-48" />
          <div className="grid grid-cols-3 gap-3 sm:grid-cols-4 md:grid-cols-6 lg:grid-cols-8">
            {Array.from({length: 8}, (_, i) => (
              <Skeleton key={i} className="aspect-[4/5] rounded-xl" />
            ))}
          </div>
        </div>
      ) : (
        <>
          <section className="flex flex-col gap-2 rounded-xl border bg-card p-4">
            <div className="flex items-baseline justify-between gap-2">
              <span className="text-sm text-muted-foreground">收集进度</span>
              <span className="font-mono tabular-nums">
                <span className="text-xl font-bold">{data.collected}</span>
                <span className="text-muted-foreground">/{total}</span>
              </span>
            </div>
            <ProgressBar value={data.collected} max={total} label="图鉴进度" />
          </section>

          <Tabs value={tab} onValueChange={setTab}>
            <TabsList>
              <TabsTrigger value="items">
                <Gem />
                藏品
              </TabsTrigger>
              <TabsTrigger value="sets">
                <Layers />
                收集组
                {data.claimable > 0 && (
                  <span className="flex size-4 items-center justify-center rounded-full bg-destructive text-[10px] text-white tabular-nums">
                    {data.claimable}
                  </span>
                )}
              </TabsTrigger>
            </TabsList>
            <TabsContent value="items" className="mt-2">
              <CodexGrid items={data.items} onView={setViewing} />
            </TabsContent>
            <TabsContent value="sets" className="mt-2 flex flex-col gap-3">
              {sets.length ? (
                sets.map((s) => (
                  <CodexSetCard
                    key={s.code}
                    set={s}
                    entries={entries}
                    claiming={claiming === s.code}
                    disabled={claiming !== null}
                    onClaim={() => void claim(s)}
                    onView={setViewing}
                  />
                ))
              ) : (
                <div className="flex h-48 flex-col items-center justify-center gap-2 rounded-xl border bg-card text-muted-foreground">
                  <Layers className="size-8" />
                  <span className="text-sm">暂时没有收集组</span>
                </div>
              )}
            </TabsContent>
          </Tabs>
        </>
      )}

      <CodexEntryDialog entry={viewing} sets={data?.sets ?? []} onClose={() => setViewing(null)} />
      <RewardResultDialog lines={result?.received ?? null} onClose={() => setResult(null)} />
    </div>
  );
}
