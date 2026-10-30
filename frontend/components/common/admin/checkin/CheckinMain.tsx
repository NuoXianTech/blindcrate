'use client';

import {useCallback, useEffect, useState} from 'react';
import {CalendarPlus, Trash2, Undo2} from 'lucide-react';
import {nanoid} from 'nanoid';
import {toast} from 'sonner';
import {Button} from '@/components/ui/button';
import {Skeleton} from '@/components/ui/skeleton';
import {BundleEditor, fromBundleDrafts, toBundleDrafts, type BundleDraft} from '@/components/common/admin/editors';
import {useAuth} from '@/components/common/auth';
import {SubmitButton} from '@/components/common/form';
import {useAdminCatalog} from '@/hooks/use-admin-catalog';
import {check, useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';
import type {Currency} from '@/lib/services/admin';
import type {AdminCheckinDay} from '@/lib/services/checkin';
import {toErrorMessage} from '@/lib/services/core';

/** 一天的奖励最多的行数, 与后端 reward.MaxLines 一致 */
const MAX_BUNDLE_LINES = 8;

/** 编辑中的一天, key 用于列表渲染, 删除中间的天数后后面的天数依次提前 */
interface DayDraft {
  key: string;
  reward: BundleDraft[];
}

function toDayDrafts(days: AdminCheckinDay[], currencies: Currency[]): DayDraft[] {
  return days.map((d) => ({key: nanoid(), reward: toBundleDrafts(d.reward, currencies)}));
}

/**
 * 签到管理: 按天编辑签到奖励, 整体保存
 * 连续签到超过周期的天数时从第 1 天重新开始; 签到的开关在系统设置中
 */
export function CheckinMain() {
  const {user: me} = useAuth();
  const isAdmin = !!me?.is_admin;
  const catalog = useAdminCatalog();
  const [days, setDays] = useState<DayDraft[] | null>(null);
  const [maxDays, setMaxDays] = useState(0);
  const [dirty, setDirty] = useState(false);
  const {pending, submit} = useSubmit();

  const load = useCallback(async () => {
    // 确认是管理员后再拉取, 否则 AdminGuard 会跳走; 金额的小数位要等货币加载完
    if (!isAdmin || !catalog.loaded) return;
    try {
      const data = await services.checkinAdmin.listDays();
      setDays(toDayDrafts(data.days, catalog.currencies));
      setMaxDays(data.max_days);
      setDirty(false);
    } catch (err) {
      toast.error(toErrorMessage(err));
    }
  }, [isAdmin, catalog.loaded, catalog.currencies]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 目录加载完后拉取签到奖励
    void load();
  }, [load]);

  const change = (next: DayDraft[]) => {
    setDays(next);
    setDirty(true);
  };

  const handleSubmit = submit(async () => {
    check(!days?.length ? '至少需要配置 1 天' : null);
    const body = {
      days: days!.map((d, i) => {
        check(d.reward.length === 0 ? `第 ${i + 1} 天的奖励不能为空` : null);
        return {reward: fromBundleDrafts(d.reward, catalog.currencies, `第 ${i + 1} 天的奖励`)};
      }),
    };
    const data = await services.checkinAdmin.updateDays(body);
    setDays(toDayDrafts(data.days, catalog.currencies));
    setDirty(false);
    toast.success(`已保存 ${data.days.length} 天的签到奖励`);
  });

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4" noValidate>
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex flex-col gap-0.5">
          <h1 className="text-2xl font-bold">签到管理</h1>
          <p className="text-sm text-muted-foreground">
            玩家按连续签到的天数领取对应一天的奖励, 超过周期的天数或断签后从第 1 天重新开始; 签到的开关在系统设置中
          </p>
        </div>
        <div className="flex items-center gap-2">
          {dirty && (
            <Button type="button" variant="outline" disabled={pending} onClick={() => void load()}>
              <Undo2 />
              撤销修改
            </Button>
          )}
          <SubmitButton pending={pending} disabled={!days || !dirty}>
            保存
          </SubmitButton>
        </div>
      </div>

      {days === null ? (
        <div className="grid gap-3 lg:grid-cols-2">
          {Array.from({length: 4}, (_, i) => (
            <Skeleton key={i} className="h-32 rounded-xl" />
          ))}
        </div>
      ) : (
        <>
          {days.length === 0 && (
            <div className="flex h-32 items-center justify-center rounded-xl border bg-card text-sm text-muted-foreground">
              还没有配置签到奖励, 玩家不能签到
            </div>
          )}
          <div className="grid gap-3 lg:grid-cols-2">
            {days.map((d, i) => (
              <section key={d.key} className="flex flex-col gap-3 rounded-xl border bg-card p-4">
                <div className="flex items-center justify-between gap-2">
                  <h2 className="font-bold">第 {i + 1} 天</h2>
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    className="text-muted-foreground"
                    title="删除这一天, 后面的天数依次提前"
                    disabled={pending}
                    onClick={() => change(days.filter((x) => x.key !== d.key))}
                  >
                    <Trash2 />
                    删除这天
                  </Button>
                </div>
                <BundleEditor
                  drafts={d.reward}
                  onChange={(reward) => change(days.map((x) => (x.key === d.key ? {...x, reward} : x)))}
                  items={catalog.items}
                  currencies={catalog.currencies}
                  max={MAX_BUNDLE_LINES}
                  empty="还没有奖励"
                  disabled={pending}
                />
              </section>
            ))}
          </div>
          <Button
            type="button"
            variant="outline"
            className="self-start"
            disabled={pending || days.length >= maxDays}
            onClick={() => change([...days, {key: nanoid(), reward: []}])}
          >
            <CalendarPlus />
            添加一天{days.length >= maxDays && ` (最多 ${maxDays} 天)`}
          </Button>
        </>
      )}
    </form>
  );
}
