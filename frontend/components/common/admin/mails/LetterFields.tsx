'use client';

import {Label} from '@/components/ui/label';
import {Textarea} from '@/components/ui/textarea';
import {BundleEditor, fromBundleDrafts, type BundleDraft} from '@/components/common/admin/editors';
import {FieldGroup, InputField} from '@/components/common/admin/venues/Fields';
import {check} from '@/hooks/use-submit';
import type {AdminItem, Currency} from '@/lib/services/admin';
import type {Bundle} from '@/lib/services/auction';
import {parseAmount} from '@/lib/utils';
import {MAX_ATTACHMENT_LINES, MAX_CONTENT, MAX_EXPIRE_DAYS, MAX_SENDER, MAX_TITLE} from './constants';

/** 邮件内容的表单状态 */
export interface LetterDraft {
  sender: string;
  title: string;
  content: string;
  attachments: BundleDraft[];
  /** 有效期天数的输入框原文, 留空为默认有效期 */
  expireDays: string;
}

export const EMPTY_LETTER: LetterDraft = {sender: '', title: '', content: '', attachments: [], expireDays: ''};

/** 校验后转为请求中的字段, 不合法时中断提交 */
export function fromLetterDraft(draft: LetterDraft, currencies: Currency[]): {
  sender: string;
  title: string;
  content: string;
  attachments: Bundle;
  expire_days: number;
} {
  const title = draft.title.trim();
  check(title.length === 0 || [...title].length > MAX_TITLE ? `标题需为 1-${MAX_TITLE} 个字符` : null);
  const expireDays = draft.expireDays.trim() ? parseAmount(draft.expireDays.trim()) : 0;
  check(expireDays === null || expireDays < 0 || expireDays > MAX_EXPIRE_DAYS ? `有效期需为 1-${MAX_EXPIRE_DAYS} 天, 留空为默认` : null);
  return {
    sender: draft.sender.trim(),
    title,
    content: draft.content.trim(),
    attachments: fromBundleDrafts(draft.attachments, currencies, '附件'),
    expire_days: expireDays!,
  };
}

/**
 * 邮件内容: 发件人、标题、正文、有效期和附件
 */
export function LetterFields({idPrefix, draft, onChange, items, currencies, disabled}: {
  idPrefix: string;
  draft: LetterDraft;
  onChange: (draft: LetterDraft) => void;
  items: AdminItem[];
  currencies: Currency[];
  disabled?: boolean;
}) {
  const set = <K extends keyof LetterDraft>(key: K) => (value: LetterDraft[K]) => onChange({...draft, [key]: value});
  return (
    <>
      <FieldGroup title="邮件内容">
        <div className="grid grid-cols-2 gap-3">
          <InputField
            id={`${idPrefix}-title`}
            label="标题"
            maxLength={MAX_TITLE}
            placeholder="如 维护补偿"
            value={draft.title}
            onChange={set('title')}
            disabled={disabled}
          />
          <InputField
            id={`${idPrefix}-sender`}
            label="发件人"
            maxLength={MAX_SENDER}
            placeholder="留空为 系统"
            value={draft.sender}
            onChange={set('sender')}
            disabled={disabled}
          />
        </div>
        <div className="flex flex-col gap-2">
          <Label htmlFor={`${idPrefix}-content`}>正文</Label>
          <Textarea
            id={`${idPrefix}-content`}
            className="min-h-28"
            maxLength={MAX_CONTENT}
            placeholder={`选填, 最多 ${MAX_CONTENT} 个字符`}
            value={draft.content}
            onChange={(e) => set('content')(e.target.value)}
            disabled={disabled}
          />
        </div>
        <InputField
          id={`${idPrefix}-expire`}
          label="有效期"
          inputMode="numeric"
          suffix="天"
          placeholder="留空为默认有效期"
          value={draft.expireDays}
          onChange={set('expireDays')}
          help="过期后从信箱中消失, 附件不能再领取; 默认有效期见系统设置中的游戏邮件"
          disabled={disabled}
        />
      </FieldGroup>

      <FieldGroup title="附件" help={`领取时发放; 已下架的物品按回收价折成货币; 最多 ${MAX_ATTACHMENT_LINES} 项`}>
        <BundleEditor
          drafts={draft.attachments}
          onChange={set('attachments')}
          items={items}
          currencies={currencies}
          max={MAX_ATTACHMENT_LINES}
          empty="没有附件"
          disabled={disabled}
        />
      </FieldGroup>
    </>
  );
}
