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
import type {SettingGroup} from '@/lib/services/settings';

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
}

/**
 * 恢复默认值的二次确认
 */
export function ResetSettingDialog({group, open, dirty, onClose, onReset}: ResetSettingDialogProps) {
  const {pending, submit} = useSubmit();
  const hasSecret = group.fields.some((f) => f.secret);

  const handleConfirm = submit(async () => {
    const updated = await services.settingsAdmin.resetSetting(group.key);
    // 没有任何字段变化时后端不保存, 保存时间不变
    toast.success(updated.updated_at === group.updated_at ? '已经是默认值, 没有变化' : `已恢复${group.label}的默认值`);
    onReset(updated);
    onClose();
  });

  return (
    <AlertDialog open={open} onOpenChange={(o) => !o && !pending && onClose()}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>恢复{group.label}的默认值</AlertDialogTitle>
          <AlertDialogDescription>
            全部字段恢复为默认值并立即生效{hasSecret && ', 密钥字段会被清空'}{dirty && ', 未保存的修改也会丢失'}。恢复后不能撤销, 原来的值可以在操作日志中查到。
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
