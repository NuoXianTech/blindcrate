'use client';

import {useEffect, useState} from 'react';
import {Ban, LoaderCircle} from 'lucide-react';
import {toast} from 'sonner';
import {cn} from 'cn';
import {Badge} from '@/components/ui/badge';
import {Button} from '@/components/ui/button';
import {Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle} from '@/components/ui/dialog';
import {Table, TableBody, TableCell, TableHead, TableHeader, TableRow} from '@/components/ui/table';
import {IntelResult, Warehouse} from '@/components/common/auction';
import {useScan} from '@/hooks/use-scan';
import {OUTCOMES, modeLabel, parseMode} from '@/lib/auction';
import services from '@/lib/services';
import {AUCTION_OUTCOME, type AuctionAdminView, type AuctionPlayer, type LotView} from '@/lib/services/auction';
import {toErrorMessage} from '@/lib/services/core';
import {MATCH_STATUS, type AdminMatch, type AdminMatchPlayer} from '@/lib/services/match';
import {formatAmount, formatDateTime} from '@/lib/utils';
import {STATUS_LABELS, phaseLabel} from './constants';

/**
 * 对局详情: 时间、每个座位, 暗拍还显示每个拍品的档次、价值和成交结果, 每个座位的带入资金、已花、当前出价、估值和盈亏, 道具的使用记录, 以及各拍品的仓库
 */
export function MatchDetailDialog({match, venueName, decimalsOf, onClose, onCancel}: {
  match: AdminMatch | null;
  venueName: (code: string) => string;
  decimalsOf: (currency: string) => number;
  onClose: () => void;
  /** 取消进行中的对局 */
  onCancel: (match: AdminMatch) => void;
}) {
  return (
    <Dialog open={!!match} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-4xl">
        {match && (
          <MatchDetail key={match.id} match={match} venueName={venueName} decimalsOf={decimalsOf} onCancel={onCancel} />
        )}
      </DialogContent>
    </Dialog>
  );
}

/**
 * 一局对局的详情
 */
