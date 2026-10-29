'use client';

import {useCallback, useEffect, useState} from 'react';
import {Eye, LoaderCircle} from 'lucide-react';
import {toast} from 'sonner';
import {Button} from '@/components/ui/button';
import {Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle} from '@/components/ui/dialog';
import {Table, TableBody, TableCell, TableHead, TableHeader, TableRow} from '@/components/ui/table';
import {useAuth} from '@/components/common/auth';
import {Pagination, TableEmpty, TableSkeleton} from '@/components/common/layout';
import services from '@/lib/services';
import type {ListPackImportsData, PackImport} from '@/lib/services/admin';
import {toErrorMessage} from '@/lib/services/core';
import {formatDateTime} from '@/lib/utils';
import {PackEntriesTable} from './PackEntriesTable';
import {PAGE_SIZE, formatSize} from './constants';

/**
 * 导入记录: 从新到旧, 可以查看每一项的处理结果
 */
export function ImportHistory() {
  const isAdmin = !!useAuth().user?.is_admin;
  const [current, setCurrent] = useState(1);
  const [data, setData] = useState<ListPackImportsData | null>(null);
  const [loading, setLoading] = useState(true);
  const [viewing, setViewing] = useState<PackImport | null>(null);

  const load = useCallback(async () => {
    // 确认是管理员后再拉取, 否则 AdminGuard 会跳走
    if (!isAdmin) return;
    setLoading(true);
    try {
      setData(await services.admin.listPackImports({current, size: PAGE_SIZE}));
    } catch (err) {
      toast.error(toErrorMessage(err));
    } finally {
      setLoading(false);
    }
  }, [current, isAdmin]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 翻页时重新拉取
    void load();
  }, [load]);

  return (
    <div className="flex flex-col gap-4">
      <div className="rounded-xl border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>记录</TableHead>
              <TableHead>配置包</TableHead>
              <TableHead>管理员</TableHead>
              <TableHead>新增</TableHead>
              <TableHead>覆盖</TableHead>
              <TableHead>跳过</TableHead>
              <TableHead>大小</TableHead>
              <TableHead>时间</TableHead>
              <TableHead className="w-0" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading && !data ? (
              <TableSkeleton cols={9} />
            ) : data?.imports.length ? (
              data.imports.map((r) => (
                <TableRow key={r.id}>
                  <TableCell className="font-mono tabular-nums">#{r.id}</TableCell>
                  <TableCell>
                    <div className="flex flex-col">
                      <span>{r.name}</span>
                      {r.author && <span className="text-xs text-muted-foreground">{r.author}</span>}
                    </div>
                  </TableCell>
                  <TableCell className="font-mono tabular-nums">#{r.user_id}</TableCell>
                  <TableCell className="tabular-nums">{r.created}</TableCell>
                  <TableCell className="tabular-nums">{r.updated}</TableCell>
                  <TableCell className="text-muted-foreground tabular-nums">{r.skipped}</TableCell>
                  <TableCell className="text-muted-foreground tabular-nums">{formatSize(r.size)}</TableCell>
                  <TableCell className="text-muted-foreground tabular-nums">{formatDateTime(r.created_at)}</TableCell>
                  <TableCell>
                    <Button variant="ghost" size="icon" title="查看详情" aria-label="查看详情" onClick={() => setViewing(r)}>
                      <Eye />
                    </Button>
                  </TableCell>
                </TableRow>
              ))
            ) : (
              <TableEmpty colSpan={9}>还没有导入过配置包</TableEmpty>
            )}
          </TableBody>
        </Table>
      </div>

      <Pagination current={current} size={PAGE_SIZE} total={data?.total ?? 0} onChange={setCurrent} />

      <Dialog open={!!viewing} onOpenChange={(open) => !open && setViewing(null)}>
        <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-4xl">
          {viewing && <ImportDetail key={viewing.id} record={viewing} />}
        </DialogContent>
      </Dialog>
    </div>
  );
}

/**
 * 一次导入的详情, 默认只列出新增、覆盖和跳过的项
 */
function ImportDetail({record}: { record: PackImport }) {
  const [entries, setEntries] = useState<PackImport['entries']>();

  useEffect(() => {
    services.admin.getPackImport(record.id)
        .then((res) => setEntries(res.entries ?? []))
        .catch((err) => toast.error(toErrorMessage(err)));
  }, [record.id]);

  // 与已有配置相同的项没有写入, 不列出
  const changed = entries?.filter((e) => e.status !== 'same' || e.action !== 'overwrite') ?? [];

  return (
    <>
      <DialogHeader>
        <DialogTitle>导入记录 #{record.id} {record.name}</DialogTitle>
        <DialogDescription>
          {formatDateTime(record.created_at)} · 管理员 #{record.user_id} · 新增 {record.created} 项, 覆盖 {record.updated} 项, 跳过 {record.skipped} 项(含相同的项)
          <span className="mt-1 block font-mono text-xs break-all">SHA-256 {record.digest}</span>
        </DialogDescription>
      </DialogHeader>
      {entries ? (
        <PackEntriesTable entries={changed} empty="与已有配置全部相同, 没有写入任何配置" />
      ) : (
        <div className="flex justify-center py-8">
          <LoaderCircle className="animate-spin text-muted-foreground" />
        </div>
      )}
    </>
  );
}
