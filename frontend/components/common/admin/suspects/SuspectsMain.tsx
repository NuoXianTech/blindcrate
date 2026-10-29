'use client';

import {useCallback, useEffect, useState} from 'react';
import Link from 'next/link';
import {toast} from 'sonner';
import {Badge} from '@/components/ui/badge';
import {Button} from '@/components/ui/button';
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select';
import {Table, TableBody, TableCell, TableHead, TableHeader, TableRow} from '@/components/ui/table';
import {useAuth} from '@/components/common/auth';
import {Pagination, TableEmpty, TableSkeleton} from '@/components/common/layout';
import services from '@/lib/services';
import type {ListSuspectsData, Suspect, SuspectUser} from '@/lib/services/admin';
import {toErrorMessage} from '@/lib/services/core';
import {formatDateTime} from '@/lib/utils';
import {
  PAGE_SIZE,
  REVIEW_FILTERS,
  formatRate,
  isReviewed,
  pairMatchesHref,
  reviewedParam,
  userHref,
  type ReviewFilter,
} from './constants';
import {ReviewSuspectDialog} from './ReviewSuspectDialog';

/**
 * 可疑名单: 排位中经常同局, 或同局时其中一人多次低价成交的两个账号, 查看一起打的对局后标记已处理
 */
export function SuspectsMain() {
  const isAdmin = !!useAuth().user?.is_admin;
  const [filter, setFilter] = useState<ReviewFilter>('pending');
  const [current, setCurrent] = useState(1);
  const [data, setData] = useState<ListSuspectsData | null>(null);
  const [loading, setLoading] = useState(true);
  const [reviewing, setReviewing] = useState<Suspect | null>(null);

  const load = useCallback(async () => {
    // 确认是管理员后再拉取, 否则 AdminGuard 会跳走
    if (!isAdmin) return;
    setLoading(true);
    try {
      setData(await services.admin.listSuspects({current, size: PAGE_SIZE, reviewed: reviewedParam(filter)}));
    } catch (err) {
      toast.error(toErrorMessage(err));
    } finally {
      setLoading(false);
    }
  }, [current, filter, isAdmin]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 翻页和筛选时重新拉取
    void load();
  }, [load]);

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between gap-2">
        <div className="flex flex-col gap-1">
          <h1 className="text-2xl font-bold">可疑名单</h1>
          <p className="text-sm text-muted-foreground">
            最近一段时间的排位中经常同局, 或同局时其中一人多次低价成交的两个账号, 可能约好不出价低价拿走仓库;
            统计天数、次数和低价成交比例在系统设置的排位中修改
          </p>
        </div>
        <Select
          value={filter}
          onValueChange={(value) => {
            setFilter(value as ReviewFilter);
            setCurrent(1);
          }}
        >
          <SelectTrigger className="w-28 shrink-0" aria-label="处理状态">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {REVIEW_FILTERS.map((f) => (
              <SelectItem key={f.value} value={f.value}>
                {f.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="rounded-xl border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>账号</TableHead>
              <TableHead className="text-right">同局</TableHead>
              <TableHead className="text-right">其中一人成交</TableHead>
              <TableHead className="text-right">低价成交</TableHead>
              <TableHead className="text-right">平均成交比例</TableHead>
              <TableHead>最近同局</TableHead>
              <TableHead>处理</TableHead>
              <TableHead className="w-0" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading && !data ? (
              <TableSkeleton cols={8} />
            ) : data?.suspects.length ? (
              data.suspects.map((s) => (
                <SuspectRow key={`${s.user_a.id}:${s.user_b.id}`} suspect={s} onReview={() => setReviewing(s)} />
              ))
            ) : (
              <TableEmpty colSpan={8}>{filter === 'pending' ? '没有待处理的可疑账号' : '没有可疑账号'}</TableEmpty>
            )}
          </TableBody>
        </Table>
      </div>

      <Pagination current={current} size={PAGE_SIZE} total={data?.total ?? 0} onChange={setCurrent} />

      <ReviewSuspectDialog suspect={reviewing} onClose={() => setReviewing(null)} onSaved={load} />
    </div>
  );
}

/**
 * 可疑名单的一行: 两个账号和同局统计, 成交比例为成交价占仓库价值的比例
 */
function SuspectRow({suspect: s, onReview}: { suspect: Suspect; onReview: () => void }) {
  const reviewed = isReviewed(s);

  return (
    <TableRow>
      <TableCell>
        <div className="flex flex-col gap-1">
          <SuspectAccount user={s.user_a} />
          <SuspectAccount user={s.user_b} />
        </div>
      </TableCell>
      <TableCell className="text-right font-mono tabular-nums">{s.matches}</TableCell>
      <TableCell className="text-right font-mono tabular-nums">{s.wins}</TableCell>
      <TableCell className="text-right font-mono tabular-nums">
        <span className={s.low_wins > 0 ? 'text-destructive' : undefined}>{s.low_wins}</span>
      </TableCell>
      <TableCell className="text-right font-mono tabular-nums">{s.wins ? formatRate(s.price_rate) : '-'}</TableCell>
      <TableCell className="text-muted-foreground">
        <span className="font-mono">#{s.last_match_id}</span> {formatDateTime(s.last_match_at)}
      </TableCell>
      <TableCell className="max-w-56">
        {reviewed ? (
          <span className="block truncate text-muted-foreground" title={s.review!.note || undefined}>
            <Badge variant="secondary">已处理</Badge> {s.review!.note}
          </span>
        ) : s.review ? (
          <Badge variant="outline" title={s.review.note || undefined}>处理后又同局</Badge>
        ) : (
          <Badge variant="outline">待处理</Badge>
        )}
      </TableCell>
      <TableCell>
        <div className="flex justify-end gap-1">
          <Button variant="ghost" size="sm" asChild>
            <Link href={pairMatchesHref(s)}>对局</Link>
          </Button>
          <Button variant="ghost" size="sm" onClick={onReview}>
            {reviewed ? '修改' : '处理'}
          </Button>
        </div>
      </TableCell>
    </TableRow>
  );
}

/**
 * 一个账号: 昵称链接到用户管理, 带上封禁、停用状态
 */
function SuspectAccount({user: u}: { user: SuspectUser }) {
  const banned = !!u.banned_until && new Date(u.banned_until) > new Date();

  return (
    <div className="flex items-center gap-2 text-sm">
      <Link href={userHref(u.username)} className="hover:underline">
        {u.nickname || u.username}
      </Link>
      <span className="text-xs text-muted-foreground">@{u.username} #{u.id}</span>
      {u.is_disabled && <Badge variant="destructive">已停用</Badge>}
      {banned && <Badge variant="outline" className="border-destructive/50 text-destructive">封禁中</Badge>}
    </div>
  );
}
