'use client';

import {useEffect, useState} from 'react';
import Link from 'next/link';
import {useRouter} from 'next/navigation';
import {LoaderCircle, RotateCcw} from 'lucide-react';
import {toast} from 'sonner';
import {cn} from 'cn';
import {Button} from '@/components/ui/button';
import {TierName} from '@/components/common/ranked';
import {PriceText} from '@/components/common/reward';
import {CurrencyIcon} from '@/components/common/wallet';
import type {Scan} from '@/hooks/use-scan';
import {ROUTES, matchHref} from '@/lib/access';
import {OUTCOMES, parseMode} from '@/lib/auction';
import {tierOf} from '@/lib/ranked';
import services from '@/lib/services';
import {AUCTION_OUTCOME, type AuctionView, type LotView} from '@/lib/services/auction';
import {toErrorMessage} from '@/lib/services/core';
import type {MatchPlayer} from '@/lib/services/match';
import type {Tier} from '@/lib/services/ranked';
import {formatAmount} from '@/lib/utils';

/**
 * 当前拍品的结算结果: 成交价、仓库价值; 最后一个拍品开箱后还有整场的名次、自己扣掉入场费的盈亏和排位的段位分、段位变化,
 * 此时拍场已结算, 对局随即释放, 开箱时就可以离开或再来一局
 * 仓库价值随开箱扫描累加, 盈亏、名次和段位分能推出仓库价值, 扫完才公开
 */
