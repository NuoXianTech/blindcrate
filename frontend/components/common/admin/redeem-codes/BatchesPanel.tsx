'use client';

import {useCallback, useEffect, useState} from 'react';
import {CopyPlus, LoaderCircle, Pencil, Plus, Power, Search, Ticket, Trash2} from 'lucide-react';
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
import {bundleText} from '@/components/common/admin/editors';
import {Pagination, TableEmpty, TableSkeleton} from '@/components/common/layout';
import {useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';
import {toErrorMessage} from '@/lib/services/core';
import type {AdminRedeemBatch, ListRedeemBatchesData, RedeemBatch} from '@/lib/services/redeem';
import {formatDateTime} from '@/lib/utils';
import {BatchDialog, type BatchOptions, type BatchTarget} from './BatchDialog';
import {CodesDialog} from './CodesDialog';
import {ALL, MAX_CODE_INPUT, PAGE_SIZE, STATUS_FILTERS, STATUS_LABELS, batchStatus, toRequest, usesText, type StatusFilter} from './constants';

/**
 * 兑换码批次: 搜索、按状态筛选, 生成、查看兑换码、编辑、启用和停用、删除
 */
export function BatchesPanel({options}: { options: BatchOptions }) {
  const [keywordInput, setKeywordInput] = useState('');
  const [keyword, setKeyword] = useState('');
  const [status, setStatus] = useState<StatusFilter>(ALL);
  const [current, setCurrent] = useState(1);
  const [data, setData] = useState<ListRedeemBatchesData | null>(null);
  // 列表加载的时间, 用于判断是否已到期
  const [loadedAt, setLoadedAt] = useState(0);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState<BatchTarget | null>(null);
  const [viewing, setViewing] = useState<RedeemBatch | null>(null);
  const [deleting, setDeleting] = useState<AdminRedeemBatch | null>(null);
  const {pending, submit} = useSubmit();

  const load = useCallback(async () => {
    setLoading(true);
    try {
      setData(
          await services.redeemAdmin.listBatches({
            current,
            size: PAGE_SIZE,
            keyword: keyword || undefined,
            status: status === ALL ? undefined : status,
          }),
      );
      setLoadedAt(Date.now());
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

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    setCurrent(1);
    setKeyword(keywordInput.trim());
  };

  // 启用和停用也走修改接口, 需提交全部字段
  const toggleEnabled = async (batch: AdminRedeemBatch) => {
    try {
      await services.redeemAdmin.updateBatch(batch.id, {...toRequest(batch), is_enabled: !batch.is_enabled});
      toast.success(batch.is_enabled ? `已停用 ${batch.name}` : `已启用 ${batch.name}`);
      await load();
    } catch (err) {
      toast.error(toErrorMessage(err));
    }
  };

  const handleDelete = submit(async () => {
    if (!deleting) return;
    await services.redeemAdmin.deleteBatch(deleting.id);
    toast.success(`已删除 ${deleting.name}`);
    setDeleting(null);
    await load();
  });

  const {items, currencies} = options;

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex flex-wrap gap-2">
          <form onSubmit={handleSearch} className="flex gap-2">
            <Input
              className="w-64"
              placeholder="批次名称或完整的兑换码"
              maxLength={MAX_CODE_INPUT}
              value={keywordInput}
              onChange={(e) => setKeywordInput(e.target.value)}
            />
            <Button type="submit" variant="outline" size="icon" aria-label="搜索">
              <Search />
            </Button>
          </form>
          <Select
            value={status}
            onValueChange={(value) => {
              setCurrent(1);
              setStatus(value as StatusFilter);
            }}
          >
            <SelectTrigger className="w-32" aria-label="状态">
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
        <Button onClick={() => setEditing({mode: 'new'})}>
          <Plus />
          生成兑换码
        </Button>
      </div>

      <div className="rounded-xl border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>批次</TableHead>
              <TableHead>奖励</TableHead>
              <TableHead>使用限制</TableHead>
              <TableHead className="text-right">已兑换</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>到期时间</TableHead>
              <TableHead>生成</TableHead>
              <TableHead className="text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading && !data ? (
              <TableSkeleton cols={8} />
            ) : data?.batches.length ? (
              data.batches.map((b) => {
                const reward = bundleText(b.reward, items, currencies);
                const batchState = batchStatus(b, loadedAt);
                return (
                  <TableRow key={b.id}>
                    <TableCell className="max-w-48">
                      <div className="flex min-w-0 flex-col">
                        <span className="truncate font-medium" title={b.name}>
                          {b.name}
                        </span>
                        <span className="font-mono text-xs text-muted-foreground">#{b.id}</span>
                      </div>
                    </TableCell>
                    <TableCell className="max-w-48 truncate" title={reward.join(', ') || undefined}>
                      {reward.join(', ')}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      <div className="flex flex-col">
                        <span>{usesText(b.max_uses)}</span>
                        <span>{b.user_limit === 0 ? '每人不限个数' : `每人最多 ${b.user_limit} 个`}</span>
                        {b.requirements.length > 0 && <span>{b.requirements.length} 个使用条件</span>}
                      </div>
                    </TableCell>
                    <TableCell className="text-right font-mono tabular-nums">
                      <div className="flex flex-col">
                        <span title="已兑换的次数 / 一共可以兑换的次数">
                          {b.redeemed} / {b.max_uses === 0 ? '不限' : b.code_count * b.max_uses}
                        </span>
                        <span className="text-xs text-muted-foreground">{b.code_count} 个兑换码</span>
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant={batchState === 'active' ? 'secondary' : 'outline'}>{STATUS_LABELS[batchState]}</Badge>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground tabular-nums">
                      {b.expires_at ? formatDateTime(b.expires_at) : '不过期'}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground tabular-nums">
                      <div className="flex flex-col">
                        <span>{formatDateTime(b.created_at)}</span>
                        <span>{b.created_username || b.created_by}</span>
                      </div>
                    </TableCell>
                    <TableCell>
                      <div className="flex justify-end gap-1">
                        <Button variant="ghost" size="icon" title="查看兑换码" aria-label="查看兑换码" onClick={() => setViewing(b)}>
                          <Ticket />
                        </Button>
                        <Button variant="ghost" size="icon" title="编辑" aria-label="编辑" onClick={() => setEditing({mode: 'edit', batch: b})}>
                          <Pencil />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          title="用相同的设置再生成一批"
                          aria-label="用相同的设置再生成一批"
                          onClick={() => setEditing({mode: 'copy', batch: b})}
                        >
                          <CopyPlus />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          title={b.is_enabled ? '停用' : '启用'}
                          aria-label={b.is_enabled ? '停用' : '启用'}
                          className={b.is_enabled ? undefined : 'text-destructive'}
                          onClick={() => toggleEnabled(b)}
                        >
                          <Power />
                        </Button>
                        <Button variant="ghost" size="icon" title="删除" aria-label="删除" className="hover:text-destructive" onClick={() => setDeleting(b)}>
                          <Trash2 />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                );
              })
            ) : (
              <TableEmpty colSpan={8}>{keyword || status !== ALL ? '没有找到兑换码批次' : '还没有生成过兑换码'}</TableEmpty>
            )}
          </TableBody>
        </Table>
      </div>

      <Pagination current={current} size={PAGE_SIZE} total={data?.total ?? 0} onChange={setCurrent} />

      <BatchDialog
        target={editing}
        options={options}
        onClose={() => setEditing(null)}
        onSaved={(created) => {
          void load();
          // 生成后直接展示兑换码, 方便复制和下载
          if (created) setViewing(created);
        }}
      />
      <CodesDialog batch={viewing} onClose={() => setViewing(null)} />

      <AlertDialog open={!!deleting} onOpenChange={(open) => !open && !pending && setDeleting(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>删除 {deleting?.name}</AlertDialogTitle>
            <AlertDialogDescription>
              删除后这个批次的 {deleting?.code_count} 个兑换码都不能再兑换, 也不能再被新的批次使用; 已兑换的奖励不收回, 兑换记录保留。删除后不能恢复,
              只是暂时不想让玩家兑换的话, 停用即可。
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
