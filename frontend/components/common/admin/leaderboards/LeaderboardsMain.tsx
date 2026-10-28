'use client';

import {useCallback, useEffect, useMemo, useState} from 'react';
import {Plus, Search} from 'lucide-react';
import {toast} from 'sonner';
import {Button} from '@/components/ui/button';
import {Input} from '@/components/ui/input';
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select';
import {Table, TableBody, TableHead, TableHeader, TableRow} from '@/components/ui/table';
import {useAuth} from '@/components/common/auth';
import {Pagination, TableEmpty, TableSkeleton} from '@/components/common/layout';
import type {ScopeOption} from '@/components/common/admin/editors';
import {MODES} from '@/components/common/admin/venues/constants';
import services from '@/lib/services';
import type {AdminLeaderboard, AdminVenue, ListLeaderboardsData, StatMetric} from '@/lib/services/admin';
import {toErrorMessage} from '@/lib/services/core';
import {DeleteLeaderboardDialog} from './DeleteLeaderboardDialog';
import {LeaderboardDialog} from './LeaderboardDialog';
import {LeaderboardRow} from './LeaderboardRow';
import {PAGE_SIZE, STATUS_FILTERS, toRequest, type StatusFilter} from './constants';

/** 统计范围的候选项最多列出多少个地图 */
const VENUE_LIMIT = 100;

/**
 * 榜单管理: 搜索和状态筛选, 新建、编辑、启停、删除
 */
export function LeaderboardsMain() {
  const {user: me} = useAuth();
  const isAdmin = !!me?.is_admin;
  const [keywordInput, setKeywordInput] = useState('');
  const [keyword, setKeyword] = useState('');
  const [status, setStatus] = useState<StatusFilter>('all');
  const [current, setCurrent] = useState(1);
  const [data, setData] = useState<ListLeaderboardsData | null>(null);
  const [loading, setLoading] = useState(true);
  const [metrics, setMetrics] = useState<StatMetric[]>([]);
  const [venues, setVenues] = useState<AdminVenue[]>([]);

  // null: 关闭; "new": 新建; 榜单: 编辑
  const [editing, setEditing] = useState<AdminLeaderboard | 'new' | null>(null);
  const [deleting, setDeleting] = useState<AdminLeaderboard | null>(null);

  const load = useCallback(async () => {
    // 确认是管理员后再拉取, 否则 AdminGuard 会跳走
    if (!isAdmin) return;
    setLoading(true);
    try {
      setData(
          await services.admin.listLeaderboards({
            current,
            size: PAGE_SIZE,
            keyword: keyword || undefined,
            is_enabled: status === 'all' ? undefined : status === 'enabled',
          }),
      );
    } catch (err) {
      toast.error(toErrorMessage(err));
    } finally {
      setLoading(false);
    }
  }, [current, keyword, status, isAdmin]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 筛选条件变化时重新拉取
    void load();
  }, [load]);

  // 指标和地图用于展示名称和编辑时的候选项, 挂载时拉取一次
  useEffect(() => {
    if (!isAdmin) return;
    Promise.all([services.admin.listLeaderboardMetrics(), services.admin.listVenues({current: 1, size: VENUE_LIMIT})])
        .then(([metricData, venueData]) => {
          setMetrics(metricData.metrics);
          setVenues(venueData.venues);
        })
        .catch((err) => toast.error(toErrorMessage(err)));
  }, [isAdmin]);

  const scopes = useMemo<ScopeOption[]>(() => [
    {value: 'all', label: '全部对局'},
    ...MODES.map((m) => ({value: `mode:${m.value}`, label: `全部${m.label}对局`})),
    ...venues.flatMap((v) => [
      {value: `venue:${v.code}`, label: v.name},
      ...MODES.map((m) => ({value: `venue:${v.code}:${m.value}`, label: `${v.name} ${m.label}`})),
    ]),
  ], [venues]);

  const handleSearch = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setCurrent(1);
    setKeyword(keywordInput.trim());
  };

  // 启停也走修改接口, 需提交全部字段
  const toggleEnabled = async (board: AdminLeaderboard) => {
    try {
      await services.admin.updateLeaderboard(board.id, {...toRequest(board), is_enabled: !board.is_enabled});
      toast.success(board.is_enabled ? `已停用 ${board.name}` : `已启用 ${board.name}`);
      await load();
    } catch (err) {
      toast.error(toErrorMessage(err));
    }
  };

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between gap-2">
        <h1 className="text-2xl font-bold">榜单管理</h1>
        <Button onClick={() => setEditing('new')}>
          <Plus />
          新建榜单
        </Button>
      </div>

      <div className="flex flex-col gap-2 sm:flex-row">
        <form onSubmit={handleSearch} className="flex flex-1 gap-2">
          <Input
            placeholder="代码或名称"
            value={keywordInput}
            onChange={(e) => setKeywordInput(e.target.value)}
            className="sm:max-w-xs"
          />
          <Button type="submit" variant="secondary">
            <Search />
            搜索
          </Button>
        </form>
        <Select
          value={status}
          onValueChange={(value) => {
            setCurrent(1);
            setStatus(value as StatusFilter);
          }}
        >
          <SelectTrigger className="sm:w-32">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {STATUS_FILTERS.map((f) => (
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
              <TableHead>榜单</TableHead>
              <TableHead>指标</TableHead>
              <TableHead>上榜规则</TableHead>
              <TableHead className="text-right">排序</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>更新时间</TableHead>
              <TableHead className="text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading && !data ? (
              <TableSkeleton cols={7} />
            ) : data?.boards.length ? (
              data.boards.map((board) => (
                <LeaderboardRow
                  key={board.id}
                  board={board}
                  metricLabel={metrics.find((m) => m.key === board.metric)?.label ?? board.metric}
                  scopeLabel={scopes.find((s) => s.value === board.scope)?.label ?? board.scope}
                  onEdit={() => setEditing(board)}
                  onToggle={() => toggleEnabled(board)}
                  onDelete={() => setDeleting(board)}
                />
              ))
            ) : (
              <TableEmpty colSpan={7}>没有找到榜单</TableEmpty>
            )}
          </TableBody>
        </Table>
      </div>

      <Pagination current={current} size={PAGE_SIZE} total={data?.total ?? 0} onChange={setCurrent} />

      <LeaderboardDialog board={editing} metrics={metrics} scopes={scopes} onClose={() => setEditing(null)} onSaved={load} />
      <DeleteLeaderboardDialog board={deleting} onClose={() => setDeleting(null)} onSaved={load} />
    </div>
  );
}
