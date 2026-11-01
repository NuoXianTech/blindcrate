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
import type {AdminCodexSet} from '@/lib/services/codex';

/**
 * 删除收集组的二次确认
 */
export function DeleteCodexSetDialog({set, onClose, onSaved}: {
  set: AdminCodexSet | null;
  onClose: () => void;
  onSaved: () => void;
}) {
  const {pending, submit} = useSubmit();

  const handleConfirm = submit(async () => {
    if (!set) return;
    await services.codexAdmin.deleteSet(set.id);
    toast.success(`已删除 ${set.name}`);
    onSaved();
    onClose();
  });

  return (
    <AlertDialog open={!!set} onOpenChange={(open) => !open && !pending && onClose()}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>删除 {set?.name}</AlertDialogTitle>
          <AlertDialogDescription>
            删除后玩家看不到这个收集组, 已领取的奖励不收回。代码 {set?.code} 之后也不能再用。只是暂时不想开放的话, 停用即可。
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={pending}>取消</AlertDialogCancel>
          <AlertDialogAction variant="destructive" disabled={pending} onClick={handleConfirm}>
            {pending && <LoaderCircle className="animate-spin" />}
            删除
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
