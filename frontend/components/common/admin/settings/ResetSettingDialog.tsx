'use client';

import {LoaderCircle} from 'lucide-react';
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
import {useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';
import type {SettingGroup} from '@/lib/services/admin';

/**
 * 恢复默认值的Props接口
 */
interface ResetSettingDialogProps {
  group: SettingGroup;
  open: boolean;
  /** 是否有未保存的修改 */
  dirty: boolean;
  onClose: () => void;
  /** 恢复成功后返回保存后的设置组 */
  onReset: (group: SettingGroup) => void;
  /** 已被别人修改时重新加载设置组 */
  onConflict: (err: unknown) => Promise<boolean>;
}

/**
 * 恢复默认值的二次确认
 */
export function ResetSettingDialog({group, open, dirty, onClose, onReset, onConflict}: ResetSettingDialogProps) {
  const {pending, submit} = useSubmit();
  const hasSecret = group.fields.some((f) => f.secret);

  const handleConfirm = submit(async () => {
    try {
      const updated = await services.admin.resetSetting(group.key, {version: group.version});
      toast.success(updated.version === group.version ? '已经是默认值, 没有变化' : `已恢复${group.label}的默认值`);
      onReset(updated);
      onClose();
    } catch (err) {
      if (await onConflict(err)) {
        onClose();
        return;
      }
      throw err;
    }
  });

  return (
    <AlertDialog open={open} onOpenChange={(o) => !o && !pending && onClose()}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>恢复{group.label}的默认值</AlertDialogTitle>
          <AlertDialogDescription>
            全部字段恢复为默认值并立即生效{hasSecret && ', 密钥字段会被清空'}{dirty && ', 未保存的修改也会丢失'}。恢复会生成一个新版本, 之后可以在历史版本中回滚。
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={pending}>取消</AlertDialogCancel>
          <AlertDialogAction variant="destructive" disabled={pending} onClick={handleConfirm}>
            {pending && <LoaderCircle className="animate-spin" />}
            恢复默认值
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
