import {Crown} from 'lucide-react';
import {cn} from 'cn';
import {Badge} from '@/components/ui/badge';
import {UserAvatar} from '@/components/common/user';
import type {AuctionView} from '@/lib/services/auction';
import type {MatchPlayer} from '@/lib/services/match';
import {formatAmount} from '@/lib/utils';

/**
 * 四个座位: 加轮时标出可以出价的人; 开箱后公开每个人的出价和成交者, 名次和分红能推出仓库价值, 扫描完才公开
 */
export function SeatList({view, players, decimals, revealed}: {
  view: AuctionView;
  players: MatchPlayer[];
  decimals: number;
  /** 开箱扫描已经扫完 */
  revealed: boolean;
}) {
  const {result} = view;
  const seats = new Map(result?.seats.map((s) => [s.seat, s]));
  const extra = view.contenders.length > 0;
  const intel = new Map(view.intel_seats.map((s) => [s.seat, s]));

  return (
    <ul className="grid grid-cols-2 gap-2">
      {players.map((p) => {
        const winner = result?.winner_seat === p.seat;
        const contender = extra && view.contenders.includes(p.seat);
        const bid = seats.get(p.seat)?.bid ?? 0;
        const share = seats.get(p.seat)?.share ?? 0;
        const uses = seats.get(p.seat)?.uses ?? [];
        const bought = intel.get(p.seat);
        let note = `${p.seat} 号位`;
        if (result) {
          note = bid ? formatAmount(bid, decimals) : '未出价';
          if (revealed && share > 0) note += ` · 分红 ${formatAmount(share, decimals)}`;
        } else if (contender) {
          note = '加轮中';
        }
        return (
          <li
            key={p.seat}
            className={cn(
                'flex items-center gap-2.5 rounded-lg border bg-card p-2.5',
                p.is_self && 'border-primary/60',
                extra && !contender && 'opacity-50',
                winner && 'glow border-quality-gold',
            )}
          >
            <span className="relative size-9 shrink-0">
              <UserAvatar user={p} className="text-sm" />
              {winner && <Crown className="absolute -top-2 -right-1.5 size-4 text-quality-gold" aria-label="成交" />}
            </span>
            <span className="flex min-w-0 flex-1 flex-col">
              <span className="flex items-center gap-1.5">
                <span className="truncate text-sm font-medium" title={p.nickname}>{p.nickname}</span>
                {p.is_self && <Badge variant="outline" className="px-1.5">我</Badge>}
                {!result && bought && (
                  <Badge
                    variant="outline"
                    className={cn('px-1.5', bought.round === view.round && 'border-quality-gold text-quality-gold')}
                    title={`买过 ${bought.count} 次情报, 最近一次在第 ${bought.round} 轮`}
                  >
                    情报 ×{bought.count}
                  </Badge>
                )}
              </span>
              <span className="font-mono text-xs text-muted-foreground tabular-nums">
                {note}
              </span>
              {uses.length > 0 && (
                <span className="truncate text-xs text-muted-foreground" title={uses.map((u) => `${u.name} ×${u.count}`).join(', ')}>
                  道具 {uses.map((u) => `${u.name} ×${u.count}`).join(', ')}
                </span>
              )}
            </span>
            {revealed && p.rank > 0 && (
              <span className="font-mono text-sm font-bold tabular-nums" title="名次">
                #{p.rank}
              </span>
            )}
          </li>
        );
      })}
    </ul>
  );
}
