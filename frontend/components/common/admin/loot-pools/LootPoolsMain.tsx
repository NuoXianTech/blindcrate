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
import type {ListLootPoolsData, LootPoolListItem} from '@/lib/services/admin';
import {toErrorMessage} from '@/lib/services/core';
import {DeleteLootPoolDialog} from './DeleteLootPoolDialog';
import {EntriesDialog} from './EntriesDialog';
import {LootPoolDialog} from './LootPoolDialog';
import {LootPoolRow} from './LootPoolRow';
import {PAGE_SIZE, STATUS_FILTERS, toRequest, type StatusFilter} from './constants';

/**
 * 掉落池管理: 搜索和状态筛选, 新建、编辑、启停、删除, 编辑条目
 */
export function LootPoolsMain() {
  const {user: me} = useAuth();
  const isAdmin = !!me?.is_admin;
  const [keywordInput, setKeywordInput] = useState('');
  const [keyword, setKeyword] = useState('');
  const [status, setStatus] = useState<StatusFilter>('all');
  const [current, setCurrent] = useState(1);
  const [data, setData] = useState<ListLootPoolsData | null>(null);
  const [loading, setLoading] = useState(true);

  // null: 关闭; "new": 新建; 掉落池: 编辑
  const [editing, setEditing] = useState<LootPoolListItem | 'new' | null>(null);
  const [entriesOf, setEntriesOf] = useState<LootPoolListItem | null>(null);
  const [deleting, setDeleting] = useState<LootPoolListItem | null>(null);

  const load = useCallback(async () => {
    // 确认是管理员后再拉取, 否则 AdminGuard 会跳走
    if (!isAdmin) return;
    setLoading(true);
    try {
      setData(
          await services.admin.listLootPools({
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

  const handleSearch = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setCurrent(1);
    setKeyword(keywordInput.trim());
  };

  // 启停也走修改接口, 需提交全部字段
  const toggleEnabled = async (pool: LootPoolListItem) => {
    try {
      await services.admin.updateLootPool(pool.id, {...toRequest(pool), is_enabled: !pool.is_enabled});
      toast.success(pool.is_enabled ? `已停用 ${pool.name}` : `已启用 ${pool.name}`);
      await load();
    } catch (err) {
      toast.error(toErrorMessage(err));
    }
  };

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between gap-2">
        <h1 className="text-2xl font-bold">掉落池</h1>
        <Button onClick={() => setEditing('new')}>
          <Plus />
          新建掉落池
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
              <TableHead>掉落池</TableHead>
              <TableHead className="text-right">条目数</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>更新时间</TableHead>
              <TableHead className="text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading && !data ? (
              <TableSkeleton cols={5} />
            ) : data?.pools.length ? (
              data.pools.map((pool) => (
                <LootPoolRow
                  key={pool.id}
                  pool={pool}
                  onEntries={() => setEntriesOf(pool)}
                  onEdit={() => setEditing(pool)}
                  onToggle={() => toggleEnabled(pool)}
                  onDelete={() => setDeleting(pool)}
                />
              ))
            ) : (
              <TableEmpty colSpan={5}>没有找到掉落池</TableEmpty>
            )}
          </TableBody>
        </Table>
      </div>

      <Pagination current={current} size={PAGE_SIZE} total={data?.total ?? 0} onChange={setCurrent} />

      <LootPoolDialog pool={editing} onClose={() => setEditing(null)} onSaved={load} />
      <EntriesDialog pool={entriesOf} onClose={() => setEntriesOf(null)} onSaved={load} />
      <DeleteLootPoolDialog pool={deleting} onClose={() => setDeleting(null)} onSaved={load} />
    </div>
  );
}
