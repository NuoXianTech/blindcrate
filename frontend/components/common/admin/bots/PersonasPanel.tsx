'use client';

import {useCallback, useEffect, useState} from 'react';
import {Copy, LoaderCircle, Pencil, Plus, Search, Trash2} from 'lucide-react';
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
import {Button} from '@/components/ui/button';
import {Input} from '@/components/ui/input';
import {Table, TableBody, TableCell, TableHead, TableHeader, TableRow} from '@/components/ui/table';
import {Pagination, TableEmpty, TableSkeleton} from '@/components/common/layout';
import {useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';
import type {BotPersona, BotStrategy, ListBotPersonasData} from '@/lib/services/bot';
import {toErrorMessage} from '@/lib/services/core';
import {formatDateTime} from '@/lib/utils';
import {PersonaDialog, type PersonaTarget} from './PersonaDialog';
import {PAGE_SIZE} from './constants';

/**
 * 人机风格: 搜索, 新建、编辑、复制、删除; 被地图引用的风格不能删除
 */
export function PersonasPanel() {
  const [keywordInput, setKeywordInput] = useState('');
  const [keyword, setKeyword] = useState('');
  const [current, setCurrent] = useState(1);
  const [data, setData] = useState<ListBotPersonasData | null>(null);
  const [loading, setLoading] = useState(true);
  const [strategies, setStrategies] = useState<BotStrategy[] | null>(null);
  const [editing, setEditing] = useState<PersonaTarget | null>(null);
  const [deleting, setDeleting] = useState<BotPersona | null>(null);
  const {pending, submit} = useSubmit();

  const load = useCallback(async () => {
    setLoading(true);
    try {
      setData(await services.botAdmin.listPersonas({current, size: PAGE_SIZE, keyword: keyword || undefined}));
    } catch (err) {
      toast.error(toErrorMessage(err));
    } finally {
      setLoading(false);
    }
  }, [current, keyword]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 筛选条件变化时重新拉取
    void load();
  }, [load]);

  // 决策策略由后端注册, 挂载时拉取一次
  useEffect(() => {
    services.botAdmin.listStrategies()
        .then(setStrategies)
        .catch((err) => toast.error(toErrorMessage(err)));
  }, []);

  const handleSearch = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setCurrent(1);
    setKeyword(keywordInput.trim());
  };

  const handleDelete = submit(async () => {
    if (!deleting) return;
    await services.botAdmin.deletePersona(deleting.id);
    toast.success(`已删除 ${deleting.name}`);
    setDeleting(null);
    await load();
  });

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
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
        <Button onClick={() => setEditing({mode: 'new'})}>
          <Plus />
          新建风格
        </Button>
      </div>

      <div className="rounded-xl border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>风格</TableHead>
              <TableHead>决策策略</TableHead>
              <TableHead>更新时间</TableHead>
              <TableHead className="text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading && !data ? (
              <TableSkeleton cols={4} />
            ) : data?.personas.length ? (
              data.personas.map((p) => {
                const strategy = strategies?.find((s) => s.code === p.strategy);
                return (
                  <TableRow key={p.id}>
                    <TableCell>
                      <div className="flex min-w-0 flex-col">
                        <span className="font-medium">{p.name}</span>
                        <span className="max-w-80 truncate text-xs text-muted-foreground" title={p.description || undefined}>
                          <span className="font-mono">{p.code}</span>
                          {p.description && ` · ${p.description}`}
                        </span>
                      </div>
                    </TableCell>
                    <TableCell>
                      {strategy ? (
                        strategy.label
                      ) : strategies ? (
                        <span className="text-destructive" title="策略已不存在, 地图不能再选这个风格">
                          未知策略 <span className="font-mono">{p.strategy}</span>
                        </span>
                      ) : (
                        <span className="font-mono">{p.strategy}</span>
                      )}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground tabular-nums">{formatDateTime(p.updated_at)}</TableCell>
                    <TableCell>
                      <div className="flex justify-end gap-1">
                        <Button variant="ghost" size="icon" title="编辑" aria-label="编辑" onClick={() => setEditing({mode: 'edit', persona: p})}>
                          <Pencil />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          title="复制为新风格"
                          aria-label="复制为新风格"
                          onClick={() => setEditing({mode: 'copy', persona: p})}
                        >
                          <Copy />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          title="删除"
                          aria-label="删除"
                          className="hover:text-destructive"
                          onClick={() => setDeleting(p)}
                        >
                          <Trash2 />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                );
              })
            ) : (
              <TableEmpty colSpan={4}>没有找到风格</TableEmpty>
            )}
          </TableBody>
        </Table>
      </div>

      <Pagination current={current} size={PAGE_SIZE} total={data?.total ?? 0} onChange={setCurrent} />

      <PersonaDialog target={editing} strategies={strategies} onClose={() => setEditing(null)} onSaved={load} />

      <AlertDialog open={!!deleting} onOpenChange={(open) => !open && !pending && setDeleting(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>删除 {deleting?.name}</AlertDialogTitle>
            <AlertDialogDescription>
              被地图引用时不能删除, 需先在地图的人机页签中移除; 进行中的对局使用开局时的快照, 不受影响。代码 {deleting?.code} 之后也不能再用。
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
