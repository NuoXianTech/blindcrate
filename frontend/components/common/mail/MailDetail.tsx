'use client';

import {ArrowLeft, Gift, LoaderCircle, MailOpen, Trash2} from 'lucide-react';
import {Badge} from '@/components/ui/badge';
import {Button} from '@/components/ui/button';
import {RewardList} from '@/components/common/reward';
import type {Mail} from '@/lib/services/mailbox';
import {formatDateTime} from '@/lib/utils';
import {claimable, expiryText} from './constants';

/**
 * 邮件详情: 正文、附件和领取按钮; 附件未领取时不能删除
 */
export function MailDetail({mail, claiming, onClaim, onDelete, onBack}: {
  mail: Mail | null;
  claiming: boolean;
  onClaim: (mail: Mail) => void;
  onDelete: (mail: Mail) => void;
  /** 窄屏时回到列表 */
  onBack: () => void;
}) {
  if (!mail) {
    return (
      <div className="flex h-full min-h-64 flex-col items-center justify-center gap-2 text-muted-foreground">
        <MailOpen className="size-8" />
        <span className="text-sm">选择一封邮件查看</span>
      </div>
    );
  }
  const pending = claimable(mail);
  return (
    <article className="flex flex-col gap-4 p-5">
      <div className="flex items-start gap-2">
        <Button variant="ghost" size="icon" className="-ml-2 lg:hidden" aria-label="返回列表" onClick={onBack}>
          <ArrowLeft />
        </Button>
        <div className="flex min-w-0 flex-1 flex-col gap-1">
          <h2 className="text-lg font-semibold break-words">{mail.title}</h2>
          <p className="text-xs text-muted-foreground">
            {mail.sender} · {formatDateTime(mail.created_at)} · {expiryText(mail.expires_at)}
          </p>
        </div>
        <Button
          variant="ghost"
          size="icon"
          aria-label="删除"
          title={pending ? '附件还没有领取, 不能删除' : '删除'}
          disabled={pending}
          onClick={() => onDelete(mail)}
        >
          <Trash2 />
        </Button>
      </div>

      {mail.content && <p className="text-sm leading-relaxed break-words whitespace-pre-wrap">{mail.content}</p>}

      {mail.attachments.length > 0 && (
        <section className="flex flex-col gap-3 rounded-xl border bg-background/50 p-4">
          <div className="flex items-center justify-between gap-2">
            <span className="text-sm font-medium">附件</span>
            {mail.claimed_at && (
              <Badge variant="secondary" title={`${formatDateTime(mail.claimed_at)} 领取`}>
                已领取
              </Badge>
            )}
          </div>
          <RewardList lines={mail.attachments} />
          {pending && (
            <Button className="self-start" disabled={claiming} onClick={() => onClaim(mail)}>
              {claiming ? <LoaderCircle className="animate-spin" /> : <Gift />}
              领取
            </Button>
          )}
        </section>
      )}
    </article>
  );
}
