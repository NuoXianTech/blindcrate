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
import type {LootPoolListItem} from '@/lib/services/admin';

/**
 * 删除掉落池的二次确认, 条目一同删除
 */
export function DeleteLootPoolDialog({pool, onClose, onSaved}: {
  pool: LootPoolListItem | null;
  onClose: () => void;
  onSaved: () => void;
}) {
  const {pending, submit} = useSubmit();

  const handleConfirm = submit(async () => {
    if (!pool) return;
    const {entry_count: entries} = await services.admin.deleteLootPool(pool.id);
    toast.success(entries > 0 ? `已删除 ${pool.name} 和 ${entries} 个条目` : `已删除 ${pool.name}`);
    onSaved();
    onClose();
  });

  return (
    <AlertDialog open={!!pool} onOpenChange={(open) => !open && !pending && onClose()}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>删除 {pool?.name}</AlertDialogTitle>
          <AlertDialogDescription>
            删除后无法恢复, {pool && pool.entry_count > 0 ? `${pool.entry_count} 个条目会一同删除。` : '当前没有条目。'}
            引用了代码 {pool?.code} 的地图将无法生成仓库, 需要先修改地图的档位。只是暂时不想用的话, 停用即可。
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
