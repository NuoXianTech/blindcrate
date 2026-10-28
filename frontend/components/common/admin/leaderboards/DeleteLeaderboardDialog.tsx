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
import type {AdminLeaderboard} from '@/lib/services/admin';

/**
 * 删除榜单的二次确认, 统计数据不受影响
 */
export function DeleteLeaderboardDialog({board, onClose, onSaved}: {
  board: AdminLeaderboard | null;
  onClose: () => void;
  onSaved: () => void;
}) {
  const {pending, submit} = useSubmit();

  const handleConfirm = submit(async () => {
    if (!board) return;
    await services.admin.deleteLeaderboard(board.id);
    toast.success(`已删除 ${board.name}`);
    onSaved();
    onClose();
  });

  return (
    <AlertDialog open={!!board} onOpenChange={(open) => !open && !pending && onClose()}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>删除 {board?.name}</AlertDialogTitle>
          <AlertDialogDescription>
            删除后无法恢复, 玩家将看不到这个榜单; 只删除榜单配置, 玩家的统计数据不受影响, 重新新建同样的榜单即可恢复排名。
            只是暂时不想展示的话, 停用即可。
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
