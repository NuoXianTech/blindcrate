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
import type {AdminShopItem, AdminVenue, ConditionType, ListShopItemsData} from '@/lib/services/admin';
import {toErrorMessage} from '@/lib/services/core';
import {DeleteShopItemDialog} from './DeleteShopItemDialog';
import {ShopItemDialog, type ShopItemOptions, type ShopItemTarget} from './ShopItemDialog';
import {ShopItemRow} from './ShopItemRow';
import {ALL, PAGE_SIZE, STATUS_FILTERS, VENUE_LIMIT, toRequest, type StatusFilter} from './constants';

/**
 * 商品管理: 搜索、按页签和上架状态筛选, 新建、编辑、复制、上下架、删除
 */
export function ShopItemsMain() {
  const {user: me} = useAuth();
  const isAdmin = !!me?.is_admin;
  const [keywordInput, setKeywordInput] = useState('');
  const [keyword, setKeyword] = useState('');
  const [tab, setTab] = useState(ALL);
  const [status, setStatus] = useState<StatusFilter>('all');
  const [current, setCurrent] = useState(1);
  const [data, setData] = useState<ListShopItemsData | null>(null);
  // 列表加载的时间, 用于判断销售状态
  const [loadedAt, setLoadedAt] = useState(0);
  const [loading, setLoading] = useState(true);

  const catalog = useAdminCatalog();
  const [types, setTypes] = useState<ConditionType[] | null>(null);
  const [venues, setVenues] = useState<AdminVenue[] | null>(null);

  const [editing, setEditing] = useState<ShopItemTarget | null>(null);
  const [deleting, setDeleting] = useState<AdminShopItem | null>(null);

  const load = useCallback(async () => {
    // 确认是管理员后再拉取, 否则 AdminGuard 会跳走
    if (!isAdmin) return;
    setLoading(true);
    try {
      setData(
          await services.admin.listShopItems({
            current,
            size: PAGE_SIZE,
            keyword: keyword || undefined,
            tab: tab === ALL ? undefined : tab,
            is_enabled: status === 'all' ? undefined : status === 'enabled',
          }),
      );
      setLoadedAt(Date.now());
    } catch (err) {
      toast.error(toErrorMessage(err));
    } finally {
      setLoading(false);
    }
  }, [current, keyword, tab, status, isAdmin]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 筛选条件变化时重新拉取
    void load();
  }, [load]);

  // 条件类型和地图只在编辑购买条件时用到, 挂载时拉取一次
  useEffect(() => {
    if (!isAdmin) return;
    Promise.all([services.admin.listConditionTypes(), services.admin.listVenues({current: 1, size: VENUE_LIMIT})])
        .then(([typeData, venueData]) => {
          setTypes(typeData.types);
          setVenues(venueData.venues);
        })
        .catch((err) => toast.error(toErrorMessage(err)));
  }, [isAdmin]);

  const tabs = data?.tabs;
  const options = useMemo<ShopItemOptions>(() => ({
    items: catalog.items,
    currencies: catalog.currencies,
    types: types ?? [],
    scopes: scopeOptions(venues ?? []),
    tabs: tabs ?? [],
    loaded: catalog.loaded && types !== null && venues !== null,
  }), [catalog.items, catalog.currencies, catalog.loaded, types, venues, tabs]);

  const handleSearch = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setCurrent(1);
    setKeyword(keywordInput.trim());
  };

  // 上下架也走修改接口, 需提交全部字段
  const toggleEnabled = async (item: AdminShopItem) => {
    try {
      await services.admin.updateShopItem(item.id, {...toRequest(item), is_enabled: !item.is_enabled});
      toast.success(item.is_enabled ? `已下架 ${item.name}` : `已上架 ${item.name}`);
      await load();
    } catch (err) {
      toast.error(toErrorMessage(err));
    }
  };

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between gap-2">
        <h1 className="text-2xl font-bold">商品管理</h1>
        <Button onClick={() => setEditing({mode: 'new'})}>
          <Plus />
          新建商品
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
          value={tab}
          onValueChange={(value) => {
            setCurrent(1);
            setTab(value);
          }}
        >
          <SelectTrigger className="sm:w-32" aria-label="页签">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL}>全部页签</SelectItem>
            {/* 筛选中的页签可能已不在列表中, 仍保留选项 */}
            {tab !== ALL && !tabs?.includes(tab) && <SelectItem value={tab}>{tab}</SelectItem>}
            {tabs?.map((t) => (
              <SelectItem key={t} value={t}>
                {t}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select
          value={status}
          onValueChange={(value) => {
            setCurrent(1);
            setStatus(value as StatusFilter);
          }}
        >
          <SelectTrigger className="sm:w-32" aria-label="上架状态">
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
              <TableHead>商品</TableHead>
              <TableHead>页签</TableHead>
              <TableHead>内容</TableHead>
              <TableHead>价格</TableHead>
              <TableHead className="text-right">销量</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>上架时间</TableHead>
              <TableHead className="text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading && !data ? (
              <TableSkeleton cols={8} />
            ) : data?.items.length ? (
              data.items.map((item) => (
                <ShopItemRow
                  key={item.id}
                  item={item}
                  items={catalog.items}
                  currencies={catalog.currencies}
                  now={loadedAt}
                  onEdit={() => setEditing({mode: 'edit', item})}
                  onCopy={() => setEditing({mode: 'copy', item})}
                  onToggle={() => toggleEnabled(item)}
                  onDelete={() => setDeleting(item)}
                />
              ))
            ) : (
              <TableEmpty colSpan={8}>没有找到商品</TableEmpty>
            )}
          </TableBody>
        </Table>
      </div>

      <Pagination current={current} size={PAGE_SIZE} total={data?.total ?? 0} onChange={setCurrent} />

      <ShopItemDialog target={editing} options={options} onClose={() => setEditing(null)} onSaved={load} />
      <DeleteShopItemDialog item={deleting} onClose={() => setDeleting(null)} onSaved={load} />
    </div>
  );
}
