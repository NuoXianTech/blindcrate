'use client';

import {useCallback, useEffect, useState} from 'react';
import {Search} from 'lucide-react';
import {toast} from 'sonner';
import {Badge} from '@/components/ui/badge';
import {Button} from '@/components/ui/button';
import {Input} from '@/components/ui/input';
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select';
import {Table, TableBody, TableCell, TableHead, TableHeader, TableRow} from '@/components/ui/table';
import {useAuth} from '@/components/common/auth';
import {Pagination, TableEmpty, TableSkeleton} from '@/components/common/layout';
import {MODES, modeLabel, parseMode, type ModeKind} from '@/lib/auction';
import services from '@/lib/services';
import type {AdminMatch, AdminVenue, Currency, ListAdminMatchesData} from '@/lib/services/admin';
import {toErrorMessage} from '@/lib/services/core';
import {MATCH_STATUS} from '@/lib/services/match';
import {formatDateTime} from '@/lib/utils';
import {CancelMatchDialog} from './CancelMatchDialog';
import {ALL, PAGE_SIZE, STATUS_LABELS, phaseLabel} from './constants';
import {MatchDetailDialog} from './MatchDetailDialog';

/** 地图筛选下拉框中最多列出的地图数 */
const VENUE_LIMIT = 100;

/**
 * 对局管理: 按状态、模式、地图和玩家筛选对局, 查看详情, 取消卡住的对局
 */
