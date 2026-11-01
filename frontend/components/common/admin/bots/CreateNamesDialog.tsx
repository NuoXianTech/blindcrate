'use client';

import {useState} from 'react';
import {toast} from 'sonner';
import {Button} from '@/components/ui/button';
import {Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle} from '@/components/ui/dialog';
import {Label} from '@/components/ui/label';
import {Textarea} from '@/components/ui/textarea';
import {SwitchField} from '@/components/common/admin/venues/Fields';
import {SubmitButton} from '@/components/common/form';
import {check, useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';
import {MAX_BATCH_NAMES, MAX_NAME_LENGTH} from './constants';

/** 输入框里的昵称, 每行一个, 去掉首尾空白和空行 */
function splitNames(text: string): string[] {
  return text.split('\n').map((line) => line.trim()).filter(Boolean);
}

/**
 * 批量添加昵称, 每行一个; 已存在的和重复的跳过
 */
export function CreateNamesDialog({open, onClose, onSaved}: {
  open: boolean;
  onClose: () => void;
  /** 添加成功后刷新列表 */
  onSaved: () => void;
}) {
  return (
    <Dialog open={open} onOpenChange={(next) => !next && onClose()}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>添加昵称</DialogTitle>
          <DialogDescription>每行一个, 最多 {MAX_BATCH_NAMES} 个; 已存在的和重复的会跳过</DialogDescription>
        </DialogHeader>
        {/* 关闭后卸载, 下次打开时表单重置 */}
        {open && <CreateNamesForm onClose={onClose} onSaved={onSaved} />}
      </DialogContent>
    </Dialog>
  );
}

function CreateNamesForm({onClose, onSaved}: { onClose: () => void; onSaved: () => void }) {
  const {pending, submit} = useSubmit();
  const [text, setText] = useState('');
  const [isEnabled, setIsEnabled] = useState(true);
  const names = splitNames(text);

  const handleSubmit = submit(async () => {
    check(names.length === 0 ? '至少填写一个昵称' : null);
    check(names.length > MAX_BATCH_NAMES ? `一次最多添加 ${MAX_BATCH_NAMES} 个昵称` : null);
    const long = names.findIndex((name) => [...name].length > MAX_NAME_LENGTH);
    check(long >= 0 ? `第 ${long + 1} 个昵称超过 ${MAX_NAME_LENGTH} 个字符` : null);
    const {created, skipped} = await services.botAdmin.createNames({names, is_enabled: isEnabled});
    toast.success(skipped.length ? `已添加 ${created.length} 个, 跳过 ${skipped.length} 个已存在或重复的` : `已添加 ${created.length} 个`);
    onSaved();
    onClose();
  });

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4" noValidate>
      <div className="flex flex-col gap-2">
        <Label htmlFor="bot-names">昵称</Label>
        <Textarea
          id="bot-names"
          rows={10}
          placeholder={'橘子汽水\n晚风\n不吃香菜'}
          value={text}
          onChange={(e) => setText(e.target.value)}
          disabled={pending}
        />
        <span className="text-xs text-muted-foreground tabular-nums">
          共 {names.length} 个, 每个 1-{MAX_NAME_LENGTH} 个字符
        </span>
      </div>
      <SwitchField
        id="bot-names-enabled"
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
        <SubmitButton pending={pending}>添加</SubmitButton>
      </DialogFooter>
    </form>
  );
}
