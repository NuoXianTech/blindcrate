'use client';

import {useCallback, useEffect, useState} from 'react';
import Link from 'next/link';
import {toast} from 'sonner';
import {cn} from 'cn';
import {Badge} from '@/components/ui/badge';
import {Button} from '@/components/ui/button';
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select';
import {Table, TableBody, TableCell, TableHead, TableHeader, TableRow} from '@/components/ui/table';
import {useAuth} from '@/components/common/auth';
import {Pagination, TableEmpty, TableSkeleton} from '@/components/common/layout';
import {CurrencyIcon} from '@/components/common/wallet';
import {matchHref} from '@/lib/access';
import {MODES, OUTCOMES, modeLabel, type ModeKind} from '@/lib/auction';
import services from '@/lib/services';
import {AUCTION_OUTCOME} from '@/lib/services/auction';
import {toErrorMessage} from '@/lib/services/core';
import {MATCH_STATUS, type ListMatchesData, type MatchView} from '@/lib/services/match';
import type {Venue} from '@/lib/services/venue';
import type {BalanceItem} from '@/lib/services/wallet';
import {formatAmount, formatDateTime} from '@/lib/utils';

/** 每页的对局数 */
const PAGE_SIZE = 20;

/** 筛选下拉框中的"全部" */
const ALL = 'all';

/**
 * 战绩: 自己参与过的对局, 可按模式和地图筛选, 点开查看结算和全部藏品
 */
export function MatchHistoryMain() {
  const {user} = useAuth();
  const [kind, setKind] = useState(ALL);
  const [venue, setVenue] = useState(ALL);
  const [current, setCurrent] = useState(1);
  const [data, setData] = useState<ListMatchesData | null>(null);
  const [loading, setLoading] = useState(true);
  const [venues, setVenues] = useState<Venue[]>([]);
  const [currencies, setCurrencies] = useState<BalanceItem[]>([]);

  useEffect(() => {
    if (!user) return;
    services.venue.listVenues()
        .then((res) => setVenues(res.venues))
        .catch(() => {
          // 只影响地图筛选的选项
        });
    services.wallet.listBalances()
        .then((res) => setCurrencies(res.balances))
        .catch(() => {
          // 只影响金额的小数位, 失败时按整数展示
        });
  }, [user]);

  const load = useCallback(async () => {
    if (!user) return;
    setLoading(true);
    try {
      setData(await services.match.listMatches({
        current,
        size: PAGE_SIZE,
        prefix: kind === ALL ? undefined : kind,
        suffix: venue === ALL ? undefined : venue,
      }));
    } catch (err) {
      toast.error(toErrorMessage(err));
    } finally {
      setLoading(false);
    }
  }, [current, kind, venue, user]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 翻页和筛选时重新拉取
    void load();
  }, [load]);

  const changeKind = (value: string) => {
    setKind(value);
    setCurrent(1);
  };
  const changeVenue = (value: string) => {
    setVenue(value);
    setCurrent(1);
  };
  const decimalsOf = (currency: string) => currencies.find((c) => c.currency === currency)?.decimals ?? 0;

  return (
    <div className="flex flex-col gap-4">
      <h1 className="text-2xl font-bold">战绩</h1>

      <div className="flex flex-col gap-2 sm:flex-row">
        <Select value={kind} onValueChange={changeKind}>
          <SelectTrigger className="sm:w-36" aria-label="模式">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL}>全部模式</SelectItem>
            {(Object.keys(MODES) as ModeKind[]).map((k) => (
              <SelectItem key={k} value={k}>
                {MODES[k].label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select value={venue} onValueChange={changeVenue}>
          <SelectTrigger className="sm:w-48" aria-label="地图">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL}>全部地图</SelectItem>
            {venues.map((v) => (
              <SelectItem key={v.code} value={v.code}>
                {v.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="rounded-xl border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>时间</TableHead>
              <TableHead>地图</TableHead>
              <TableHead>模式</TableHead>
              <TableHead>名次</TableHead>
              <TableHead>结果</TableHead>
              <TableHead className="text-right">盈亏</TableHead>
              <TableHead className="w-0" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading && !data ? (
              <TableSkeleton cols={7} />
            ) : data?.matches.length ? (
              data.matches.map((m) => <MatchRow key={m.id} match={m} decimalsOf={decimalsOf} />)
            ) : (
              <TableEmpty colSpan={7}>{kind === ALL && venue === ALL ? '还没有打过对局' : '没有符合条件的对局'}</TableEmpty>
            )}
          </TableBody>
        </Table>
      </div>

      <Pagination current={current} size={PAGE_SIZE} total={data?.total ?? 0} onChange={setCurrent} />
    </div>
  );
}

/** 对局的结果, 如 "我拍下"、"流拍" */
function resultOf(m: MatchView): { label: string; className?: string } {
  const s = m.summary;
  if (m.status === MATCH_STATUS.playing && !s?.outcome) return {label: '进行中', className: 'text-primary'};
  if (!s?.outcome) return {label: m.status === MATCH_STATUS.cancelled ? '已取消' : '-'};
  if (s.outcome === AUCTION_OUTCOME.sold) {
    return s.won ? {label: '我拍下', className: 'font-medium text-quality-gold'} : {label: '他人拍下'};
  }
  return {label: OUTCOMES[s.outcome] ?? s.outcome, className: 'text-muted-foreground'};
}

/**
 * 战绩中的一局
 */
function MatchRow({match: m, decimalsOf}: { match: MatchView; decimalsOf: (currency: string) => number }) {
  const s = m.summary;
  const self = m.players.find((p) => p.is_self);
  const result = resultOf(m);
  const settled = !!s?.outcome;

  return (
    <TableRow>
      <TableCell className="text-muted-foreground">{formatDateTime(m.created_at)}</TableCell>
      <TableCell className="max-w-48 truncate" title={s?.venue_name}>
        {s?.venue_name ?? m.mode}
      </TableCell>
      <TableCell>
        <span className="flex items-center gap-1.5">
          {modeLabel(m.mode)}
          {s?.is_practice && <Badge variant="outline">练习</Badge>}
        </span>
      </TableCell>
      <TableCell className="font-mono tabular-nums">
        {self?.rank ? (
          <>
            {self.rank}
            <span className="text-muted-foreground"> / {m.players.length}</span>
            {!!s?.rating_delta && (
              <span
                className={cn('ml-2 text-xs', s.rating_delta > 0 ? 'text-quality-green' : 'text-quality-red')}
                title="段位分变化"
              >
                {formatAmount(s.rating_delta, 0, {signed: true})} 分
              </span>
            )}
          </>
        ) : (
          '-'
        )}
      </TableCell>
      <TableCell className={result.className}>{result.label}</TableCell>
      <TableCell className="text-right">
        {s && settled ? (
          <span
            className={cn(
                'inline-flex items-center gap-1 font-mono tabular-nums',
                s.profit > 0 && 'text-quality-green',
                s.profit < 0 && 'text-quality-red',
                s.is_practice && 'opacity-60',
            )}
            title={s.is_practice ? '练习局, 仅供参考' : undefined}
          >
            <CurrencyIcon currency={s.bid_currency} className="size-4" />
            {formatAmount(s.profit, decimalsOf(s.bid_currency), {signed: true})}
          </span>
        ) : (
          <span className="text-muted-foreground">-</span>
        )}
      </TableCell>
      <TableCell>
        <Button variant="ghost" size="sm" asChild>
          <Link href={matchHref(m.id)}>查看</Link>
        </Button>
      </TableCell>
    </TableRow>
  );
}
