'use client';

import {useState} from 'react';
import {nanoid} from 'nanoid';
import {toast} from 'sonner';
import {Label} from '@/components/ui/label';
import {Textarea} from '@/components/ui/textarea';
import {FieldGroup} from '@/components/common/admin/venues/Fields';
import {SubmitButton} from '@/components/common/form';
import {check, useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';
import type {AdminItem, Currency} from '@/lib/services/admin';
import {EMPTY_LETTER, LetterFields, fromLetterDraft, type LetterDraft} from './LetterFields';
import {MAX_RECIPIENTS, parseUserIDs} from './constants';

/**
 * 发给指定用户: 填写用户 ID 和邮件内容; 发送成功后清空表单
 * 幂等键在第一次提交时生成, 修改内容或发送成功后作废; 超时后重试同一份内容只发送一次
 */
export function SendPanel({items, currencies, loaded}: { items: AdminItem[]; currencies: Currency[]; loaded: boolean }) {
  const {pending, submit} = useSubmit();
  const [userIDs, setUserIDs] = useState('');
  const [letter, setLetter] = useState<LetterDraft>(EMPTY_LETTER);
  const [idempotencyKey, setIdempotencyKey] = useState<string | null>(null);

  // 修改内容后视为新的一次发送
  const change = <T, >(setter: (value: T) => void) => (value: T) => {
    setter(value);
    setIdempotencyKey(null);
  };

  const handleSubmit = submit(async () => {
    const ids = parseUserIDs(userIDs);
    check(ids === null ? '用户 ID 需为正整数, 用逗号、空格或换行分隔' : null);
    check(!ids!.length || ids!.length > MAX_RECIPIENTS ? `收件人需为 1-${MAX_RECIPIENTS} 人` : null);
    const key = idempotencyKey ?? nanoid();
    setIdempotencyKey(key);
    const {sent} = await services.mailboxAdmin.sendMails({
      ...fromLetterDraft(letter, currencies),
      user_ids: ids!,
      idempotency_key: key,
    });
    toast.success(sent ? `已发送 ${sent} 封邮件` : '这封邮件已经发送过了');
    setUserIDs('');
    setLetter(EMPTY_LETTER);
    setIdempotencyKey(null);
  });

  return (
    <form onSubmit={handleSubmit} className="flex max-w-2xl flex-col gap-6 rounded-xl border bg-card p-5" noValidate>
      <FieldGroup title="收件人" help={`用户 ID, 用逗号、空格或换行分隔, 最多 ${MAX_RECIPIENTS} 人; 有用户不存在时整体不发送`}>
        <div className="flex flex-col gap-2">
          <Label htmlFor="mail-send-users" className="sr-only">
            用户 ID
          </Label>
          <Textarea
            id="mail-send-users"
            className="font-mono"
            placeholder="如 1, 2, 3"
            value={userIDs}
            onChange={(e) => change(setUserIDs)(e.target.value)}
            disabled={pending}
          />
        </div>
      </FieldGroup>
      {/* 物品和货币加载前不能编辑附件 */}
      <LetterFields idPrefix="mail-send" draft={letter} onChange={change(setLetter)} items={items} currencies={currencies} disabled={pending || !loaded} />
      <SubmitButton pending={pending} disabled={!loaded} className="self-start">
        发送
      </SubmitButton>
    </form>
  );
}
