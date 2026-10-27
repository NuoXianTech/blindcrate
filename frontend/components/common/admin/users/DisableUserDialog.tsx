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
import type {UserDialogProps} from './EditUserDialog';

/**
 * 停用用户的二次确认, 停用后无法登录直到重新启用
 */
export function DisableUserDialog({user, onClose, onSaved}: UserDialogProps) {
  const {pending, submit} = useSubmit();

  const handleConfirm = submit(async () => {
    if (!user) return;
    await services.admin.setUserDisabled(user.id, true);
    toast.success(`已停用 ${user.username}`);
    onSaved();
    onClose();
  });

  return (
    <AlertDialog open={!!user} onOpenChange={(open) => !open && !pending && onClose()}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>停用 {user?.username}</AlertDialogTitle>
          <AlertDialogDescription>停用后该账号无法登录, 所有设备会立即下线, 直到重新启用。</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={pending}>取消</AlertDialogCancel>
          <AlertDialogAction variant="destructive" disabled={pending} onClick={handleConfirm}>
            {pending && <LoaderCircle className="animate-spin" />}
            停用
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
