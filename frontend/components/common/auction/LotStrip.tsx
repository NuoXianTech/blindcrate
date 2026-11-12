import {cn} from 'cn';
import type {Scan} from '@/hooks/use-scan';
import {OUTCOMES} from '@/lib/auction';
import {AUCTION_OUTCOME, type AuctionPublicView} from '@/lib/services/auction';
import type {MatchPlayer} from '@/lib/services/match';
import {formatAmount} from '@/lib/utils';

/**
 * 拍场进度: 每个拍品的格子大小和结果, 开局时就能看到后面还有多大的仓库, 用来分配带入资金
 * 点一个拍品查看它的仓库; 当前拍品的价值扫描完才公开
 */
export function LotStrip({view, players, decimals, scan, shown, onSelect}: {
  view: AuctionPublicView;
  players: MatchPlayer[];
  decimals: number;
  /** 当前拍品的开箱扫描进度 */
  scan: Scan;
  /** 正在查看的拍品的序号 */
  shown: number;
  onSelect: (lotNo: number) => void;
}) {
  return (
    <ol className="flex gap-2 overflow-x-auto p-0.5 pb-1">
      {view.lots.map((l) => {
        const current = l.lot_no === view.lot;
        const selected = l.lot_no === shown;
        const winner = players.find((p) => p.seat === l.winner_seat);
        const showValue = l.outcome !== '' && (!current || scan.complete);
        let status = current ? '竞拍中' : '未开拍';
        if (l.outcome === AUCTION_OUTCOME.sold) status = `${winner?.is_self ? '我' : winner?.nickname ?? `${l.winner_seat} 号位`} ${formatAmount(l.price, decimals)}`;
        else if (l.outcome) status = OUTCOMES[l.outcome] ?? l.outcome;
        return (
          <li key={l.lot_no}>
            <button
              type="button"
              aria-current={current ? 'step' : undefined}
              aria-pressed={selected}
              title={selected ? undefined : `查看拍品 ${l.lot_no} 的仓库`}
              onClick={() => onSelect(l.lot_no)}
              className={cn(
                  'flex min-w-32 cursor-pointer flex-col gap-0.5 rounded-lg border bg-card px-3 py-2 text-left text-xs outline-none transition-colors',
                  'hover:bg-accent focus-visible:ring-2 focus-visible:ring-ring',
                  current && 'border-primary/60',
                  selected && 'ring-2 ring-primary',
                  !current && !selected && l.outcome === '' && 'opacity-60',
              )}
            >
              <span className="flex items-center justify-between gap-2">
                <span className="font-medium">拍品 {l.lot_no}</span>
                <span className="font-mono tabular-nums text-muted-foreground">{l.grid.width}×{l.grid.height}</span>
              </span>
              <span className="truncate text-muted-foreground" title={status}>{status}</span>
              <span className="font-mono tabular-nums text-muted-foreground">
                {showValue ? `价值 ${formatAmount(l.value, decimals)}` : `起拍 ${formatAmount(l.start_price, decimals)}`}
                {l.tier && ` · ${l.tier}`}
              </span>
            </button>
          </li>
        );
      })}
    </ol>
  );
}
