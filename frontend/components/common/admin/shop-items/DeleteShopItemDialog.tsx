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
import type {AdminShopItem} from '@/lib/services/admin';

/**
 * 删除商品的二次确认
 */
export function DeleteShopItemDialog({item, onClose, onSaved}: {
  item: AdminShopItem | null;
  onClose: () => void;
  onSaved: () => void;
}) {
  const {pending, submit} = useSubmit();

  const handleConfirm = submit(async () => {
    if (!item) return;
    await services.admin.deleteShopItem(item.id);
    toast.success(`已删除 ${item.name}`);
    onSaved();
    onClose();
  });

  return (
    <AlertDialog open={!!item} onOpenChange={(open) => !open && !pending && onClose()}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>删除 {item?.name}</AlertDialogTitle>
          <AlertDialogDescription>
            删除后商城中不再显示, 已有订单保留, 仍可退款。代码 {item?.code} 之后也不能再用。只是暂时不想卖的话, 下架即可。
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
