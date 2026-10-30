'use client';

import {useCallback, useEffect, useRef, useState} from 'react';
import {CalendarCheck, Check, LoaderCircle} from 'lucide-react';
import {toast} from 'sonner';
import {cn} from 'cn';
import {Button} from '@/components/ui/button';
import {Skeleton} from '@/components/ui/skeleton';
import {useAuth} from '@/components/common/auth';
import {ItemImage} from '@/components/common/item';
import {CurrencyIcon} from '@/components/common/wallet';
import services from '@/lib/services';
import type {CheckinDay, CheckinResult, CheckinStatus} from '@/lib/services/checkin';
import {toErrorMessage} from '@/lib/services/core';
import type {RewardLine} from '@/lib/services/reward';
import {formatAmount} from '@/lib/utils';
import {CheckinResultDialog} from './CheckinResultDialog';

/** 一行奖励的文字, 用于悬停提示 */
function lineText(line: RewardLine) {
  return line.currency ? `${line.name} ${formatAmount(line.amount, line.decimals)}` : `${line.name} ×${line.amount}`;
}

/**
 * 日历中的一天: 第几天、第一项奖励的图标和数量, 其余奖励在悬停提示中
 * 今天的格子高亮, 还没签到时可以点击签到; 本周期已签过的天数打勾
 */
function DayCell({day, done, today, onClick}: {
  day: CheckinDay;
  done: boolean;
  today: boolean;
  /** 今天还没签到时才有 */
  onClick?: () => void;
}) {
  const [first] = day.reward;
  const more = day.reward.length - 1;
  const title = `第 ${day.day} 天: ${day.reward.map(lineText).join('、')}`;
  const content = (
    <>
      <span className={cn('text-xs', today ? 'font-medium text-primary' : 'text-muted-foreground')}>
        {today ? '今天' : `第 ${day.day} 天`}
      </span>
      <span className="relative flex size-10 items-center justify-center overflow-hidden rounded-md border bg-background p-1">
        {first?.currency ? <CurrencyIcon currency={first.currency} className="size-5" /> : first && <ItemImage item={first} />}
        {done && (
          <span className="absolute inset-0 flex items-center justify-center bg-background/70">
            <Check className="size-5 text-quality-green" />
          </span>
        )}
      </span>
      <span className="max-w-full truncate font-mono text-xs tabular-nums">
        {first && (first.currency ? formatAmount(first.amount, first.decimals) : `×${first.amount}`)}
        {more > 0 && <span className="text-muted-foreground"> +{more}</span>}
      </span>
    </>
  );
  const className = cn(
      'flex w-20 shrink-0 flex-col items-center gap-1 rounded-lg border p-2',
      today && 'border-primary bg-primary/5',
      done && !today && 'opacity-60',
  );
  if (onClick) {
    return (
      <button type="button" title={title} className={cn(className, 'ring-2 ring-primary/40 transition-colors hover:bg-primary/10')} onClick={onClick}>
        {content}
      </button>
    );
  }
  return (
    <div title={title} className={className}>
      {content}
    </div>
  );
}

/**
 * 大厅的签到卡片: 连续和累计天数、整个周期的日历和签到按钮
 * 签到关闭、没有配置奖励或拉取失败时不显示; 回到页面时重新拉取, 过了零点后可以再签
 */
export function CheckinCard({onCheckedIn}: {
  /** 签到成功后调用, 用于刷新任务等依赖签到天数的信息 */
  onCheckedIn?: () => void;
}) {
  const {user} = useAuth();
  const [status, setStatus] = useState<CheckinStatus | null>(null);
  const [failed, setFailed] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [result, setResult] = useState<CheckinResult | null>(null);
  const listRef = useRef<HTMLDivElement>(null);
  const todayRef = useRef<HTMLDivElement>(null);

  const load = useCallback(async () => {
    try {
      setStatus(await services.checkin.getStatus());
      setFailed(false);
    } catch {
      // 只是大厅的附加信息, 失败时不显示卡片
      setFailed(true);
    }
  }, []);

  useEffect(() => {
    if (!user) return;
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 进入大厅时拉取签到情况
    void load();
    const onVisible = () => {
      if (document.visibilityState === 'visible') void load();
    };
    document.addEventListener('visibilitychange', onVisible);
    return () => document.removeEventListener('visibilitychange', onVisible);
  }, [user, load]);

  // 天数多时把今天滚到可见范围的中间, 只滚动日历本身
  useEffect(() => {
    const list = listRef.current;
    const cell = todayRef.current;
    if (!list || !cell) return;
    list.scrollLeft = cell.offsetLeft - (list.clientWidth - cell.clientWidth) / 2;
  }, [status?.today]);

  const checkin = async () => {
    setSubmitting(true);
    try {
      setResult(await services.checkin.checkin());
      onCheckedIn?.();
    } catch (err) {
      toast.error(toErrorMessage(err));
    } finally {
      setSubmitting(false);
      void load();
    }
  };

  if (failed && !status) return null;
  if (!status) return <Skeleton className="h-36 rounded-xl" />;
  if (!status.enabled || !status.days.length) return null;

  const canCheckin = !status.checked_in && !submitting;
  return (
    <section className="flex flex-col gap-3 rounded-xl border bg-card p-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex flex-col gap-0.5">
          <h2 className="flex items-center gap-1.5 font-bold">
            <CalendarCheck className="size-4" />
            每日签到
          </h2>
          <p className="text-xs text-muted-foreground">
            已连续签到 <span className="font-mono text-foreground tabular-nums">{status.streak}</span> 天, 累计{' '}
            <span className="font-mono text-foreground tabular-nums">{status.total}</span> 天; 断签后从第 1 天重新开始
          </p>
        </div>
        <Button size="sm" disabled={!canCheckin} onClick={checkin}>
          {submitting && <LoaderCircle className="animate-spin" />}
          {status.checked_in ? '今日已签到' : '签到'}
        </Button>
      </div>

      <div ref={listRef} className="relative flex gap-2 overflow-x-auto pb-1">
        {status.days.map((d) => {
          const today = d.day === status.today;
          return (
            <div key={d.day} ref={today ? todayRef : undefined} className="shrink-0">
              <DayCell
                day={d}
                today={today}
                done={d.day < status.today || (today && status.checked_in)}
                onClick={today && canCheckin ? checkin : undefined}
              />
            </div>
          );
        })}
      </div>

      <CheckinResultDialog result={result} onClose={() => setResult(null)} />
    </section>
  );
}
