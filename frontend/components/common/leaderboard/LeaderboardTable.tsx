import {cn} from 'cn';
import {Table, TableBody, TableCell, TableHead, TableHeader, TableRow} from '@/components/ui/table';
import {TableEmpty, TableSkeleton} from '@/components/common/layout';
import {UserAvatar} from '@/components/common/user';
import {CurrencyIcon} from '@/components/common/wallet';
import type {Leaderboard, LeaderboardEntry} from '@/lib/services/leaderboard';
import {formatAmount} from '@/lib/utils';

/** 前三名的名次颜色 */
const PODIUM_CLASSES: Record<number, string> = {
  1: 'bg-quality-gold/20 text-quality-gold',
  2: 'bg-quality-gray/20 text-quality-gray',
  3: 'bg-quality-red/15 text-quality-red',
};

/** 名次徽章, 前三名突出显示 */
export function RankBadge({rank, className}: { rank: number; className?: string }) {
  return (
    <span
      className={cn(
          'inline-flex h-8 min-w-8 items-center justify-center rounded-full px-1.5 font-mono text-sm font-bold tabular-nums',
          PODIUM_CLASSES[rank] ?? 'text-muted-foreground',
          className,
      )}
    >
      {rank}
    </span>
  );
}

/** 榜单数值, 净收益按货币的小数位显示并带正负号 */
export function BoardValue({board, value, decimals}: { board: Leaderboard; value: number; decimals: number }) {
  if (!board.currency) {
    return <span className="font-mono font-bold tabular-nums">{formatAmount(value)}</span>;
  }
  return (
    <span
      className={cn(
          'inline-flex items-center gap-1 font-mono font-bold tabular-nums',
          value > 0 && 'text-quality-green',
          value < 0 && 'text-destructive',
      )}
    >
      <CurrencyIcon currency={board.currency} className="size-4" />
      {formatAmount(value, decimals, {signed: true})}
    </span>
  );
}

/**
 * 榜单的前 N 名: 名次、头像、昵称和数值, 自己所在的行高亮
 */
export function LeaderboardTable({board, entries, decimals}: {
  board: Leaderboard;
  /** 为 null 时加载中 */
  entries: LeaderboardEntry[] | null;
  decimals: number;
}) {
  return (
    <div className="rounded-xl border bg-card">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-20 text-center">名次</TableHead>
            <TableHead>玩家</TableHead>
            <TableHead className="text-right">{board.metric_name}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {!entries ? (
            <TableSkeleton cols={3} />
          ) : entries.length ? (
            entries.map((e) => (
              <TableRow key={e.rank} className={cn(e.is_self && 'bg-primary/10 hover:bg-primary/15')}>
                <TableCell className="text-center">
                  <RankBadge rank={e.rank} />
                </TableCell>
                <TableCell>
                  <span className="flex min-w-0 items-center gap-3">
                    <span className="size-8 shrink-0">
                      <UserAvatar user={e} className="text-sm" />
                    </span>
                    <span className="truncate font-medium">{e.nickname}</span>
                    {e.is_self && <span className="shrink-0 rounded-md border px-1.5 text-xs text-muted-foreground">我</span>}
                  </span>
                </TableCell>
                <TableCell className="text-right">
                  <BoardValue board={board} value={e.value} decimals={decimals} />
                </TableCell>
              </TableRow>
            ))
          ) : (
            <TableEmpty colSpan={3}>还没有人上榜</TableEmpty>
          )}
        </TableBody>
      </Table>
    </div>
  );
}
