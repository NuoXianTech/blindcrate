'use client';

import {useState} from 'react';
import {nanoid} from 'nanoid';
import {toast} from 'sonner';
import {Button} from '@/components/ui/button';
import {Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle} from '@/components/ui/dialog';
import {Skeleton} from '@/components/ui/skeleton';
import {ConditionEditor, fromConditionDrafts, type ConditionDraft, type ScopeOption} from '@/components/common/admin/editors';
import {FieldGroup, SwitchField} from '@/components/common/admin/venues/Fields';
import {SubmitButton} from '@/components/common/form';
import {useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';
import type {AdminItem, ConditionType, Currency} from '@/lib/services/admin';
import {EMPTY_LETTER, LetterFields, fromLetterDraft, type LetterDraft} from './LetterFields';
import {MAX_REQUIREMENTS} from './constants';

/** 编辑全服邮件用到的物品、货币、条件类型和统计范围 */
export interface BroadcastOptions {
  items: AdminItem[];
  currencies: Currency[];
  types: ConditionType[];
  scopes: ScopeOption[];
  loaded: boolean;
}

/**
 * 发送全服邮件; 发送后不能修改, 只能撤回
 */
export function BroadcastDialog({open, options, onClose, onSaved}: {
  open: boolean;
  options: BroadcastOptions;
  onClose: () => void;
  onSaved: () => void;
}) {
  return (
    <Dialog open={open} onOpenChange={(next) => !next && onClose()}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>发送全服邮件</DialogTitle>
          <DialogDescription>玩家打开信箱时放进信箱, 满足收件条件才能收到; 发送后不能修改, 只能撤回</DialogDescription>
        </DialogHeader>
        {open && !options.loaded && (
          <div className="flex flex-col gap-2">
            <Skeleton className="h-9" />
            <Skeleton className="h-9" />
            <Skeleton className="h-9" />
          </div>
        )}
        {open && options.loaded && <BroadcastForm options={options} onClose={onClose} onSaved={onSaved} />}
      </DialogContent>
    </Dialog>
  );
}

/**
 * 全服邮件表单, 每次打开都是新的
 * 幂等键在第一次提交时生成, 修改内容后作废; 超时后重试同一份内容只发送一次
 */
function BroadcastForm({options, onClose, onSaved}: { options: BroadcastOptions; onClose: () => void; onSaved: () => void }) {
  const {pending, submit} = useSubmit();
  const {items, currencies, types, scopes} = options;
  const [letter, setLetter] = useState<LetterDraft>(EMPTY_LETTER);
  const [requirements, setRequirements] = useState<ConditionDraft[]>([]);
  const [includeNewUsers, setIncludeNewUsers] = useState(false);
  const [idempotencyKey, setIdempotencyKey] = useState<string | null>(null);

  // 修改内容后视为新的一次发送
  const change = <T, >(setter: (value: T) => void) => (value: T) => {
    setter(value);
    setIdempotencyKey(null);
  };

  const handleSubmit = submit(async () => {
    const key = idempotencyKey ?? nanoid();
    setIdempotencyKey(key);
    const created = await services.mailboxAdmin.createBroadcast({
      ...fromLetterDraft(letter, currencies),
      requirements: fromConditionDrafts(requirements, types, '收件条件'),
      include_new_users: includeNewUsers,
      idempotency_key: key,
    });
    toast.success(`已发送全服邮件 ${created.title}`);
    onSaved();
    onClose();
  });

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-6" noValidate>
      <LetterFields idPrefix="mail-broadcast" draft={letter} onChange={change(setLetter)} items={items} currencies={currencies} disabled={pending} />

      <FieldGroup title="收件范围" help={`收件条件需要全部满足, 玩家打开信箱时检查, 之后满足的也能收到; 最多 ${MAX_REQUIREMENTS} 个`}>
        <SwitchField
          id="mail-broadcast-new-users"
          label="之后注册的用户也能收到"
          help="关闭时只发给现在已注册的用户, 如维护补偿; 开启时适合新手福利等长期邮件"
          checked={includeNewUsers}
          onChange={change(setIncludeNewUsers)}
          disabled={pending}
        />
        <ConditionEditor
          drafts={requirements}
          onChange={change(setRequirements)}
          types={types}
          items={items}
          currencies={currencies}
          scopes={scopes}
          max={MAX_REQUIREMENTS}
          empty="没有收件条件, 所有玩家都能收到"
          disabled={pending}
        />
      </FieldGroup>

      <DialogFooter>
        <Button type="button" variant="outline" onClick={onClose} disabled={pending}>
          取消
        </Button>
        <SubmitButton pending={pending}>发送</SubmitButton>
      </DialogFooter>
    </form>
  );
}
