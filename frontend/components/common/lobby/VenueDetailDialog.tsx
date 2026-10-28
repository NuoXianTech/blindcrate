'use client';

import {useEffect, useState, type ReactNode} from 'react';
import {cn} from 'cn';
import {Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle} from '@/components/ui/dialog';
import {Skeleton} from '@/components/ui/skeleton';
import {ItemImage} from '@/components/common/item';
import {CurrencyIcon} from '@/components/common/wallet';
import {BASIS_POINTS, MODES, formatRatio} from '@/lib/auction';
import {qualityOf} from '@/lib/quality';
import services from '@/lib/services';
import {toErrorMessage} from '@/lib/services/core';
import type {OddsEntry, SlotOdds, Venue, VenueDetail} from '@/lib/services/venue';
import {formatAmount} from '@/lib/utils';
import {EntryCost, RequirementList} from './VenueInfo';

/**
 * 地图详情: 拍卖规则、入场代价和条件、仓库价值区间, 以及仓库各档位的掉落概率
 * 列表中的地图先用来显示标题, 打开后拉取详情
 */
export function VenueDetailDialog({venue, decimals, onClose}: {
  /** 要查看的地图, null 时关闭 */
  venue: Venue | null;
  /** 出价货币的展示小数位 */
  decimals: number;
  onClose: () => void;
}) {
  return (
    <Dialog open={!!venue} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{venue?.name}</DialogTitle>
          <DialogDescription className="whitespace-pre-wrap">{venue?.description || '地图的规则和掉落概率'}</DialogDescription>
        </DialogHeader>
        {/* 每次打开都重新挂载, 拉取最新的条件进度和概率 */}
        {venue && <DetailContent key={venue.code} code={venue.code} decimals={decimals} />}
      </DialogContent>
    </Dialog>
  );
}

/**
 * 详情内容
 */
function DetailContent({code, decimals}: { code: string; decimals: number }) {
  const [detail, setDetail] = useState<VenueDetail | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    services.venue.getVenue(code)
        .then(setDetail)
        .catch((err) => setError(toErrorMessage(err)));
  }, [code]);

  if (error) return <p className="py-8 text-center text-sm text-muted-foreground">{error}</p>;
  if (!detail) {
    return (
      <div className="flex flex-col gap-4">
        <Skeleton className="h-32 rounded-lg" />
        <Skeleton className="h-48 rounded-lg" />
      </div>
    );
  }

  const v = detail;
  const ranges = [
    {kind: 'pve', enabled: v.pve_enabled, range: v.pve.value_range},
    {kind: 'ranked', enabled: v.ranked_enabled, range: v.value_range},
  ] as const;

  return (
    <div className="flex flex-col gap-6">
      <Section title="拍卖规则">
        <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1.5 text-sm">
          <dt className="text-muted-foreground">起拍价</dt>
          <dd className="flex items-center gap-1 font-mono tabular-nums">
            <CurrencyIcon currency={v.bid_currency} className="size-4" />
            {formatAmount(v.auction.start_price, decimals)}
          </dd>
          <dt className="text-muted-foreground">带入资金</dt>
          <dd>
            <span className="flex items-center gap-1 font-mono tabular-nums">
              <CurrencyIcon currency={v.bid_currency} className="size-4" />
              {formatAmount(v.auction.max_carry, decimals)}
            </span>
            <p className="text-xs text-muted-foreground">开局时冻结可用余额, 最多冻结这么多, 出价不能超过冻结的金额</p>
          </dd>
          <dt className="text-muted-foreground">轮数</dt>
          <dd>
            {v.auction.rounds} 轮, 每轮 {v.auction.round_seconds} 秒
          </dd>
          <dt className="text-muted-foreground">速胜</dt>
          <dd>
            {v.auction.quick_win.length ? (
              <>
                {v.auction.quick_win.map((x, i) => `第 ${i + 1} 轮 ${formatRatio(x)} 倍`).join('、')}
                <p className="text-xs text-muted-foreground">第一名的出价不低于第二名的这么多倍时直接成交</p>
              </>
            ) : (
              '无'
            )}
          </dd>
          <dt className="text-muted-foreground">加轮</dt>
          <dd>
            {v.auction.max_extra_rounds > 0 ?
              `最后一轮最高价并列时加轮, 最多 ${v.auction.max_extra_rounds} 轮, 之后按先出价的人成交` :
              '最后一轮最高价并列时按先出价的人成交'}
          </dd>
          <dt className="text-muted-foreground">仓库</dt>
          <dd>
            {v.grid.width} × {v.grid.height} 格
          </dd>
        </dl>
      </Section>

      <Section title="入场">
        <div className="flex flex-col gap-2 text-sm">
          <div className="flex items-start gap-4">
            <span className="shrink-0 text-muted-foreground">入场代价</span>
            <EntryCost venue={v} decimals={decimals} />
          </div>
          {v.requirements.length > 0 ? (
            <RequirementList venue={v} />
          ) : (
            <span className="text-muted-foreground">没有入场条件</span>
          )}
        </div>
      </Section>

      <Section title="仓库价值" hint="模拟的仓库价值, 八成的仓库在低和高之间">
        <div className="grid grid-cols-4 gap-x-4 gap-y-1.5 text-sm">
          <span className="text-muted-foreground">模式</span>
          <span className="text-right text-muted-foreground">低</span>
          <span className="text-right text-muted-foreground">中位</span>
          <span className="text-right text-muted-foreground">高</span>
          {ranges.filter((r) => r.enabled).map(({kind, range}) => (
            <ValueRow key={kind} label={MODES[kind].label} values={[range.p10, range.p50, range.p90]} decimals={decimals} />
          ))}
        </div>
      </Section>

      <Section title="掉落概率" hint="仓库由各档位抽到的藏品组成, 概率为每次抽取抽中该藏品的概率">
        <div className="flex flex-col gap-3">
          {v.slots.map((slot, i) => (
            <SlotCard key={i} index={i} slot={slot} />
          ))}
        </div>
      </Section>
    </div>
  );
}

