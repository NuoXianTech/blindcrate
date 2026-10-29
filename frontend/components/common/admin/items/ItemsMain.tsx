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
import {useQualities} from '@/components/common/site';
import services from '@/lib/services';
import type {AdminItem, Currency, ItemCategory} from '@/lib/services/admin';
import {toErrorMessage} from '@/lib/services/core';
import {DeleteItemDialog} from './DeleteItemDialog';
import {ItemDialog} from './ItemDialog';
import {ItemRow} from './ItemRow';
import {PAGE_SIZE, STATUS_FILTERS, toRequest, type StatusFilter} from './constants';

/**
 * 物品管理: 搜索、品质、类别和状态筛选, 新建、编辑、上下架、删除
 */
export function ItemsMain() {
  const {user: me} = useAuth();
  const {qualities} = useQualities();
  const isAdmin = !!me?.is_admin;
  const [keywordInput, setKeywordInput] = useState('');
  const [keyword, setKeyword] = useState('');
  const [quality, setQuality] = useState('all');
  const [category, setCategory] = useState('all');
  const [status, setStatus] = useState<StatusFilter>('all');
  const [current, setCurrent] = useState(1);
  const [data, setData] = useState<{ total: number; items: AdminItem[] } | null>(null);
  const [loading, setLoading] = useState(true);
  const [currencies, setCurrencies] = useState<Currency[]>([]);
  const [categories, setCategories] = useState<ItemCategory[]>([]);

  // null: 关闭; "new": 新建; 物品: 编辑
  const [editing, setEditing] = useState<AdminItem | 'new' | null>(null);
  const [deleting, setDeleting] = useState<AdminItem | null>(null);

  useEffect(() => {
    if (!isAdmin) return;
    Promise.all([services.admin.listCurrencies(), services.admin.listItemCategories()])
        .then(([currencyData, categoryData]) => {
          setCurrencies(currencyData.currencies);
          setCategories(categoryData.categories);
        })
        .catch((err) => toast.error(toErrorMessage(err)));
  }, [isAdmin]);

  const load = useCallback(async () => {
    // 确认是管理员后再拉取, 否则 AdminGuard 会跳走
    if (!isAdmin) return;
    setLoading(true);
    try {
      setData(
          await services.admin.listItems({
            current,
            size: PAGE_SIZE,
            keyword: keyword || undefined,
            quality: quality === 'all' ? undefined : Number(quality),
            category: category === 'all' ? undefined : category,
            is_enabled: status === 'all' ? undefined : status === 'enabled',
          }),
      );
    } catch (err) {
      toast.error(toErrorMessage(err));
    } finally {
      setLoading(false);
    }
  }, [current, keyword, quality, category, status, isAdmin]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 筛选条件变化时重新拉取
    void load();
  }, [load]);

  const handleSearch = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setCurrent(1);
    setKeyword(keywordInput.trim());
  };

  // 上下架也走修改接口, 需提交全部字段
  const toggleEnabled = async (item: AdminItem) => {
    try {
      await services.admin.updateItem(item.id, {...toRequest(item), is_enabled: !item.is_enabled});
      toast.success(item.is_enabled ? `已下架 ${item.name}` : `已上架 ${item.name}`);
      await load();
    } catch (err) {
      toast.error(toErrorMessage(err));
    }
  };

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between gap-2">
        <h1 className="text-2xl font-bold">物品管理</h1>
        <Button onClick={() => setEditing('new')}>
          <Plus />
          新建物品
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
          value={quality}
          onValueChange={(value) => {
            setCurrent(1);
            setQuality(value);
          }}
        >
          <SelectTrigger className="sm:w-32">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部品质</SelectItem>
            {qualities.map((q) => (
              <SelectItem key={q.level} value={String(q.level)}>
                <span className={q.text}>{q.name}</span>
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select
          value={category}
          onValueChange={(value) => {
            setCurrent(1);
            setCategory(value);
          }}
        >
          <SelectTrigger className="sm:w-32">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部类别</SelectItem>
            {categories.map((c) => (
              <SelectItem key={c.code} value={c.code}>
                {c.label}
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
              <TableHead>物品</TableHead>
              <TableHead>品质</TableHead>
              <TableHead>类别</TableHead>
              <TableHead className="text-right">回收价</TableHead>
              <TableHead>尺寸</TableHead>
              <TableHead className="text-right">持有人数</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>更新时间</TableHead>
              <TableHead className="text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading && !data ? (
              <TableSkeleton cols={9} />
            ) : data?.items.length ? (
              data.items.map((item) => (
                <ItemRow
                  key={item.id}
                  item={item}
                  currency={currencies.find((c) => c.code === item.sell_currency)}
                  category={categories.find((c) => c.code === item.category)}
                  onEdit={() => setEditing(item)}
                  onToggle={() => toggleEnabled(item)}
                  onDelete={() => setDeleting(item)}
                />
              ))
            ) : (
              <TableEmpty colSpan={9}>没有找到物品</TableEmpty>
            )}
          </TableBody>
        </Table>
      </div>

      <Pagination current={current} size={PAGE_SIZE} total={data?.total ?? 0} onChange={setCurrent} />

      <ItemDialog
        item={editing}
        currencies={currencies}
        categories={categories}
        onClose={() => setEditing(null)}
        onSaved={load}
      />
      <DeleteItemDialog item={deleting} onClose={() => setDeleting(null)} onSaved={load} />
    </div>
  );
}
