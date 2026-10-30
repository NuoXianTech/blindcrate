'use client';

import {Gift, Inbox, Paperclip} from 'lucide-react';
import {cn} from 'cn';
import {Skeleton} from '@/components/ui/skeleton';
import type {Mail} from '@/lib/services/mailbox';
import {formatDateTime} from '@/lib/utils';
import {claimable} from './constants';

/**
 * 邮件列表: 未读的标题加粗并带圆点, 附件未领取的显示礼物图标
 */
export function MailList({mails, selectedId, onSelect}: {
  /** null 时显示加载占位 */
  mails: Mail[] | null;
  selectedId: number | null;
  onSelect: (mail: Mail) => void;
}) {
  if (mails === null) {
    return (
      <div className="flex flex-col gap-2 p-2">
        {Array.from({length: 5}, (_, i) => (
          <Skeleton key={i} className="h-14 rounded-lg" />
        ))}
      </div>
    );
  }
  if (!mails.length) {
    return (
      <div className="flex h-48 flex-col items-center justify-center gap-2 text-muted-foreground">
        <Inbox className="size-8" />
        <span className="text-sm">信箱是空的</span>
      </div>
    );
  }
  return (
    <ul className="flex flex-col gap-1 p-2">
      {mails.map((mail) => (
        <li key={mail.id}>
          <button
            type="button"
            onClick={() => onSelect(mail)}
            className={cn(
                'flex w-full items-center gap-3 rounded-lg px-3 py-2 text-left transition-colors hover:bg-accent',
                mail.id === selectedId && 'bg-accent',
            )}
          >
            <span className={cn('size-2 shrink-0 rounded-full', !mail.read_at && 'bg-primary')} />
            <span className="flex min-w-0 flex-1 flex-col gap-0.5">
              <span className={cn('truncate text-sm', !mail.read_at && 'font-semibold')} title={mail.title}>
                {mail.title}
              </span>
              <span className="truncate text-xs text-muted-foreground">
                {mail.sender} · {formatDateTime(mail.created_at)}
              </span>
            </span>
            {claimable(mail) ? (
              <Gift className="size-4 shrink-0 text-primary" aria-label="附件未领取" />
            ) : mail.attachments.length > 0 ? (
              <Paperclip className="size-4 shrink-0 text-muted-foreground" aria-label="附件已领取" />
            ) : null}
          </button>
        </li>
      ))}
    </ul>
  );
}