/**
 * 详情中的一节
 */
function Section({title, hint, children}: { title: string; hint?: string; children: ReactNode }) {
  return (
    <section className="flex flex-col gap-2">
      <div className="flex flex-col">
        <h3 className="font-bold">{title}</h3>
        {hint && <p className="text-xs text-muted-foreground">{hint}</p>}
      </div>
      {children}
    </section>
  );
}

/**
 * 仓库价值区间的一行
 */
function ValueRow({label, values, decimals}: { label: string; values: number[]; decimals: number }) {
  return (
    <>
      <span>{label}</span>
      {values.map((x, i) => (
        <span key={i} className="text-right font-mono tabular-nums">
          {formatAmount(x, decimals)}
        </span>
      ))}
    </>
  );
}

/** 同一品质的条目 */
interface QualityGroup {
  quality: number;
  entries: OddsEntry[];
  /** 抽中这一品质的概率 */
  probability: number;
}

/** 条目按品质分组, 稀有的在前, 组内按概率从高到低 */
function groupByQuality(entries: OddsEntry[]): QualityGroup[] {
  const groups = new Map<number, QualityGroup>();
  for (const e of entries) {
    const group = groups.get(e.quality) ?? {quality: e.quality, entries: [], probability: 0};
    group.entries.push(e);
    group.probability += e.probability;
    groups.set(e.quality, group);
  }
  const list = [...groups.values()].sort((a, b) => b.quality - a.quality);
  for (const g of list) g.entries.sort((a, b) => b.probability - a.probability);
  return list;
}

/** 概率格式化为百分比, 很小的概率保留更多位数 */
function formatPercent(p: number): string {
  const percent = p * 100;
  if (percent > 0 && percent < 0.01) return '<0.01%';
  return `${Number(percent.toFixed(percent < 1 ? 2 : 1))}%`;
}

/** 次数或数量的范围, 如 "2-3" */
function formatRange(min: number, max: number): string {
  return min === max ? String(min) : `${min}-${max}`;
}

/**
 * 仓库的一个档位: 出现概率、抽取次数, 以及掉落池按品质分组的概率表
 */
function SlotCard({index, slot}: { index: number; slot: SlotOdds }) {
  const groups = slot.odds ? groupByQuality(slot.odds.entries) : [];
  const total = groups.reduce((sum, g) => sum + g.probability, 0);

  return (
    <div className="flex flex-col rounded-lg border">
      <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-1 border-b px-3 py-2 text-sm">
        <span className="font-medium">
          第 {index + 1} 档 · {slot.odds?.name ?? slot.pool}
        </span>
        <span className="text-muted-foreground">
          {slot.chance >= BASIS_POINTS ? '必定出现' : `${formatPercent(slot.chance / BASIS_POINTS)} 出现`}, 抽 {formatRange(slot.min, slot.max)} 次
        </span>
      </div>
      {!slot.odds ? (
        <p className="px-3 py-4 text-center text-sm text-muted-foreground">掉落池暂不可用, 地图暂时无法开局</p>
      ) : (
        <div className="flex flex-col divide-y">
          {groups.map((g) => {
            const q = qualityOf(g.quality);
            return (
              <div key={g.quality} className="flex flex-col gap-1 px-3 py-2">
                <div className="flex items-center justify-between gap-4 text-sm">
                  <span className="flex items-center gap-2 font-medium">
                    <span className={cn('h-3 w-1 rounded-full', q.bar)} />
                    <span className={q.text}>{g.quality > 0 ? q.label : '其他'}</span>
                  </span>
                  <span className="font-mono font-medium tabular-nums">{formatPercent(g.probability)}</span>
                </div>
                <ul className="flex flex-col gap-1">
                  {g.entries.map((e) => (
                    <li key={e.id} className="flex items-center gap-2 text-sm">
                      <span className="flex size-7 shrink-0 items-center justify-center overflow-hidden rounded border bg-card p-0.5">
                        <ItemImage item={e} />
                      </span>
                      <span className="min-w-0 flex-1 truncate" title={e.name}>
                        {e.name}
                        <span className="ml-1.5 font-mono text-xs text-muted-foreground tabular-nums">
                          ×{formatRange(e.min_quantity, e.max_quantity)}
                        </span>
                      </span>
                      <span className="font-mono text-muted-foreground tabular-nums">{formatPercent(e.probability)}</span>
                    </li>
                  ))}
                </ul>
              </div>
            );
          })}
          <div className="flex items-center justify-between gap-4 px-3 py-2 text-sm text-muted-foreground">
            <span>合计</span>
            <span className="font-mono tabular-nums">{formatPercent(total)}</span>
          </div>
        </div>
      )}
    </div>
  );
}
