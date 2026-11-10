import {Crown} from 'lucide-react';
import {cn} from 'cn';
import {Badge} from '@/components/ui/badge';
import {UserAvatar} from '@/components/common/user';
import {seatSpent} from '@/lib/auction';
import type {AuctionView, LotView} from '@/lib/services/auction';
import type {MatchPlayer} from '@/lib/services/match';
import {formatAmount} from '@/lib/utils';

/**
 * 四个座位: 出价时显示已公开的最高出价, 加轮时标出可以出价的人; 开箱后公开每个人的出价和成交者; 各人已花的钱来自公开的成交结果
 * 名次能推出仓库价值, 扫描完才公开
 */
export function SeatList({view, lot, players, decimals, revealed}: {
  view: AuctionView;
  /** 当前拍品 */
  lot: LotView;
  players: MatchPlayer[];
  decimals: number;
  /** 开箱扫描已经扫完 */
  revealed: boolean;
}) {
  const settled = lot.outcome !== '';
  const seats = new Map(lot.seats?.map((s) => [s.seat, s]));
  const extra = lot.contenders.length > 0;

  return (
    <ul className="grid grid-cols-2 gap-2">
      {players.map((p) => {
        const winner = settled && lot.winner_seat === p.seat;
        const contender = extra && lot.contenders.includes(p.seat);
        const bid = seats.get(p.seat)?.bid ?? 0;
        const shown = lot.bids.reduce((best, b) => (b.seat === p.seat ? Math.max(best, b.amount) : best), 0);
        const spent = seatSpent(view.lots, p.seat);
        const uses = seats.get(p.seat)?.uses ?? [];
        let note = `${p.seat} 号位`;
        if (settled) {
          note = bid ? formatAmount(bid, decimals) : '未出价';
        } else if (contender) {
          note = '加轮中';
        } else if (shown) {
          note = `已出 ${formatAmount(shown, decimals)}`;
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
              </span>
              <span className="font-mono text-xs text-muted-foreground tabular-nums">
                {note}
              </span>
              {spent > 0 && (
                <span className="font-mono text-xs text-muted-foreground tabular-nums">已花 {formatAmount(spent, decimals)}</span>
              )}
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
