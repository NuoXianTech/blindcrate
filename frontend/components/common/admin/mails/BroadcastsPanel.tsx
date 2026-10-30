'use client';

import {useCallback, useEffect, useState} from 'react';
import {LoaderCircle, Plus, Search, Undo2} from 'lucide-react';
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
import {Table, TableBody, TableCell, TableHead, TableHeader, TableRow} from '@/components/ui/table';
import {bundleText} from '@/components/common/admin/editors';
import {Pagination, TableEmpty, TableSkeleton} from '@/components/common/layout';
import {useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';
import {toErrorMessage} from '@/lib/services/core';
import type {ListMailBroadcastsData, MailBroadcast} from '@/lib/services/mailbox';
import {formatDateTime} from '@/lib/utils';
import {BroadcastDialog, type BroadcastOptions} from './BroadcastDialog';
import {PAGE_SIZE} from './constants';

/**
 * 全服邮件: 列表带放进信箱和已领取的人数, 发送新的和撤回
 */
export function BroadcastsPanel({options}: { options: BroadcastOptions }) {
  const [keywordInput, setKeywordInput] = useState('');
  const [keyword, setKeyword] = useState('');
  const [current, setCurrent] = useState(1);
  const [data, setData] = useState<ListMailBroadcastsData | null>(null);
  // 列表加载的时间, 用于判断是否已过期
  const [loadedAt, setLoadedAt] = useState(0);
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);
  const [revoking, setRevoking] = useState<MailBroadcast | null>(null);
  const {pending, submit} = useSubmit();

  const load = useCallback(async () => {
    setLoading(true);
    try {
      setData(await services.mailboxAdmin.listBroadcasts({current, size: PAGE_SIZE, keyword: keyword || undefined}));
      setLoadedAt(Date.now());
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

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    setCurrent(1);
    setKeyword(keywordInput.trim());
  };

  const handleRevoke = submit(async () => {
    if (!revoking) return;
    const {removed} = await services.mailboxAdmin.revokeBroadcast(revoking.id);
    toast.success(removed ? `已撤回, ${removed} 封未领取的邮件移出了信箱` : '已撤回');
    setRevoking(null);
    await load();
  });

  const {items, currencies} = options;

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <form onSubmit={handleSearch} className="flex gap-2">
          <Input className="w-56" placeholder="按标题搜索" value={keywordInput} onChange={(e) => setKeywordInput(e.target.value)} />
          <Button type="submit" variant="outline" size="icon" aria-label="搜索">
            <Search />
          </Button>
        </form>
        <Button onClick={() => setCreating(true)}>
          <Plus />
          发送全服邮件
        </Button>
      </div>

      <div className="rounded-xl border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>标题</TableHead>
              <TableHead>附件</TableHead>
              <TableHead>收件范围</TableHead>
              <TableHead className="text-right">放进信箱 / 已领取</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>发送</TableHead>
              <TableHead className="w-12" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading && !data ? (
              <TableSkeleton cols={7} />
            ) : data?.broadcasts.length ? (
              data.broadcasts.map((b) => {
                const attachments = bundleText(b.attachments, items, currencies);
                const expired = new Date(b.expires_at).getTime() <= loadedAt;
                return (
                  <TableRow key={b.id}>
                    <TableCell className="max-w-56">
                      <div className="flex min-w-0 flex-col">
                        <span className="truncate font-medium" title={b.title}>
                          {b.title}
                        </span>
                        <span className="truncate text-xs text-muted-foreground">{b.sender}</span>
                      </div>
                    </TableCell>
                    <TableCell className="max-w-48 truncate" title={attachments.join(', ') || undefined}>
                      {attachments.length ? attachments.join(', ') : <span className="text-muted-foreground">无</span>}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      <div className="flex flex-col">
                        <span>{b.registered_before ? '发送时已注册的用户' : '包括之后注册的用户'}</span>
                        {b.requirements.length > 0 && <span>{b.requirements.length} 个收件条件</span>}
                      </div>
                    </TableCell>
                    <TableCell className="text-right font-mono tabular-nums">
                      {b.expanded} / {b.claimed}
                    </TableCell>
                    <TableCell>
                      {b.revoked_at ? (
                        <Badge variant="outline" title={`${b.revoked_username || b.revoked_by} 于 ${formatDateTime(b.revoked_at)} 撤回`}>
                          已撤回
                        </Badge>
                      ) : expired ? (
                        <Badge variant="outline">已过期</Badge>
                      ) : (
                        <Badge variant="secondary" title={`${formatDateTime(b.expires_at)} 过期`}>
                          生效中
                        </Badge>
                      )}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground tabular-nums">
                      <div className="flex flex-col">
                        <span>{formatDateTime(b.created_at)}</span>
                        <span>{b.created_username || b.created_by}</span>
                      </div>
                    </TableCell>
                    <TableCell>
                      {!b.revoked_at && !expired && (
                        <Button variant="ghost" size="icon" title="撤回" aria-label="撤回" onClick={() => setRevoking(b)}>
                          <Undo2 />
                        </Button>
                      )}
                    </TableCell>
                  </TableRow>
                );
              })
            ) : (
              <TableEmpty colSpan={7}>还没有全服邮件</TableEmpty>
            )}
          </TableBody>
        </Table>
      </div>

      <Pagination current={current} size={PAGE_SIZE} total={data?.total ?? 0} onChange={setCurrent} />

      <BroadcastDialog open={creating} options={options} onClose={() => setCreating(false)} onSaved={() => void load()} />

      <AlertDialog open={!!revoking} onOpenChange={(open) => !open && !pending && setRevoking(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>撤回 {revoking?.title}</AlertDialogTitle>
            <AlertDialogDescription>
              撤回后不再放进信箱; 已放进信箱但附件未领取的邮件移出信箱, 已领取的附件不收回。撤回后不能恢复。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={pending}>取消</AlertDialogCancel>
            <AlertDialogAction variant="destructive" disabled={pending} onClick={handleRevoke}>
              {pending && <LoaderCircle className="animate-spin" />}
              撤回
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
