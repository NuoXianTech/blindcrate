'use client';

import {useEffect, useState} from 'react';
import {toast} from 'sonner';
import {cn} from 'cn';
import {Select, SelectContent, SelectGroup, SelectItem, SelectLabel, SelectTrigger, SelectValue} from '@/components/ui/select';
import {Skeleton} from '@/components/ui/skeleton';
import {useAuth} from '@/components/common/auth';
import {CurrencyIcon} from '@/components/common/wallet';
import {MODES, type ModeKind} from '@/lib/auction';
import services from '@/lib/services';
import {toErrorMessage} from '@/lib/services/core';
import {PROFIT_METRIC_PREFIX, STATS_METRIC, STATS_SCOPE_ALL, type UserStats} from '@/lib/services/stats';
import type {Venue} from '@/lib/services/venue';
import type {BalanceItem} from '@/lib/services/wallet';
import {formatAmount} from '@/lib/utils';
import {DailyChart} from './DailyChart';

/** 单独展示为比率的指标, 不再作为普通指标重复显示 */
const RATE_METRICS: string[] = [STATS_METRIC.matches, STATS_METRIC.wins, STATS_METRIC.bidWon];

/** 比率, 如 "33.3%"; 没有对局时为 "-" */
function formatRate(count: number, matches: number) {
  return matches > 0 ? `${Number(((count / matches) * 100).toFixed(1))}%` : '-';
}

/**
 * 个人数据: 按统计范围显示对局数、胜率、成交率、各货币净收益和其他计数指标, 以及最近 7 天的对局数和净收益
 * 练习局不计入
 */
export function StatsMain() {
  const {user} = useAuth();
  const [scope, setScope] = useState(STATS_SCOPE_ALL);
  const [data, setData] = useState<UserStats | null>(null);
  const [venues, setVenues] = useState<Venue[]>([]);
  const [currencies, setCurrencies] = useState<BalanceItem[]>([]);

  useEffect(() => {
    if (!user) return;
    services.venue.listVenues()
        .then((res) => setVenues(res.venues))
        .catch(() => {
          // 只影响地图范围的选项
        });
    services.wallet.listBalances()
        .then((res) => setCurrencies(res.balances))
        .catch(() => {
          // 只影响金额的小数位和货币名称
        });
  }, [user]);

  useEffect(() => {
    if (!user) return;
    let cancelled = false;
    services.stats.getStats(scope)
        .then((res) => {
          if (!cancelled) setData(res);
        })
        .catch((err) => toast.error(toErrorMessage(err)));
    return () => {
      cancelled = true;
    };
  }, [scope, user]);

  // 切换范围后不再显示上一个范围的数据
  const current = data?.scope === scope ? data : null;
  const currencyOf = (code: string) => currencies.find((c) => c.currency === code);

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h1 className="text-2xl font-bold">个人数据</h1>
        <Select value={scope} onValueChange={setScope}>
          <SelectTrigger className="w-48" aria-label="统计范围">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={STATS_SCOPE_ALL}>全部对局</SelectItem>
            {(Object.keys(MODES) as ModeKind[]).map((k) => (
              <SelectItem key={k} value={`mode:${k}`}>
                {MODES[k].label}
              </SelectItem>
            ))}
            {venues.length > 0 && (
              <SelectGroup>
                <SelectLabel>地图</SelectLabel>
                {venues.map((v) => (
                  <SelectItem key={v.code} value={`venue:${v.code}`}>
                    {v.name}
                  </SelectItem>
                ))}
              </SelectGroup>
            )}
          </SelectContent>
        </Select>
      </div>
      <p className="-mt-2 text-xs text-muted-foreground">练习局不计入; 净收益按货币分开统计</p>

      {!current ? (
        <div className="flex flex-col gap-4">
          <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
            {Array.from({length: 4}, (_, i) => (
              <Skeleton key={i} className="h-24 rounded-xl" />
            ))}
          </div>
          <Skeleton className="h-60 rounded-xl" />
        </div>
      ) : (
        <StatsContent data={current} currencyOf={currencyOf} />
      )}
    </div>
  );
}

/**
 * 一个统计范围的数据
 */
function StatsContent({data, currencyOf}: { data: UserStats; currencyOf: (code: string) => BalanceItem | undefined }) {
  const total = (metric: string) => data.total[metric] ?? 0;
  const matches = total(STATS_METRIC.matches);
  // 累计或最近几天出现过的净收益货币
  const profitCurrencies = [...new Set(
      [data.total, ...data.days.map((d) => d.values)]
          .flatMap((values) => Object.keys(values))
          .filter((key) => key.startsWith(PROFIT_METRIC_PREFIX))
          .map((key) => key.slice(PROFIT_METRIC_PREFIX.length)),
  )].sort();
  const others = data.metrics.filter((m) => !RATE_METRICS.includes(m.key));

  return (
    <div className="flex flex-col gap-4">
      <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
        <StatTile label="对局数" value={formatAmount(matches)} />
        <StatTile label="胜率" value={formatRate(total(STATS_METRIC.wins), matches)} hint={`第一名 ${total(STATS_METRIC.wins)} 次`} />
        <StatTile label="成交率" value={formatRate(total(STATS_METRIC.bidWon), matches)} hint={`拍下 ${total(STATS_METRIC.bidWon)} 次`} />
        {profitCurrencies.map((code) => {
          const profit = total(PROFIT_METRIC_PREFIX + code);
          return (
            <StatTile
              key={code}
              label={`${currencyOf(code)?.name ?? code}净收益`}
              icon={<CurrencyIcon currency={code} className="size-4" />}
              value={formatAmount(profit, currencyOf(code)?.decimals ?? 0, {signed: true})}
              className={cn(profit > 0 && 'text-quality-green', profit < 0 && 'text-quality-red')}
            />
          );
        })}
        {others.map((m) => (
          <StatTile key={m.key} label={m.label} value={formatAmount(total(m.key))} />
        ))}
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <DailyChart
          title="最近 7 天对局数"
          points={data.days.map((d) => ({day: d.day, value: d.values[STATS_METRIC.matches] ?? 0}))}
          format={(v) => `${formatAmount(v)} 局`}
        />
        {profitCurrencies.map((code) => {
          const currency = currencyOf(code);
          return (
            <DailyChart
              key={code}
              title={`最近 7 天${currency?.name ?? code}净收益`}
              points={data.days.map((d) => ({day: d.day, value: d.values[PROFIT_METRIC_PREFIX + code] ?? 0}))}
              format={(v) => formatAmount(v, currency?.decimals ?? 0, {signed: true})}
              diverging
            />
          );
        })}
      </div>
    </div>
  );
}

/**
 * 一个数据块
 */
function StatTile({label, value, hint, icon, className}: {
  label: string;
  value: string;
  hint?: string;
  icon?: React.ReactNode;
  className?: string;
}) {
  return (
    <div className="flex flex-col gap-1 rounded-xl border bg-card p-4">
      <span className="flex items-center gap-1.5 text-xs text-muted-foreground">
        {icon}
        {label}
      </span>
      <span className={cn('font-mono text-2xl font-bold tabular-nums', className)}>{value}</span>
      {hint && <span className="text-xs text-muted-foreground">{hint}</span>}
    </div>
  );
}
