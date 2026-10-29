'use client';

import {useEffect, useState} from 'react';
import {CalendarRange, Gift} from 'lucide-react';
import {cn} from 'cn';
import {Popover, PopoverContent, PopoverTrigger} from '@/components/ui/popover';
import {useAuth} from '@/components/common/auth';
import {TierName} from '@/components/common/ranked';
import {CurrencyIcon} from '@/components/common/wallet';
import services from '@/lib/services';
import {tierOf} from '@/lib/ranked';
import type {Tier} from '@/lib/services/ranked';
import type {CurrentSeasonData, SeasonRewardView} from '@/lib/services/season';
import {formatAmount} from '@/lib/utils';

const HOUR = 60 * 60 * 1000;
const DAY = 24 * HOUR;

const startFormat = new Intl.DateTimeFormat('zh-CN', {month: 'long', day: 'numeric', hour: '2-digit', minute: '2-digit', hour12: false});

/** 赛季剩余时间, 如 "剩余 12 天" */
function remaining(endsAt: string) {
  const ms = new Date(endsAt).getTime() - Date.now();
  if (ms >= DAY) return `剩余 ${Math.floor(ms / DAY)} 天`;
  if (ms >= HOUR) return `剩余 ${Math.floor(ms / HOUR)} 小时`;
  return '即将结束';
}

const badgeClass = 'flex items-center gap-1.5 rounded-full border bg-card px-3 py-1 text-xs text-muted-foreground';

/**
 * 大厅标题旁的赛季信息: 进行中的赛季和剩余时间, 没有时显示下一个赛季的开始时间, 都没有时不显示
 * 进行中的赛季有奖励时可以点开查看
 */
export function SeasonBadge() {
  const {user} = useAuth();
  const [data, setData] = useState<CurrentSeasonData | null>(null);
  const [tiers, setTiers] = useState<Tier[]>([]);

  useEffect(() => {
    if (!user) return;
    services.season.getCurrent()
        .then((data) => {
          setData(data);
          // 段位只用于显示奖励的分数线, 失败时只显示分数
          if (data.rewards.length) return services.ranked.listTiers().then((t) => setTiers(t.tiers));
        })
        .catch(() => {
          // 只是提示信息, 失败时不显示
        });
  }, [user]);

  const text = data?.current ?
    `${data.current.name} · ${remaining(data.current.ends_at)}` :
    data?.next ? `${data.next.name} · ${startFormat.format(new Date(data.next.starts_at))} 开始` : null;
  if (!text) return null;

  if (!data?.current || !data.rewards.length) {
    return (
      <span className={badgeClass}>
        <CalendarRange className="size-3.5" />
        {text}
      </span>
    );
  }

  return (
    <Popover>
      <PopoverTrigger asChild>
        <button type="button" className={cn(badgeClass, 'cursor-pointer transition-colors hover:text-foreground')}>
          <CalendarRange className="size-3.5" />
          {text}
          <Gift className="size-3.5 text-primary" />
        </button>
      </PopoverTrigger>
      <PopoverContent align="end" className="w-80">
        <div className="flex flex-col gap-3">
          <div className="flex flex-col gap-0.5">
            <h3 className="text-sm font-medium">{data.current.name}奖励</h3>
            <p className="text-xs text-muted-foreground">赛季结束后按本赛季的最高段位分发放满足条件的最高一档, 需打过排位</p>
          </div>
          <ul className="flex flex-col gap-2">
            {[...data.rewards].reverse().map((r) => (
              <RewardRow key={r.min_rating} reward={r} tier={tierOf(tiers, r.min_rating)} />
            ))}
          </ul>
        </div>
      </PopoverContent>
    </Popover>
  );
}

/**
 * 一档奖励: 分数线和对应的段位, 下面列出货币和物品
 */
function RewardRow({reward: r, tier}: { reward: SeasonRewardView; tier: Tier | null }) {
  return (
    <li className="flex flex-col gap-1 rounded-lg border p-2">
      <div className="flex items-center gap-2 text-sm">
        {tier && <TierName tier={tier} />}
        <span className="text-muted-foreground tabular-nums">≥ {r.min_rating} 分</span>
      </div>
      <div className="flex flex-wrap gap-x-3 gap-y-1 text-sm">
        {r.lines.map((line) => (
          <span key={line.currency ?? `item:${line.item_id}`} className="flex items-center gap-1">
            {line.currency && <CurrencyIcon currency={line.currency} className="size-4" />}
            <span className="font-mono tabular-nums">
              {line.currency ? formatAmount(line.amount, line.decimals) : `×${line.amount}`}
            </span>
            <span className="text-muted-foreground">{line.name}</span>
          </span>
        ))}
      </div>
    </li>
  );
}
