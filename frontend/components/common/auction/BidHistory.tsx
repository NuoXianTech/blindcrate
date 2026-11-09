import type {AuctionPublicView, LotView} from '@/lib/services/auction';
import type {MatchPlayer} from '@/lib/services/match';
import {formatAmount} from '@/lib/utils';

/**
 * 当前拍品的公开出价, 每轮一行: 前 N-1 轮结束后实名公开, 最后一轮是暗标, 开箱后才公开
 */
export function BidHistory({view, lot, players, decimals}: {
  view: AuctionPublicView;
  /** 当前拍品 */
  lot: LotView;
  players: MatchPlayer[];
  decimals: number;
}) {
  const rounds = [...new Set(lot.bids.map((b) => b.round))].sort((a, b) => b - a);
  const amountOf = (round: number, seat: number) => lot.bids.find((b) => b.round === round && b.seat === seat);

  return (
    <section className="flex flex-col gap-2">
      <h2 className="text-sm font-medium">公开出价</h2>
      {rounds.length === 0 ? (
        <p className="text-sm text-muted-foreground">
          每轮结束后公开所有人这一轮的出价; 最后一轮是暗标, 开箱后才公开, 共 {view.venue.rounds} 轮
        </p>
      ) : (
        <div className="overflow-x-auto rounded-lg border bg-card">
          <table className="w-full text-xs">
            <thead>
              <tr className="border-b text-muted-foreground">
                <th className="px-2 py-1.5 text-left font-normal">轮</th>
                {players.map((p) => (
                  <th key={p.seat} className="max-w-16 truncate px-2 py-1.5 text-right font-normal" title={p.nickname}>
                    {p.is_self ? '我' : p.nickname}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {rounds.map((round) => (
                <tr key={round} className="border-b last:border-0">
                  <td className="px-2 py-1.5 text-muted-foreground">{round - lot.first_round + 1}</td>
                  {players.map((p) => {
                    const bid = amountOf(round, p.seat);
                    return (
                      <td key={p.seat} className="px-2 py-1.5 text-right font-mono tabular-nums">
                        {bid?.amount ? formatAmount(bid.amount, decimals) : <span className="text-muted-foreground">-</span>}
                      </td>
                    );
                  })}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}