export function ResultPanel({view, lot, players, decimals, scan, finished, released}: {
  view: AuctionView;
  /** 当前拍品 */
  lot: LotView;
  players: MatchPlayer[];
  decimals: number;
  /** 开箱扫描的进度, 见 useScan */
  scan: Scan;
  /** 开箱阶段结束, 对局已结束 */
  finished: boolean;
  /** 对局已释放, 可以离开去排下一局 */
  released: boolean;
}) {
  const router = useRouter();
  const [joining, setJoining] = useState(false);
  const [tiers, setTiers] = useState<Tier[] | null>(null);
  const {session, self} = view;
  const sold = lot.outcome === AUCTION_OUTCOME.sold;
  // 排位至少成交一个拍品时按名次加减段位分, 全部流拍和练习局不计分
  const rated = session?.outcome === AUCTION_OUTCOME.sold && parseMode(view.mode)?.kind === 'ranked' && !self.is_practice;

  useEffect(() => {
    if (!rated) return;
    services.ranked.listTiers()
        .then((data) => setTiers(data.tiers))
        .catch(() => {
          // 只影响段位的展示, 失败时只显示段位分
        });
  }, [rated]);

  const winner = players.find((p) => p.seat === lot.winner_seat);
  // 结算前后的段位, 由段位分换算; 段位按当前的分数线计算
  const tierBefore = tiers && tierOf(tiers, self.rating - self.rating_delta);
  const tierAfter = tiers && tierOf(tiers, self.rating);
  const tierChanged = !!tiers && tierBefore?.id !== tierAfter?.id;

  // 入场费只有一个拍品都没成交时才退
  const refunded = !view.lots.some((l) => l.outcome === AUCTION_OUTCOME.sold);
  let title: string;
  if (sold && winner?.is_self) {
    title = '你拍下了这个仓库';
  } else if (sold) {
    title = `${winner?.nickname ?? `${lot.winner_seat} 号位`} 拍下了这个仓库`;
  } else if (lot.outcome === AUCTION_OUTCOME.unsold) {
    title = session && refunded ? '全部流拍, 入场代价已退还' : '这个拍品流拍';
  } else {
    title = refunded ? '对局已取消, 入场代价已退还' : '对局已取消, 已成交的拍品不受影响';
  }

  const again = async () => {
    setJoining(true);
    try {
      const status = await services.match.joinMatchmaking(view.mode);
      router.push(status.match_id ? matchHref(status.match_id) : ROUTES.gameHome);
    } catch (err) {
      toast.error(toErrorMessage(err));
      setJoining(false);
    }
  };

  const rows = [
    {label: '成交价', value: sold ? formatAmount(lot.price, decimals) : '-'},
    {label: scan.complete ? '仓库价值' : '仓库价值 (扫描中)', value: formatAmount(scan.complete ? lot.value : scan.value, decimals)},
    {label: '我的剩余资金', value: formatAmount(self.remaining, decimals)},
  ];
  // 整场盈亏要扣掉入场费; 其余的入场代价不是出价货币, 单独列出
  const net = self.profit - self.entry_fee;
  // 全部流拍时名次为 0, 按座位号
  const standings = session ? [...session.seats].sort((a, b) => a.rank - b.rank || a.seat - b.seat) : [];

  return (
    <div className="flex flex-col gap-4 rounded-xl border bg-card p-4">
      <div className="flex flex-col gap-1">
        <span className="text-xs text-muted-foreground">拍品 {lot.lot_no}/{view.lot_count} · {OUTCOMES[lot.outcome] ?? lot.outcome}</span>
        <span className="text-lg font-bold">{title}</span>
      </div>

      <dl className="grid grid-cols-2 gap-3">
        {rows.map((row) => (
          <div key={row.label} className="flex flex-col gap-0.5">
            <dt className="text-xs text-muted-foreground">{row.label}</dt>
            <dd className="flex items-center gap-1 font-mono tabular-nums">
              <CurrencyIcon currency={view.venue.bid_currency} className="size-4" />
              {row.value}
            </dd>
          </div>
        ))}
      </dl>

      {session && !scan.complete && (
        <div className="flex flex-col gap-0.5">
          <span className="text-xs text-muted-foreground">我的盈亏</span>
          <span className="flex items-center gap-2 text-sm text-muted-foreground">
            <LoaderCircle className="size-4 animate-spin" />
            开箱扫描中, 扫完揭晓
          </span>
        </div>
      )}

      {session && scan.complete && (
        <div className="flex flex-col gap-0.5">
          <span className="text-xs text-muted-foreground">
            整场盈亏{self.is_practice && ' (练习局, 仅供参考)'}
          </span>
          <span
            className={cn(
                'font-mono text-3xl font-bold tabular-nums',
                net > 0 && 'text-quality-green',
                net < 0 && 'text-quality-red',
            )}
          >
            {formatAmount(net, decimals, {signed: true})}
          </span>
          {self.entry_fee > 0 && (
            <span className="font-mono text-xs tabular-nums text-muted-foreground">
              拍品 {formatAmount(self.profit, decimals, {signed: true})} · 入场费 {formatAmount(-self.entry_fee, decimals, {signed: true})}
            </span>
          )}
          {self.entry_cost.length > 0 && (
            <span className="flex flex-wrap items-center gap-x-2 text-xs text-muted-foreground">
              另付入场代价
              <PriceText lines={self.entry_cost} />
            </span>
          )}
        </div>
      )}

      {session && scan.complete && (
        <ul className="flex flex-col gap-1 text-sm">
          {standings.map((s) => {
            const player = players.find((p) => p.seat === s.seat);
            return (
              <li key={s.seat} className={cn('flex items-center gap-2', player?.is_self && 'font-medium')}>
                <span className="w-6 font-mono tabular-nums text-muted-foreground">{s.rank > 0 ? `#${s.rank}` : '-'}</span>
                <span className="min-w-0 flex-1 truncate">{player?.nickname ?? `${s.seat} 号位`}</span>
                <span className="font-mono text-xs tabular-nums text-muted-foreground" title="花费 / 带入, 右侧为拍品的盈亏, 不含入场费">
                  {formatAmount(s.spent, decimals)} / {formatAmount(s.carry, decimals)}
                </span>
                <span
                  className={cn('w-20 text-right font-mono tabular-nums', s.profit > 0 && 'text-quality-green', s.profit < 0 && 'text-quality-red')}
                >
                  {formatAmount(s.profit, decimals, {signed: true})}
                </span>
              </li>
            );
          })}
        </ul>
      )}

      {rated && scan.complete && (
        <div className="flex flex-col gap-0.5">
          <span className="text-xs text-muted-foreground">段位分</span>
          <span className="flex items-baseline gap-2 font-mono tabular-nums">
            {tiers && <TierName tier={tierAfter} className="self-center font-sans" />}
            <span className="text-xl font-bold">{self.rating}</span>
            <span
              className={cn(
                  'text-sm',
                  self.rating_delta > 0 && 'text-quality-green',
                  self.rating_delta < 0 && 'text-quality-red',
              )}
            >
              {formatAmount(self.rating_delta, 0, {signed: true})}
            </span>
          </span>
          {tierChanged && (
            <span className="flex items-center gap-1.5 text-xs text-muted-foreground">
              {self.rating_delta > 0 ? '升段' : '降段'}
              <TierName tier={tierBefore} />
              →
              <TierName tier={tierAfter} />
            </span>
          )}
        </div>
      )}

      {!session ? (
        lot.lot_no === view.lot && (
          <span className="text-center text-xs text-muted-foreground">开箱后进入下一个拍品, 带入资金继续沿用</span>
        )
      ) : released ? (
        <div className="flex flex-col gap-2">
          <div className="grid grid-cols-2 gap-2">
            <Button variant="outline" asChild>
              <Link href={ROUTES.gameHome}>{finished ? '返回大厅' : '离开'}</Link>
            </Button>
            <Button disabled={joining} onClick={again}>
              {joining ? <LoaderCircle className="animate-spin" /> : <RotateCcw />}
              再来一局
            </Button>
          </div>
          {!finished && (
            <span className="text-center text-xs text-muted-foreground">已结算, 可以留下看完开箱, 也可以直接离开</span>
          )}
        </div>
      ) : (
        <span className="text-center text-xs text-muted-foreground">开箱中, 稍后可以再来一局</span>
      )}
    </div>
  );
}
