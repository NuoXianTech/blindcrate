import {Bot, Info, LoaderCircle, Swords} from 'lucide-react';
import {cn} from 'cn';
import {Badge} from '@/components/ui/badge';
import {Button} from '@/components/ui/button';
import {CurrencyIcon} from '@/components/common/wallet';
import {MODES, type ModeKind} from '@/lib/auction';
import type {ProfitCap, Venue, ValueRange} from '@/lib/services/venue';
import {formatAmount} from '@/lib/utils';
import {EntryCost, RequirementList} from './VenueInfo';

/**
 * 大厅里的一张地图: 简介、入场代价、仓库价值区间、入场条件, 以及人机和排位两个入口
 * busy 为正在加入的模式, queued 为正在这张地图排队的模式
 */
export function VenueCard({venue: v, decimals, busy, queued, disabled, onJoin, onDetail}: {
  venue: Venue;
  /** 出价货币的展示小数位 */
  decimals: number;
  busy: ModeKind | null;
  queued: ModeKind | null;
  /** 正在排队或对局中时不能再加入 */
  disabled: boolean;
  onJoin: (kind: ModeKind) => void;
  /** 查看规则和掉落概率 */
  onDetail: () => void;
}) {
  const modes: { kind: ModeKind; enabled: boolean; range: ValueRange; icon: typeof Bot }[] = [
    {kind: 'pve', enabled: v.pve_enabled, range: v.pve.value_range, icon: Bot},
    {kind: 'ranked', enabled: v.ranked_enabled, range: v.value_range, icon: Swords},
  ];
  const capReached = v.pve.profit_caps.some((c) => c.profit >= c.limit);

  return (
    <div className="flex flex-col overflow-hidden rounded-xl border bg-card">
      {v.image_url && (
        // 图片地址由管理员填写, 域名不固定, 不走 next/image
        // eslint-disable-next-line @next/next/no-img-element
        <img src={v.image_url} alt={v.name} className="h-32 w-full object-cover" />
      )}
      <div className="flex flex-1 flex-col gap-4 p-4">
        <div className="flex flex-col gap-1">
          <div className="flex items-center justify-between gap-2">
            <h2 className="text-lg font-bold">{v.name}</h2>
            <Button variant="ghost" size="sm" className="shrink-0" onClick={onDetail}>
              <Info />
              详情
            </Button>
          </div>
          {v.description && <p className="text-sm whitespace-pre-wrap text-muted-foreground">{v.description}</p>}
        </div>

        <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1.5 text-sm">
          <dt className="text-muted-foreground">入场代价</dt>
          <dd>
            <EntryCost venue={v} decimals={decimals} />
          </dd>
          <dt className="text-muted-foreground">起拍 / 带入</dt>
          <dd className="flex items-center gap-1 font-mono tabular-nums">
            <CurrencyIcon currency={v.bid_currency} className="size-4" />
            {formatAmount(v.auction.start_price, decimals)} / {formatAmount(v.auction.max_carry, decimals)}
          </dd>
          <dt className="text-muted-foreground">轮数</dt>
          <dd>
            {v.auction.rounds} 轮, 每轮 {v.auction.round_seconds} 秒
          </dd>
        </dl>

        {v.requirements.length > 0 && <RequirementList venue={v} />}

        <div className="mt-auto grid gap-2 sm:grid-cols-2">
          {modes.map(({kind, enabled, range, icon: Icon}) => (
            <div key={kind} className="flex flex-col gap-1.5 rounded-lg border p-3">
              <div className="flex items-center justify-between gap-2">
                <span className="flex items-center gap-1.5 font-medium whitespace-nowrap">
                  <Icon className="size-4 text-primary" />
                  {MODES[kind].label}
                </span>
                {kind === 'pve' && (v.pve.practice_only || capReached) && (
                  <Badge variant="outline" title={v.pve.practice_only ? undefined : '今日人机收益已达上限, 明天恢复'}>
                    只能练习
                  </Badge>
                )}
              </div>
              <span className="text-xs text-muted-foreground" title="模拟的仓库价值, 八成的仓库在这个区间内">
                价值约 {formatAmount(range.p10, decimals, {compact: true})} - {formatAmount(range.p90, decimals, {compact: true})}
              </span>
              {kind === 'pve' && v.pve.profit_caps.map((c) => (
                <ProfitCapLine key={c.scope} cap={c} currency={v.bid_currency} decimals={decimals} />
              ))}
              <Button
                size="sm"
                variant={kind === 'pve' ? 'default' : 'secondary'}
                disabled={!enabled || !v.eligible || disabled || busy !== null}
                onClick={() => onJoin(kind)}
              >
                {busy === kind && <LoaderCircle className="animate-spin" />}
                {!enabled ? '未开放' : !v.eligible ? '未满足条件' : queued === kind ? '匹配中' : '开始'}
              </Button>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

/**
 * 人机模式的一个每日收益上限, 如 "今日收益 120 / 5,000"; 达到后显示为警示色
 */
function ProfitCapLine({cap, currency, decimals}: { cap: ProfitCap; currency: string; decimals: number }) {
  const reached = cap.profit >= cap.limit;
  const global = !cap.scope.startsWith('venue:');
  return (
    <span
      className={cn('flex items-center justify-between gap-2 text-xs whitespace-nowrap', reached ? 'text-quality-red' : 'text-muted-foreground')}
      title={`${global ? '全部地图' : '这张地图'}今天的人机净收益 / 每日上限; ${reached ? '已达上限, 今天的人机对局都是练习局' : '达到上限后, 今天的人机对局转为练习局'}`}
    >
      {global ? '今日收益' : '本图今日'}
      <span className="flex items-center gap-1 font-mono tabular-nums">
        <CurrencyIcon currency={currency} className="size-3.5" />
        {formatAmount(cap.profit, decimals, {signed: true})} / {formatAmount(cap.limit, decimals, {compact: true})}
      </span>
    </span>
  );
}
