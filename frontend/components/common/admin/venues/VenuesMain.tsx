'use client';

import {useCallback, useEffect, useMemo, useState} from 'react';
import {Plus, Search} from 'lucide-react';
import {toast} from 'sonner';
import {Button} from '@/components/ui/button';
import {Input} from '@/components/ui/input';
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select';
import {Table, TableBody, TableHead, TableHeader, TableRow} from '@/components/ui/table';
import {scopeOptions} from '@/components/common/admin/editors';
import {useAuth} from '@/components/common/auth';
import {Pagination, TableEmpty, TableSkeleton} from '@/components/common/layout';
import {useAdminCatalog} from '@/hooks/use-admin-catalog';
import services from '@/lib/services';
import type {AdminVenue, ConditionType, ItemEffect, ListVenuesData, LootPoolListItem} from '@/lib/services/admin';
import {toErrorMessage} from '@/lib/services/core';
import {DeleteVenueDialog} from './DeleteVenueDialog';
import {VenueDialog, type VenueOptions} from './VenueDialog';
import {VenueRow} from './VenueRow';
import {PAGE_SIZE, STATUS_FILTERS, toRequest, type StatusFilter} from './constants';

/** 档位可选的掉落池数量上限, 与列表接口的分页上限一致 */
const POOL_LIMIT = 100;

/**
 * 地图管理: 搜索和状态筛选, 新建、编辑、启停、删除, 编辑时可以模拟仓库
 */
export function VenuesMain() {
  const {user: me} = useAuth();
  const isAdmin = !!me?.is_admin;
  const [keywordInput, setKeywordInput] = useState('');
  const [keyword, setKeyword] = useState('');
  const [status, setStatus] = useState<StatusFilter>('all');
  const [current, setCurrent] = useState(1);
  const [data, setData] = useState<ListVenuesData | null>(null);
  const [loading, setLoading] = useState(true);

  const catalog = useAdminCatalog();
  const [pools, setPools] = useState<LootPoolListItem[] | null>(null);
  const [types, setTypes] = useState<ConditionType[] | null>(null);
  const [effects, setEffects] = useState<ItemEffect[] | null>(null);

  // null: 关闭; "new": 新建; 地图: 编辑
  const [editing, setEditing] = useState<AdminVenue | 'new' | null>(null);
  const [deleting, setDeleting] = useState<AdminVenue | null>(null);

  const load = useCallback(async () => {
    // 确认是管理员后再拉取, 否则 AdminGuard 会跳走
    if (!isAdmin) return;
    setLoading(true);
    try {
      setData(
          await services.admin.listVenues({
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

  // 掉落池、条件类型和使用效果只在编辑时用到, 挂载时拉取一次
  useEffect(() => {
    if (!isAdmin) return;
    Promise.all([
      services.admin.listLootPools({current: 1, size: POOL_LIMIT}),
      services.admin.listConditionTypes(),
      services.admin.listItemEffects(),
    ])
        .then(([poolData, typeData, effectData]) => {
          setPools(poolData.pools);
          setTypes(typeData.types);
          setEffects(effectData.effects);
        })
        .catch((err) => toast.error(toErrorMessage(err)));
  }, [isAdmin]);

  const venues = data?.venues;
  const options = useMemo<VenueOptions>(() => {
    return {
      items: catalog.items,
      currencies: catalog.currencies,
      pools: pools ?? [],
      types: types ?? [],
      scopes: scopeOptions(venues ?? []),
      effects: effects ?? [],
      loaded: catalog.loaded && pools !== null && types !== null && effects !== null,
    };
  }, [catalog.items, catalog.currencies, catalog.loaded, pools, types, effects, venues]);

  const handleSearch = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setCurrent(1);
    setKeyword(keywordInput.trim());
  };

  // 启停也走修改接口, 需提交全部字段, 后端会重新模拟
  const toggleEnabled = async (venue: AdminVenue) => {
    try {
      await services.admin.updateVenue(venue.id, {...toRequest(venue), is_enabled: !venue.is_enabled});
      toast.success(venue.is_enabled ? `已停用 ${venue.name}` : `已启用 ${venue.name}`);
      await load();
    } catch (err) {
      toast.error(toErrorMessage(err));
    }
  };

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between gap-2">
        <h1 className="text-2xl font-bold">地图管理</h1>
        <Button onClick={() => setEditing('new')}>
          <Plus />
          新建地图
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
              <TableHead>地图</TableHead>
              <TableHead>模式</TableHead>
              <TableHead>入场代价</TableHead>
              <TableHead className="text-right">价值中位数</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>更新时间</TableHead>
              <TableHead className="text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading && !data ? (
              <TableSkeleton cols={7} />
            ) : venues?.length ? (
              venues.map((venue) => (
                <VenueRow
                  key={venue.id}
                  venue={venue}
                  items={catalog.items}
                  currencies={catalog.currencies}
                  onEdit={() => setEditing(venue)}
                  onToggle={() => toggleEnabled(venue)}
                  onDelete={() => setDeleting(venue)}
                />
              ))
            ) : (
              <TableEmpty colSpan={7}>没有找到地图</TableEmpty>
            )}
          </TableBody>
        </Table>
      </div>

      <Pagination current={current} size={PAGE_SIZE} total={data?.total ?? 0} onChange={setCurrent} />

      <VenueDialog venue={editing} options={options} onClose={() => setEditing(null)} onSaved={load} />
      <DeleteVenueDialog venue={deleting} onClose={() => setDeleting(null)} onSaved={load} />
    </div>
  );
}
