'use client';

import {useEffect, useState} from 'react';
import {Ban, LoaderCircle} from 'lucide-react';
import {toast} from 'sonner';
import {cn} from 'cn';
import {Badge} from '@/components/ui/badge';
import {Button} from '@/components/ui/button';
import {Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle} from '@/components/ui/dialog';
import {Table, TableBody, TableCell, TableHead, TableHeader, TableRow} from '@/components/ui/table';
import {Warehouse} from '@/components/common/auction';
import {CurrencyIcon} from '@/components/common/wallet';
import {OUTCOMES, modeLabel, parseMode} from '@/lib/auction';
import services from '@/lib/services';
import type {AdminMatch, AdminMatchPlayer} from '@/lib/services/admin';
import {AUCTION_OUTCOME, type AuctionAdminView, type AuctionPlayer} from '@/lib/services/auction';
import {toErrorMessage} from '@/lib/services/core';
import {MATCH_STATUS} from '@/lib/services/match';
import {formatAmount, formatDateTime} from '@/lib/utils';
import {STATUS_LABELS, phaseLabel} from './constants';

/**
 * 对局详情: 时间、每个座位, 暗拍还显示每个座位的带入资金、出价、估值和盈亏, 以及结算结果和仓库
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

  useEffect(() => {
    if (!isAuction) return;
    services.admin.getMatchAuction(m.id)
        .then(setAuction)
        .catch((err) => toast.error(toErrorMessage(err)))
        .finally(() => setLoading(false));
  }, [m.id, isAuction]);

  const status = STATUS_LABELS[m.status];
  const currency = auction?.venue.bid_currency ?? '';
  const decimals = decimalsOf(currency);
  const result = auction?.result ?? null;
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

      {result && (
        <dl className="grid grid-cols-2 gap-3 rounded-lg border p-3 text-sm sm:grid-cols-4">
          <div className="flex flex-col gap-0.5">
            <dt className="text-xs text-muted-foreground">结果</dt>
            <dd className="font-medium">
              {OUTCOMES[result.outcome] ?? result.outcome}
              {result.outcome === AUCTION_OUTCOME.sold && ` · ${result.winner_seat} 号位`}
            </dd>
          </div>
          {[
            {label: '成交价', value: result.outcome === AUCTION_OUTCOME.sold ? formatAmount(result.price, decimals) : '-'},
            {label: '仓库价值', value: formatAmount(result.value, decimals)},
          ].map((row) => (
            <div key={row.label} className="flex flex-col gap-0.5">
              <dt className="text-xs text-muted-foreground">{row.label}</dt>
              <dd className="flex items-center gap-1 font-mono tabular-nums">
                <CurrencyIcon currency={currency} className="size-4" />
                {row.value}
              </dd>
            </div>
          ))}
        </dl>
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
                  <TableHead className="text-right">出价</TableHead>
                  <TableHead className="text-right" title="人机对仓库价值的估值">
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
                settled={!!result}
                winner={result?.outcome === AUCTION_OUTCOME.sold && result.winner_seat === p.seat}
                decimals={decimals}
              />
            ))}
          </TableBody>
        </Table>
      </div>

      {loading ? (
        <div className="flex h-40 items-center justify-center text-muted-foreground">
          <LoaderCircle className="animate-spin" />
        </div>
      ) : (
        auction && (
          <div className="mx-auto w-full max-w-2xl">
            <Warehouse view={auction} decimals={decimals} finished />
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
            <Badge variant="secondary">人机 Lv{p.bot_level}</Badge>
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
