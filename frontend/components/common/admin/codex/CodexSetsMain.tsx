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
import {useAdminCatalog} from '@/hooks/use-admin-catalog';
import services from '@/lib/services';
import type {LootPoolListItem} from '@/lib/services/admin';
import type {AdminCodexSet, ListCodexSetsData} from '@/lib/services/codex';
import {toErrorMessage} from '@/lib/services/core';
import {CodexSetDialog, type CodexSetFormOptions, type CodexSetTarget} from './CodexSetDialog';
import {CodexSetRow} from './CodexSetRow';
import {DeleteCodexSetDialog} from './DeleteCodexSetDialog';
import {PAGE_SIZE, POOL_LIMIT, STATUS_FILTERS, memberStatus, toRequest, type StatusFilter} from './constants';

/**
 * 图鉴管理: 收集组的搜索、按启用状态筛选, 新建、编辑、复制、启停、删除
 * 图鉴收录全部上架的藏品, 不需要单独配置
 */
export function CodexSetsMain() {
  const {user: me} = useAuth();
  const isAdmin = !!me?.is_admin;
  const [keywordInput, setKeywordInput] = useState('');
  const [keyword, setKeyword] = useState('');
  const [status, setStatus] = useState<StatusFilter>('all');
  const [current, setCurrent] = useState(1);
  const [data, setData] = useState<ListCodexSetsData | null>(null);
  const [loading, setLoading] = useState(true);

  const catalog = useAdminCatalog();
  const [pools, setPools] = useState<LootPoolListItem[] | null>(null);

  const [editing, setEditing] = useState<CodexSetTarget | null>(null);
  const [deleting, setDeleting] = useState<AdminCodexSet | null>(null);

  const load = useCallback(async () => {
    // 确认是管理员后再拉取, 否则 AdminGuard 会跳走
    if (!isAdmin) return;
    setLoading(true);
    try {
      setData(
          await services.codexAdmin.listSets({
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

  // 掉落池只在编辑时用于批量添加藏品, 挂载时拉取一次
  useEffect(() => {
    if (!isAdmin) return;
    services.admin.listLootPools({current: 1, size: POOL_LIMIT})
        .then((poolData) => setPools(poolData.pools))
        .catch((err) => toast.error(toErrorMessage(err)));
  }, [isAdmin]);

  const options = useMemo<CodexSetFormOptions>(() => ({
    items: catalog.items,
    currencies: catalog.currencies,
    pools: pools ?? [],
    loaded: catalog.loaded && pools !== null,
  }), [catalog.items, catalog.currencies, catalog.loaded, pools]);

  // 物品目录加载完成后才能判断成员是否计入进度, 之前各行不显示不计入的数量
  const itemMap = useMemo(
      () => catalog.loaded ? new Map(catalog.items.map((i) => [i.id, i])) : null,
      [catalog.items, catalog.loaded],
  );

  const handleSearch = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setCurrent(1);
    setKeyword(keywordInput.trim());
  };

  // 启停也走修改接口, 需提交全部字段
  const toggleEnabled = async (set: AdminCodexSet) => {
    try {
      await services.codexAdmin.updateSet(set.id, {...toRequest(set), is_enabled: !set.is_enabled});
      toast.success(set.is_enabled ? `已停用 ${set.name}` : `已启用 ${set.name}`);
      await load();
    } catch (err) {
      toast.error(toErrorMessage(err));
    }
  };

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between gap-2">
        <div className="flex flex-col gap-0.5">
          <h1 className="text-2xl font-bold">图鉴管理</h1>
          <p className="text-sm text-muted-foreground">图鉴收录全部上架的藏品; 这里配置收集组, 集齐后玩家可以领取奖励</p>
        </div>
        <Button onClick={() => setEditing({mode: 'new'})}>
          <Plus />
          新建收集组
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
          <SelectTrigger className="sm:w-32" aria-label="启用状态">
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
              <TableHead>收集组</TableHead>
              <TableHead>藏品</TableHead>
              <TableHead>奖励</TableHead>
              <TableHead className="text-right">排序</TableHead>
              <TableHead>状态</TableHead>
              <TableHead className="text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading && !data ? (
              <TableSkeleton cols={6} />
            ) : data?.sets.length ? (
              data.sets.map((s) => (
                <CodexSetRow
                  key={s.id}
                  set={s}
                  items={catalog.items}
                  currencies={catalog.currencies}
                  excluded={itemMap && s.item_ids.filter((id) => memberStatus(itemMap.get(id)) !== 'counted').length}
                  onEdit={() => setEditing({mode: 'edit', set: s})}
                  onCopy={() => setEditing({mode: 'copy', set: s})}
                  onToggle={() => toggleEnabled(s)}
                  onDelete={() => setDeleting(s)}
                />
              ))
            ) : (
              <TableEmpty colSpan={6}>没有找到收集组</TableEmpty>
            )}
          </TableBody>
        </Table>
      </div>

      <Pagination current={current} size={PAGE_SIZE} total={data?.total ?? 0} onChange={setCurrent} />

      <CodexSetDialog target={editing} options={options} onClose={() => setEditing(null)} onSaved={load} />
      <DeleteCodexSetDialog set={deleting} onClose={() => setDeleting(null)} onSaved={load} />
    </div>
  );
}
