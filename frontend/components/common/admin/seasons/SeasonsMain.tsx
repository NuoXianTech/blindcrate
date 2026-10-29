'use client';

import {useCallback, useEffect, useState} from 'react';
import {Plus, Search} from 'lucide-react';
import {toast} from 'sonner';
import {Button} from '@/components/ui/button';
import {Input} from '@/components/ui/input';
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select';
import {Table, TableBody, TableHead, TableHeader, TableRow} from '@/components/ui/table';
import {useAuth} from '@/components/common/auth';
import {Pagination, TableEmpty, TableSkeleton} from '@/components/common/layout';
import services from '@/lib/services';
import type {ListSeasonsData} from '@/lib/services/admin';
import {toErrorMessage} from '@/lib/services/core';
import type {Season} from '@/lib/services/season';
import {SeasonActionDialog, type SeasonAction} from './SeasonActionDialog';
import {SeasonDialog} from './SeasonDialog';
import {SeasonRow} from './SeasonRow';
import {PAGE_SIZE, STATUS_FILTERS, type StatusFilter} from './constants';

/**
 * 赛季管理: 搜索和状态筛选, 新建、编辑、提前结束、删除
 */
export function SeasonsMain() {
  const {user: me} = useAuth();
  const isAdmin = !!me?.is_admin;
  const [keywordInput, setKeywordInput] = useState('');
  const [keyword, setKeyword] = useState('');
  const [status, setStatus] = useState<StatusFilter>('all');
  const [current, setCurrent] = useState(1);
  const [data, setData] = useState<ListSeasonsData | null>(null);
  const [loading, setLoading] = useState(true);

  // null: 关闭; "new": 新建; 赛季: 编辑
  const [editing, setEditing] = useState<Season | 'new' | null>(null);
  const [acting, setActing] = useState<{ season: Season; action: SeasonAction } | null>(null);

  const load = useCallback(async () => {
    // 确认是管理员后再拉取, 否则 AdminGuard 会跳走
    if (!isAdmin) return;
    setLoading(true);
    try {
      setData(
          await services.admin.listSeasons({
            current,
            size: PAGE_SIZE,
            keyword: keyword || undefined,
            status: status === 'all' ? undefined : status,
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

  const handleSearch = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setCurrent(1);
    setKeyword(keywordInput.trim());
  };

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between gap-2">
        <h1 className="text-2xl font-bold">赛季管理</h1>
        <Button onClick={() => setEditing('new')}>
          <Plus />
          新建赛季
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
              <TableHead>赛季</TableHead>
              <TableHead>开始时间</TableHead>
              <TableHead>结束时间</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>更新时间</TableHead>
              <TableHead className="text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading && !data ? (
              <TableSkeleton cols={6} />
            ) : data?.seasons.length ? (
              data.seasons.map((season) => (
                <SeasonRow
                  key={season.id}
                  season={season}
                  onEdit={() => setEditing(season)}
                  onEnd={() => setActing({season, action: 'end'})}
                  onDelete={() => setActing({season, action: 'delete'})}
                />
              ))
            ) : (
              <TableEmpty colSpan={6}>没有找到赛季</TableEmpty>
            )}
          </TableBody>
        </Table>
      </div>

      <Pagination current={current} size={PAGE_SIZE} total={data?.total ?? 0} onChange={setCurrent} />

      <SeasonDialog season={editing} onClose={() => setEditing(null)} onSaved={load} />
      <SeasonActionDialog target={acting} onClose={() => setActing(null)} onSaved={load} />
    </div>
  );
}
