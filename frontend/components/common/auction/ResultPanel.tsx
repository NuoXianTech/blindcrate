'use client';

import {useEffect, useState} from 'react';
import Link from 'next/link';
import {useRouter} from 'next/navigation';
import {LoaderCircle, RotateCcw} from 'lucide-react';
import {toast} from 'sonner';
import {cn} from 'cn';
import {Button} from '@/components/ui/button';
import {TierName} from '@/components/common/ranked';
import {CurrencyIcon} from '@/components/common/wallet';
import {ROUTES, matchHref} from '@/lib/access';
import {OUTCOMES, parseMode} from '@/lib/auction';
import {tierOf} from '@/lib/ranked';
import services from '@/lib/services';
import {AUCTION_OUTCOME, type AuctionView} from '@/lib/services/auction';
import {toErrorMessage} from '@/lib/services/core';
import type {MatchPlayer} from '@/lib/services/match';
import type {Tier} from '@/lib/services/ranked';
import {formatAmount} from '@/lib/utils';

/**
 * 结算结果: 成交价、仓库价值、自己的盈亏和排位的段位分、段位变化; 结算后对局随即释放, 开箱时就可以离开或再来一局
 */
export function ResultPanel({view, players, decimals, finished, released}: {
  view: AuctionView;
  players: MatchPlayer[];
  decimals: number;
  /** 开箱阶段结束, 对局已结束 */
  finished: boolean;
  /** 对局已释放, 可以离开去排下一局 */
  released: boolean;
}) {
  const router = useRouter();
  const [joining, setJoining] = useState(false);
  const [tiers, setTiers] = useState<Tier[] | null>(null);
  const result = view.result;
  const sold = result?.outcome === AUCTION_OUTCOME.sold;
  // 排位成交后按名次加减段位分, 流拍和练习局不计分
  const rated = sold && parseMode(view.mode)?.kind === 'ranked' && !view.self.is_practice;

  useEffect(() => {
    if (!rated) return;
    services.ranked.listTiers()
        .then((data) => setTiers(data.tiers))
        .catch(() => {
          // 只影响段位的展示, 失败时只显示段位分
        });
  }, [rated]);

  if (!result) return null;

  const winner = players.find((p) => p.seat === result.winner_seat);
  // 结算前后的段位, 由段位分换算; 段位按当前的分数线计算
  const tierBefore = tiers && tierOf(tiers, result.rating - result.rating_delta);
  const tierAfter = tiers && tierOf(tiers, result.rating);
  const tierChanged = !!tiers && tierBefore?.id !== tierAfter?.id;

  let title: string;
  if (sold && winner?.is_self) {
    title = '你拍下了这个仓库';
  } else if (sold) {
    title = `${winner?.nickname ?? `${result.winner_seat} 号位`} 拍下了这个仓库`;
  } else if (result.outcome === AUCTION_OUTCOME.unsold) {
    title = '流拍, 入场代价已退还';
  } else {
    title = '对局已取消, 入场代价已退还';
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
    {label: '成交价', value: sold ? formatAmount(result.price, decimals) : '-'},
    {label: '仓库价值', value: formatAmount(result.value, decimals)},
  ];

  return (
    <div className="flex flex-col gap-4 rounded-xl border bg-card p-4">
      <div className="flex flex-col gap-1">
        <span className="text-xs text-muted-foreground">{OUTCOMES[result.outcome] ?? result.outcome}</span>
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

      <div className="flex flex-col gap-0.5">
        <span className="text-xs text-muted-foreground">
          我的盈亏{view.self.is_practice && ' (练习局, 仅供参考)'}
        </span>
        <span
          className={cn(
              'font-mono text-3xl font-bold tabular-nums',
              result.profit > 0 && 'text-quality-green',
              result.profit < 0 && 'text-quality-red',
          )}
        >
          {formatAmount(result.profit, decimals, {signed: true})}
        </span>
      </div>

      {rated && (
        <div className="flex flex-col gap-0.5">
          <span className="text-xs text-muted-foreground">段位分</span>
          <span className="flex items-baseline gap-2 font-mono tabular-nums">
            {tiers && <TierName tier={tierAfter} className="self-center font-sans" />}
            <span className="text-xl font-bold">{result.rating}</span>
            <span
              className={cn(
                  'text-sm',
                  result.rating_delta > 0 && 'text-quality-green',
                  result.rating_delta < 0 && 'text-quality-red',
              )}
            >
              {formatAmount(result.rating_delta, 0, {signed: true})}
            </span>
          </span>
          {tierChanged && (
            <span className="flex items-center gap-1.5 text-xs text-muted-foreground">
              {result.rating_delta > 0 ? '升段' : '降段'}
              <TierName tier={tierBefore} />
              →
              <TierName tier={tierAfter} />
            </span>
          )}
        </div>
      )}

      {released ? (
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
