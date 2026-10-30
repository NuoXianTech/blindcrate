'use client';

import {useCallback, useEffect, useState} from 'react';
import {ClipboardList, Trophy} from 'lucide-react';
import {toast} from 'sonner';
import {Badge} from '@/components/ui/badge';
import {Skeleton} from '@/components/ui/skeleton';
import {Tabs, TabsContent, TabsList, TabsTrigger} from '@/components/ui/tabs';
import {useAuth} from '@/components/common/auth';
import services from '@/lib/services';
import {toErrorMessage} from '@/lib/services/core';
import type {ClaimMissionResult, Mission, MissionsData} from '@/lib/services/mission';
import {ClaimResultDialog} from './ClaimResultDialog';
import {MissionCard} from './MissionCard';
import {ResetCountdown} from './ResetCountdown';
import {claimableCount, missionRank} from './constants';

/** 进入页面时选中的页签: 第一个有奖励可领的, 都没有时为第一个 */
function initialTab(data: MissionsData) {
  return (data.groups.find((g) => claimableCount(g.missions) > 0) ?? data.groups[0])?.cycle ?? '';
}

/**
 * 任务: 每日、每周、成就三个页签, 每项有进度条和领取按钮; 进入时选中第一个有奖励可领的页签
 * 周期重置时重新拉取; 进度来自对局结算后的统计, 不推送, 重新进入页面时刷新
 */
export function MissionsMain() {
  const {user} = useAuth();
  const [data, setData] = useState<MissionsData | null>(null);
  const [tab, setTab] = useState('');
  const [claiming, setClaiming] = useState<string | null>(null);
  const [result, setResult] = useState<ClaimMissionResult | null>(null);

  const load = useCallback(async () => {
    if (!user) return;
    try {
      const next = await services.mission.list();
      setData(next);
      // 只在第一次加载时自动选择, 之后刷新不切换页签
      setTab((prev) => prev || initialTab(next));
    } catch (err) {
      toast.error(toErrorMessage(err));
    }
  }, [user]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 进入页面时拉取任务
    void load();
  }, [load]);

  const claim = async (mission: Mission) => {
    setClaiming(mission.code);
    try {
      setResult(await services.mission.claim(mission.code));
    } catch (err) {
      toast.error(toErrorMessage(err));
    } finally {
      setClaiming(null);
      void load();
    }
  };

  const groups = data?.groups ?? [];
  const active = groups.some((g) => g.cycle === tab) ? tab : groups[0]?.cycle;

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h1 className="text-2xl font-bold">任务</h1>
        {!!data?.claimable && <Badge>{data.claimable} 个奖励待领取</Badge>}
      </div>

      {!data ? (
        <div className="flex flex-col gap-3">
          <Skeleton className="h-9 w-60" />
          {Array.from({length: 3}, (_, i) => (
            <Skeleton key={i} className="h-28 rounded-xl" />
          ))}
        </div>
      ) : (
        <Tabs value={active} onValueChange={setTab}>
          <TabsList>
            {groups.map((g) => {
              const count = claimableCount(g.missions);
              return (
                <TabsTrigger key={g.cycle} value={g.cycle}>
                  {g.label}
                  {count > 0 && (
                    <span className="flex size-4 items-center justify-center rounded-full bg-destructive text-[10px] text-white tabular-nums">
                      {count}
                    </span>
                  )}
                </TabsTrigger>
              );
            })}
          </TabsList>
          {groups.map((g) => (
            <TabsContent key={g.cycle} value={g.cycle} className="mt-2 flex flex-col gap-3">
              {g.resets_at ? (
                <ResetCountdown at={g.resets_at} onReset={load} />
              ) : (
                <span className="flex items-center gap-1.5 text-sm text-muted-foreground">
                  <Trophy className="size-4" />
                  成就的奖励只能领取一次
                </span>
              )}
              {g.missions.length ? (
                [...g.missions]
                    .sort((a, b) => missionRank(a) - missionRank(b))
                    .map((m) => (
                      <MissionCard
                        key={m.code}
                        mission={m}
                        claiming={claiming === m.code}
                        disabled={claiming !== null}
                        onClaim={() => void claim(m)}
                      />
                    ))
              ) : (
                <div className="flex h-48 flex-col items-center justify-center gap-2 rounded-xl border bg-card text-muted-foreground">
                  <ClipboardList className="size-8" />
                  <span className="text-sm">暂时没有{g.label}任务</span>
                </div>
              )}
            </TabsContent>
          ))}
        </Tabs>
      )}

      <ClaimResultDialog result={result} onClose={() => setResult(null)} />
    </div>
  );
}