export function MatchesMain() {
  const isAdmin = !!useAuth().user?.is_admin;
  const [status, setStatus] = useState(ALL);
  const [kind, setKind] = useState(ALL);
  const [venue, setVenue] = useState(ALL);
  const [userInput, setUserInput] = useState('');
  const [userID, setUserID] = useState<number | undefined>();
  const [current, setCurrent] = useState(1);
  const [data, setData] = useState<ListAdminMatchesData | null>(null);
  const [loading, setLoading] = useState(true);
  const [venues, setVenues] = useState<AdminVenue[]>([]);
  const [currencies, setCurrencies] = useState<Currency[]>([]);
  const [viewing, setViewing] = useState<AdminMatch | null>(null);
  const [cancelling, setCancelling] = useState<AdminMatch | null>(null);

  useEffect(() => {
    if (!isAdmin) return;
    services.admin.listVenues({current: 1, size: VENUE_LIMIT})
        .then((res) => setVenues(res.venues))
        .catch(() => {
          // 只影响地图筛选的选项和地图名称
        });
    services.admin.listCurrencies()
        .then((res) => setCurrencies(res.currencies))
        .catch(() => {
          // 只影响金额的小数位, 失败时按整数展示
        });
  }, [isAdmin]);

  const load = useCallback(async () => {
    // 确认是管理员后再拉取, 否则 AdminGuard 会跳走
    if (!isAdmin) return;
    setLoading(true);
    try {
      setData(await services.admin.listMatches({
        current,
        size: PAGE_SIZE,
        status: status === ALL ? undefined : status,
        prefix: kind === ALL ? undefined : kind,
        suffix: venue === ALL ? undefined : venue,
        user_id: userID,
      }));
    } catch (err) {
      toast.error(toErrorMessage(err));
    } finally {
      setLoading(false);
    }
  }, [current, status, kind, venue, userID, isAdmin]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 翻页和筛选时重新拉取
    void load();
  }, [load]);

  const handleUserSearch = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const value = userInput.trim();
    if (value && !/^\d+$/.test(value)) {
      toast.error('用户 ID 只能填数字');
      return;
    }
    setCurrent(1);
    setUserID(value ? Number(value) : undefined);
  };
  const changeFilter = (set: (value: string) => void) => (value: string) => {
    set(value);
    setCurrent(1);
  };
  const venueName = (code: string) => venues.find((v) => v.code === code)?.name ?? code;
  const decimalsOf = (currency: string) => currencies.find((c) => c.code === currency)?.decimals ?? 0;

  return (
    <div className="flex flex-col gap-4">
      <h1 className="text-2xl font-bold">对局管理</h1>

      <div className="flex flex-col gap-2 sm:flex-row">
        <form onSubmit={handleUserSearch} className="flex flex-1 gap-2">
          <Input
            inputMode="numeric"
            placeholder="玩家的用户 ID"
            value={userInput}
            onChange={(e) => setUserInput(e.target.value)}
            className="sm:max-w-40"
          />
          <Button type="submit" variant="secondary">
            <Search />
            筛选
          </Button>
        </form>
        <Select value={status} onValueChange={changeFilter(setStatus)}>
          <SelectTrigger className="sm:w-32" aria-label="状态">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL}>全部状态</SelectItem>
            {Object.entries(STATUS_LABELS).map(([value, s]) => (
              <SelectItem key={value} value={value}>
                {s.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select value={kind} onValueChange={changeFilter(setKind)}>
          <SelectTrigger className="sm:w-32" aria-label="模式">
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
        <Select value={venue} onValueChange={changeFilter(setVenue)}>
          <SelectTrigger className="sm:w-44" aria-label="地图">
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
              <TableHead>ID</TableHead>
              <TableHead>模式</TableHead>
              <TableHead>地图</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>阶段</TableHead>
              <TableHead>玩家</TableHead>
              <TableHead>开始时间</TableHead>
              <TableHead className="w-0" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading && !data ? (
              <TableSkeleton cols={8} />
            ) : data?.matches.length ? (
              data.matches.map((m) => (
                <MatchRow key={m.id} match={m} venueName={venueName} onView={setViewing} onCancel={setCancelling} />
              ))
            ) : (
              <TableEmpty colSpan={8}>没有对局</TableEmpty>
            )}
          </TableBody>
        </Table>
      </div>

      <Pagination current={current} size={PAGE_SIZE} total={data?.total ?? 0} onChange={setCurrent} />

      <MatchDetailDialog
        match={viewing}
        venueName={venueName}
        decimalsOf={decimalsOf}
        onClose={() => setViewing(null)}
        onCancel={setCancelling}
      />
      <CancelMatchDialog
        match={cancelling}
        onClose={() => setCancelling(null)}
        onSaved={() => {
          setViewing(null);
          void load();
        }}
      />
    </div>
  );
}

/**
 * 对局列表中的一局
 */
function MatchRow({match: m, venueName, onView, onCancel}: {
  match: AdminMatch;
  venueName: (code: string) => string;
  onView: (match: AdminMatch) => void;
  onCancel: (match: AdminMatch) => void;
}) {
  const parsed = parseMode(m.mode);
  const status = STATUS_LABELS[m.status];
  const humans = m.players.filter((p) => !p.is_bot);
  const bots = m.players.length - humans.length;

  return (
    <TableRow>
      <TableCell className="font-mono tabular-nums">{m.id}</TableCell>
      <TableCell>{modeLabel(m.mode)}</TableCell>
      <TableCell className="max-w-40 truncate">{parsed ? venueName(parsed.venue) : m.mode}</TableCell>
      <TableCell>
        <Badge variant={status?.variant ?? 'outline'}>{status?.label ?? m.status}</Badge>
      </TableCell>
      <TableCell className="text-muted-foreground">{phaseLabel(m)}</TableCell>
      <TableCell className="max-w-56">
        <span className="block truncate" title={humans.map((p) => `${p.nickname} #${p.user_id}`).join(', ')}>
          {humans.map((p) => p.nickname).join(', ') || '-'}
          {bots > 0 && <span className="text-muted-foreground"> + {bots} 人机</span>}
        </span>
      </TableCell>
      <TableCell className="text-muted-foreground">{formatDateTime(m.created_at)}</TableCell>
      <TableCell>
        <div className="flex justify-end gap-1">
          <Button variant="ghost" size="sm" onClick={() => onView(m)}>
            详情
          </Button>
          {m.status === MATCH_STATUS.playing && (
            <Button variant="ghost" size="sm" className="text-destructive" onClick={() => onCancel(m)}>
              取消
            </Button>
          )}
        </div>
      </TableCell>
    </TableRow>
  );
}
