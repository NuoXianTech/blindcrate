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
import type {AdminItem} from '@/lib/services/admin';

/**
 * 删除物品的二次确认, 玩家已有的保留为未知物品
 */
export function DeleteItemDialog({item, onClose, onSaved}: { item: AdminItem | null; onClose: () => void; onSaved: () => void }) {
  const {pending, submit} = useSubmit();

  const handleConfirm = submit(async () => {
    if (!item) return;
    const {holder_count: holders} = await services.admin.deleteItem(item.id);
    toast.success(holders > 0 ? `已删除 ${item.name}, ${holders} 名玩家的背包中保留为未知物品` : `已删除 ${item.name}`);
    onSaved();
    onClose();
  });

  return (
    <AlertDialog open={!!item} onOpenChange={(open) => !open && !pending && onClose()}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>删除 {item?.name}</AlertDialogTitle>
          <AlertDialogDescription>
            删除后无法恢复, 物品 ID 不会被复用。
            {item && item.holder_count > 0 ?
              ` 当前有 ${item.holder_count} 名玩家持有, 他们背包中的物品会保留并显示为「未知物品 #${item.id}」。` :
              ' 当前没有玩家持有。'}
            只是不想再掉落或发放的话, 下架即可。
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
