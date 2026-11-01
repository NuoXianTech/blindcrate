'use client';

import {useState} from 'react';
import {toast} from 'sonner';
import {Button} from '@/components/ui/button';
import {Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle} from '@/components/ui/dialog';
import {InputField, SwitchField} from '@/components/common/admin/venues/Fields';
import {SubmitButton} from '@/components/common/form';
import {check, useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';
import type {BotName} from '@/lib/services/bot';
import {MAX_NAME_LENGTH} from './constants';

/**
 * 修改一个昵称
 */
export function NameDialog({name, onClose, onSaved}: {
  /** null 时关闭 */
  name: BotName | null;
  onClose: () => void;
  /** 保存成功后刷新列表 */
  onSaved: () => void;
}) {
  return (
    <Dialog open={!!name} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>编辑 {name?.name}</DialogTitle>
          <DialogDescription>已开局的对局中保存的是昵称本身, 修改不影响进行中和已结束的对局</DialogDescription>
        </DialogHeader>
        {/* key 保证切换昵称时表单重置 */}
        {name && <NameForm key={name.id} name={name} onClose={onClose} onSaved={onSaved} />}
      </DialogContent>
    </Dialog>
  );
}

function NameForm({name, onClose, onSaved}: { name: BotName; onClose: () => void; onSaved: () => void }) {
  const {pending, submit} = useSubmit();
  const [text, setText] = useState(name.name);
  const [isEnabled, setIsEnabled] = useState(name.is_enabled);

  const handleSubmit = submit(async () => {
    const value = text.trim();
    check(value.length === 0 || [...value].length > MAX_NAME_LENGTH ? `昵称需为 1-${MAX_NAME_LENGTH} 个字符` : null);
    await services.botAdmin.updateName(name.id, {name: value, is_enabled: isEnabled});
    toast.success(`已保存 ${value}`);
    onSaved();
    onClose();
  });

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4" noValidate>
      <InputField id="bot-name" label="昵称" maxLength={MAX_NAME_LENGTH} value={text} onChange={setText} disabled={pending} />
      <SwitchField
        id="bot-name-enabled"
        label="启用"
        help="停用的昵称开局时不会选取"
        checked={isEnabled}
        onChange={setIsEnabled}
        disabled={pending}
      />
      <DialogFooter>
        <Button type="button" variant="outline" onClick={onClose} disabled={pending}>
          取消
        </Button>
        <SubmitButton pending={pending}>保存</SubmitButton>
      </DialogFooter>
    </form>
  );
}
