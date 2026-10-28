'use client';

import {useState} from 'react';
import Link from 'next/link';
import {useRouter} from 'next/navigation';
import {LoaderCircle, RotateCcw} from 'lucide-react';
import {toast} from 'sonner';
import {cn} from 'cn';
import {Button} from '@/components/ui/button';
import {CurrencyIcon} from '@/components/common/wallet';
import {ROUTES, matchHref} from '@/lib/access';
import {OUTCOMES} from '@/lib/auction';
import services from '@/lib/services';
import {AUCTION_OUTCOME, type AuctionView} from '@/lib/services/auction';
import {toErrorMessage} from '@/lib/services/core';
import type {MatchPlayer} from '@/lib/services/match';
import {formatAmount} from '@/lib/utils';

/**
 * 结算结果: 成交价、仓库价值和自己的盈亏; 对局结束后可以返回大厅或再来一局
 */
export function ResultPanel({view, players, decimals, finished}: {
  view: AuctionView;
  players: MatchPlayer[];
  decimals: number;
  /** 开箱阶段结束, 对局已结束, 这时才能再来一局 */
  finished: boolean;
}) {
  const router = useRouter();
  const [joining, setJoining] = useState(false);
  const result = view.result;
  if (!result) return null;

  const winner = players.find((p) => p.seat === result.winner_seat);
  const sold = result.outcome === AUCTION_OUTCOME.sold;

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

      {finished ? (
        <div className="grid grid-cols-2 gap-2">
          <Button variant="outline" asChild>
            <Link href={ROUTES.gameHome}>返回大厅</Link>
          </Button>
          <Button disabled={joining} onClick={again}>
            {joining ? <LoaderCircle className="animate-spin" /> : <RotateCcw />}
            再来一局
          </Button>
        </div>
      ) : (
        <span className="text-center text-xs text-muted-foreground">开箱中, 稍后可以再来一局</span>
      )}
    </div>
  );
}
