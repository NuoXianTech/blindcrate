'use client';

import {useCallback, useEffect, useState} from 'react';
import {LoaderCircle, Pencil, Plus, Power, Search, Trash2} from 'lucide-react';
import {toast} from 'sonner';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import {Badge} from '@/components/ui/badge';
import {Button} from '@/components/ui/button';
import {Input} from '@/components/ui/input';
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select';
import {Table, TableBody, TableCell, TableHead, TableHeader, TableRow} from '@/components/ui/table';
import {Pagination, TableEmpty, TableSkeleton} from '@/components/common/layout';
import {useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';
import type {BotName, ListBotNamesData} from '@/lib/services/bot';
import {toErrorMessage} from '@/lib/services/core';
import {formatDateTime} from '@/lib/utils';
import {CreateNamesDialog} from './CreateNamesDialog';
import {NameDialog} from './NameDialog';
import {PAGE_SIZE, STATUS_FILTERS, type StatusFilter} from './constants';

/**
 * 昵称库: 搜索和状态筛选, 批量添加、修改、启停、删除
 * 开局时从启用的昵称中随机选取, 同一局不重复, 不够时用 "玩家" 加四位数字补齐
 */
export function NamesPanel() {
  const [keywordInput, setKeywordInput] = useState('');
  const [keyword, setKeyword] = useState('');
  const [status, setStatus] = useState<StatusFilter>('all');
  const [current, setCurrent] = useState(1);
  const [data, setData] = useState<ListBotNamesData | null>(null);
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);
  const [editing, setEditing] = useState<BotName | null>(null);
  const [deleting, setDeleting] = useState<BotName | null>(null);
  const {pending, submit} = useSubmit();

  const load = useCallback(async () => {
    setLoading(true);
    try {
      setData(
          await services.botAdmin.listNames({
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
  }, [current, keyword, status]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 筛选条件变化时重新拉取
    void load();
  }, [load]);

  const handleSearch = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setCurrent(1);
    setKeyword(keywordInput.trim());
  };

  const toggleEnabled = async (name: BotName) => {
    try {
      await services.botAdmin.updateName(name.id, {name: name.name, is_enabled: !name.is_enabled});
      toast.success(name.is_enabled ? `已停用 ${name.name}` : `已启用 ${name.name}`);
      await load();
    } catch (err) {
      toast.error(toErrorMessage(err));
    }
  };

  const handleDelete = submit(async () => {
    if (!deleting) return;
    await services.botAdmin.deleteName(deleting.id);
    toast.success(`已删除 ${deleting.name}`);
    setDeleting(null);
    await load();
  });

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
        <form onSubmit={handleSearch} className="flex flex-1 gap-2">
          <Input
            placeholder="按昵称搜索"
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
        <Button onClick={() => setCreating(true)}>
          <Plus />
          添加昵称
        </Button>
      </div>

      <div className="rounded-xl border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>昵称</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>更新时间</TableHead>
              <TableHead className="text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading && !data ? (
              <TableSkeleton cols={4} />
            ) : data?.names.length ? (
              data.names.map((n) => (
                <TableRow key={n.id}>
                  <TableCell className="font-medium">{n.name}</TableCell>
                  <TableCell>
                    <Badge variant={n.is_enabled ? 'secondary' : 'outline'}>{n.is_enabled ? '已启用' : '已停用'}</Badge>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground tabular-nums">{formatDateTime(n.updated_at)}</TableCell>
                  <TableCell>
                    <div className="flex justify-end gap-1">
                      <Button variant="ghost" size="icon" title="编辑" aria-label="编辑" onClick={() => setEditing(n)}>
                        <Pencil />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        title={n.is_enabled ? '停用' : '启用'}
                        aria-label={n.is_enabled ? '停用' : '启用'}
                        className={n.is_enabled ? undefined : 'text-destructive'}
                        onClick={() => toggleEnabled(n)}
                      >
                        <Power />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        title="删除"
                        aria-label="删除"
                        className="hover:text-destructive"
                        onClick={() => setDeleting(n)}
                      >
                        <Trash2 />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))
            ) : (
              <TableEmpty colSpan={4}>没有找到昵称</TableEmpty>
            )}
          </TableBody>
        </Table>
      </div>

      <Pagination current={current} size={PAGE_SIZE} total={data?.total ?? 0} onChange={setCurrent} />

      <CreateNamesDialog open={creating} onClose={() => setCreating(false)} onSaved={load} />
      <NameDialog name={editing} onClose={() => setEditing(null)} onSaved={load} />

      <AlertDialog open={!!deleting} onOpenChange={(open) => !open && !pending && setDeleting(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>删除 {deleting?.name}</AlertDialogTitle>
            <AlertDialogDescription>
              删除后开局时不再选取, 已开局的对局中保存的是昵称本身, 不受影响。只是暂时不想用的话, 停用即可。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={pending}>取消</AlertDialogCancel>
            <AlertDialogAction variant="destructive" disabled={pending} onClick={handleDelete}>
              {pending && <LoaderCircle className="animate-spin" />}
              删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
