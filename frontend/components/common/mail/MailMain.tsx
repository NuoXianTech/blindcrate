'use client';

import {useCallback, useEffect, useState} from 'react';
import {CheckCheck, Gift, LoaderCircle} from 'lucide-react';
import {toast} from 'sonner';
import {cn} from 'cn';
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
import {useAuth} from '@/components/common/auth';
import {Pagination} from '@/components/common/layout';
import {afterJitter, refreshMailCounts, useMailCounts} from '@/hooks/use-mail-counts';
import {useSubmit} from '@/hooks/use-submit';
import {useWsEvent} from '@/hooks/use-ws';
import services from '@/lib/services';
import {ApiError, toErrorMessage} from '@/lib/services/core';
import {MAILBOX_MESSAGES, type ClaimMailsResult, type ListMailsData, type Mail} from '@/lib/services/mailbox';
import {ClaimResultDialog} from './ClaimResultDialog';
import {MailDetail} from './MailDetail';
import {MailList} from './MailList';
import {MAIL_PAGE_SIZE} from './constants';

/**
 * 信箱: 左边是邮件列表, 右边是详情; 窄屏时选中邮件后只显示详情
 * 打开邮件时标记为已读, 收到新邮件推送时刷新当前页
 */
export function MailMain() {
  const {user} = useAuth();
  const counts = useMailCounts();
  const [current, setCurrent] = useState(1);
  const [data, setData] = useState<ListMailsData | null>(null);
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [deleting, setDeleting] = useState<Mail | null>(null);
  const [result, setResult] = useState<ClaimMailsResult | null>(null);
  const claimSubmit = useSubmit();
  const bulkSubmit = useSubmit();
  const deleteSubmit = useSubmit();

  const load = useCallback(async () => {
    if (!user) return;
    try {
      setData(await services.mailbox.list({current, size: MAIL_PAGE_SIZE}));
    } catch (err) {
      toast.error(toErrorMessage(err));
    }
  }, [current, user]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 翻页时重新拉取
    void load();
  }, [load]);

  useWsEvent(MAILBOX_MESSAGES.new, () => afterJitter(() => void load()));

  const selected = data?.mails.find((m) => m.id === selectedId) ?? null;

  /** 本地更新一封邮件, 不重新拉取 */
  const patch = (id: number, fields: Partial<Mail>) => {
    setData((prev) => prev && {...prev, mails: prev.mails.map((m) => (m.id === id ? {...m, ...fields} : m))});
  };

  const handleSelect = (mail: Mail) => {
    setSelectedId(mail.id);
    if (mail.read_at) return;
    patch(mail.id, {read_at: new Date().toISOString()});
    services.mailbox
        .read(mail.id)
        .then(() => refreshMailCounts())
        .catch(() => {
          // 已读只是标记, 失败时下次打开再标记
        });
  };

  const handleClaim = (mail: Mail) =>
    claimSubmit.submit(async () => {
      try {
        setResult(await services.mailbox.claim(mail.id));
      } catch (err) {
        // 已在别处领取、过期或被删除, 刷新后再提示
        if (err instanceof ApiError && (err.status === 404 || err.status === 409)) void load();
        throw err;
      } finally {
        void refreshMailCounts();
      }
      const now = new Date().toISOString();
      patch(mail.id, {claimed_at: now, read_at: mail.read_at ?? now});
    })();

  const handleClaimAll = bulkSubmit.submit(async () => {
    const res = await services.mailbox.claimAll();
    if (res.claimed > 0) setResult(res);
    else if (res.skipped > 0) toast.error(`${res.skipped} 封邮件的附件暂时无法领取, 请稍后再试`);
    else toast.info('没有可以领取的附件');
    await Promise.all([load(), refreshMailCounts()]);
  });

  const handleReadAll = bulkSubmit.submit(async () => {
    const {read} = await services.mailbox.readAll();
    toast.success(read ? `已将 ${read} 封邮件标为已读` : '没有未读的邮件');
    await Promise.all([load(), refreshMailCounts()]);
  });

  const handleDelete = deleteSubmit.submit(async () => {
    if (!deleting) return;
    await services.mailbox.remove(deleting.id);
    if (deleting.id === selectedId) setSelectedId(null);
    setDeleting(null);
    toast.success('已删除');
    // 删掉了这一页的最后一封时回到上一页, 由翻页触发拉取
    if (data?.mails.length === 1 && current > 1) setCurrent(current - 1);
    else await load();
  });

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-baseline gap-3">
          <h1 className="text-2xl font-bold">邮件</h1>
          {counts && (
            <span className="text-sm text-muted-foreground">
              未读 {counts.unread} · 附件待领取 {counts.claimable}
            </span>
          )}
        </div>
        <div className="flex gap-2">
          <Button variant="outline" disabled={bulkSubmit.pending || !counts?.unread} onClick={handleReadAll}>
            <CheckCheck />
            全部已读
          </Button>
          <Button disabled={bulkSubmit.pending || !counts?.claimable} onClick={handleClaimAll}>
            {bulkSubmit.pending ? <LoaderCircle className="animate-spin" /> : <Gift />}
            一键领取
          </Button>
        </div>
      </div>

      <div className="grid gap-4 lg:grid-cols-[22rem_minmax(0,1fr)]">
        <div className={cn('flex-col gap-3', selected ? 'hidden lg:flex' : 'flex')}>
          <div className="rounded-xl border bg-card">
            <MailList mails={data?.mails ?? null} selectedId={selectedId} onSelect={handleSelect} />
          </div>
          <Pagination current={current} size={MAIL_PAGE_SIZE} total={data?.total ?? 0} onChange={setCurrent} />
        </div>
        <div className={cn('rounded-xl border bg-card lg:self-start', !selected && 'hidden lg:block')}>
          <MailDetail
            mail={selected}
            claiming={claimSubmit.pending}
            onClaim={handleClaim}
            onDelete={setDeleting}
            onBack={() => setSelectedId(null)}
          />
        </div>
      </div>

      <AlertDialog open={!!deleting} onOpenChange={(open) => !open && !deleteSubmit.pending && setDeleting(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>删除邮件</AlertDialogTitle>
            <AlertDialogDescription>删除后不能恢复。确定删除「{deleting?.title}」吗?</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleteSubmit.pending}>取消</AlertDialogCancel>
            <AlertDialogAction variant="destructive" disabled={deleteSubmit.pending} onClick={handleDelete}>
              {deleteSubmit.pending && <LoaderCircle className="animate-spin" />}
              删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <ClaimResultDialog result={result} onClose={() => setResult(null)} />
    </div>
  );
}