function MatchDetail({match: m, venueName, decimalsOf, onCancel}: {
  match: AdminMatch;
  venueName: (code: string) => string;
  decimalsOf: (currency: string) => number;
  onCancel: (match: AdminMatch) => void;
}) {
  const parsed = parseMode(m.mode);
  const isAuction = !!parsed;
  const [auction, setAuction] = useState<AuctionAdminView | null>(null);
  const [loading, setLoading] = useState(isAuction);
  const [shownLot, setShownLot] = useState(0);

  useEffect(() => {
    if (!isAuction) return;
    services.auctionAdmin.getAuction(m.id)
        .then(setAuction)
        .catch((err) => toast.error(toErrorMessage(err)))
        .finally(() => setLoading(false));
  }, [m.id, isAuction]);

  const status = STATUS_LABELS[m.status];
  const currency = auction?.venue.bid_currency ?? '';
  const decimals = decimalsOf(currency);
  // 拍场结束后才有名次和盈亏
  const settled = !!auction?.session;
  const lotNo = shownLot || auction?.lot || 1;
  const lot = auction?.lots[lotNo - 1];
  const times = [
    {label: '开始', value: m.created_at},
    {label: '释放', value: m.released_at},
    {label: '结束', value: m.finished_at},
  ];

  return (
    <>
      <DialogHeader>
        <DialogTitle className="flex items-center gap-2">
          对局 #{m.id}
          <Badge variant={status?.variant ?? 'outline'}>{status?.label ?? m.status}</Badge>
        </DialogTitle>
        <DialogDescription>
          {modeLabel(m.mode)} · {parsed ? (auction?.venue.name ?? venueName(parsed.venue)) : m.mode} · {phaseLabel(m)}
        </DialogDescription>
      </DialogHeader>

      <dl className="grid grid-cols-3 gap-3 text-sm">
        {times.map((t) => (
          <div key={t.label} className="flex flex-col gap-0.5">
            <dt className="text-xs text-muted-foreground">{t.label}</dt>
            <dd>{t.value ? formatDateTime(t.value) : '-'}</dd>
          </div>
        ))}
      </dl>

      {auction && (
        <div className="rounded-lg border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>拍品</TableHead>
                <TableHead>档次</TableHead>
                <TableHead>格子</TableHead>
                <TableHead>结果</TableHead>
                <TableHead className="text-right">起拍价</TableHead>
                <TableHead className="text-right">成交价</TableHead>
                <TableHead className="text-right">仓库价值</TableHead>
                <TableHead className="text-right">件数</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {auction.lot_rows.map((l) => (
                <TableRow
                  key={l.lot_no}
                  className={cn('cursor-pointer', l.lot_no === lotNo && 'bg-muted/50')}
                  onClick={() => setShownLot(l.lot_no)}
                  title="查看这个拍品的仓库"
                >
                  <TableCell className="font-mono tabular-nums">
                    {l.lot_no}
                    {l.lot_no === auction.lot && !settled && <Badge variant="outline" className="ml-1.5">当前</Badge>}
                  </TableCell>
                  <TableCell>{auction.lots[l.lot_no - 1]?.tier || l.tier}</TableCell>
                  <TableCell className="font-mono tabular-nums">{l.grid.width}×{l.grid.height}</TableCell>
                  <TableCell>
                    {l.outcome ? OUTCOMES[l.outcome] ?? l.outcome : l.first_round ? '竞拍中' : '未开拍'}
                    {l.outcome === AUCTION_OUTCOME.sold && ` · ${l.winner_seat} 号位`}
                  </TableCell>
                  <TableCell className="text-right font-mono tabular-nums">{formatAmount(l.start_price, decimals)}</TableCell>
                  <TableCell className="text-right font-mono tabular-nums">
                    {l.outcome === AUCTION_OUTCOME.sold ? formatAmount(l.price, decimals) : '-'}
                  </TableCell>
                  <TableCell className="text-right font-mono tabular-nums">{formatAmount(l.value, decimals)}</TableCell>
                  <TableCell className="text-right font-mono tabular-nums">{l.piece_count}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      <div className="rounded-lg border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>座位</TableHead>
              <TableHead>玩家</TableHead>
              <TableHead>名次</TableHead>
              {auction && (
                <>
                  <TableHead className="text-right">带入</TableHead>
                  <TableHead className="text-right">已花</TableHead>
                  <TableHead className="text-right" title="当前拍品的出价">出价</TableHead>
                  <TableHead className="text-right" title="人机对当前拍品价值的估值">
                    估值
                  </TableHead>
                  <TableHead className="text-right">盈亏</TableHead>
                </>
              )}
            </TableRow>
          </TableHeader>
          <TableBody>
            {m.players.map((p) => (
              <SeatRow
                key={p.seat}
                player={p}
                auction={auction?.players.find((a) => a.seat === p.seat)}
                settled={settled}
                winner={settled && auction?.players.find((a) => a.seat === p.seat)?.rank === 1}
                decimals={decimals}
              />
            ))}
          </TableBody>
        </Table>
      </div>

      {auction && auction.uses.length > 0 && (
        <div className="flex flex-col gap-2">
          <h3 className="text-sm font-medium">道具使用记录</h3>
          <div className="rounded-lg border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>座位</TableHead>
                  <TableHead>拍品</TableHead>
                  <TableHead>轮次</TableHead>
                  <TableHead>道具</TableHead>
                  <TableHead>情报</TableHead>
                  <TableHead>时间</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {auction.uses.map((u) => (
                  <TableRow key={u.id}>
                    <TableCell className="font-mono tabular-nums">{u.seat}</TableCell>
                    <TableCell className="font-mono tabular-nums">{u.lot_no}</TableCell>
                    <TableCell className="font-mono tabular-nums">{u.round}</TableCell>
                    <TableCell>
                      <span className="flex flex-col">
                        {u.name}
                        <span className="text-xs text-muted-foreground">#{u.item_id}</span>
                      </span>
                    </TableCell>
                    <TableCell className="whitespace-normal">
                      <IntelResult intel={u.result} currency={currency} decimals={decimals} />
                    </TableCell>
                    <TableCell className="text-muted-foreground">{formatDateTime(u.created_at)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </div>
      )}

      {loading ? (
        <div className="flex h-40 items-center justify-center text-muted-foreground">
          <LoaderCircle className="animate-spin" />
        </div>
      ) : (
        auction && lot && (
          <div className="mx-auto w-full max-w-2xl">
            <AdminWarehouse key={lot.lot_no} view={auction} lot={lot} decimals={decimals} />
          </div>
        )
      )}

      {m.status === MATCH_STATUS.playing && (
        <DialogFooter>
          <Button variant="destructive" onClick={() => onCancel(m)}>
            <Ban />
            取消对局
          </Button>
        </DialogFooter>
      )}
    </>
  );
}

/**
 * 详情中一个拍品的仓库, 不播放开箱扫描, 结算后直接显示全部藏品
 */
function AdminWarehouse({view, lot, decimals}: { view: AuctionAdminView; lot: LotView; decimals: number }) {
  const scan = useScan(lot, view.venue.bid_currency, true);
  return <Warehouse view={view} lot={lot} decimals={decimals} scan={scan} />;
}

/**
 * 详情中的一个座位
 */
function SeatRow({player: p, auction: a, settled, winner, decimals}: {
  player: AdminMatchPlayer;
  /** 暗拍数据, 不是暗拍时为空 */
  auction?: AuctionPlayer;
  settled: boolean;
  winner: boolean;
  decimals: number;
}) {
  const amount = (value: number) => (value ? formatAmount(value, decimals) : '-');
  return (
    <TableRow className={cn(winner && 'bg-quality-gold/10')}>
      <TableCell className="font-mono tabular-nums">{p.seat}</TableCell>
      <TableCell>
        <span className="flex items-center gap-1.5">
          {p.nickname}
          {p.is_bot ? (
            <Badge variant="secondary" title={a?.persona.code}>
              {a?.persona.name ? `人机 · ${a.persona.name}` : '人机'}
            </Badge>
          ) : (
            <span className="text-xs text-muted-foreground">#{p.user_id}</span>
          )}
          {a?.is_practice && <Badge variant="outline">练习</Badge>}
        </span>
      </TableCell>
      <TableCell className="font-mono tabular-nums">
        {p.rank || '-'}
        {!!a?.rating_delta && (
          <span
            className={cn('ml-1.5 text-xs', a.rating_delta > 0 ? 'text-quality-green' : 'text-quality-red')}
            title={`段位分变化, 结算后为 ${a.rating}`}
          >
            {formatAmount(a.rating_delta, 0, {signed: true})}
          </span>
        )}
      </TableCell>
      {a && (
        <>
          <TableCell className="text-right font-mono tabular-nums">{amount(a.carry)}</TableCell>
          <TableCell className="text-right font-mono tabular-nums">{amount(a.spent)}</TableCell>
          <TableCell className="text-right font-mono tabular-nums">{amount(a.bid)}</TableCell>
          <TableCell className="text-right font-mono tabular-nums">{amount(a.estimate)}</TableCell>
          <TableCell
            className={cn(
                'text-right font-mono tabular-nums',
                settled && a.profit > 0 && 'text-quality-green',
                settled && a.profit < 0 && 'text-quality-red',
            )}
          >
            {settled ? formatAmount(a.profit, decimals, {signed: true}) : '-'}
          </TableCell>
        </>
      )}
    </TableRow>
  );
}
