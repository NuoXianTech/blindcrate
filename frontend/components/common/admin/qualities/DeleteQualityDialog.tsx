'use client';

import {useRouter} from 'next/navigation';
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
import type {AdminQuality} from '@/lib/services/admin';
import {refreshQualities} from '@/lib/site-actions';

/**
 * 删除品质的二次确认, 还有物品使用该品质时不能删除
 */
export function DeleteQualityDialog({quality, onClose, onSaved}: {
  quality: AdminQuality | null;
  onClose: () => void;
  onSaved: () => void;
}) {
  const router = useRouter();
  const {pending, submit} = useSubmit();
  const inUse = !!quality && quality.item_count > 0;

  const handleConfirm = submit(async () => {
    if (!quality) return;
    await services.admin.deleteQuality(quality.level);
    toast.success(`已删除 ${quality.name}`);
    await refreshQualities();
    router.refresh();
    onSaved();
    onClose();
  });

  return (
    <AlertDialog open={!!quality} onOpenChange={(open) => !open && !pending && onClose()}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>删除 {quality?.name}</AlertDialogTitle>
          <AlertDialogDescription>
            {inUse ?
              `还有 ${quality.item_count} 个物品使用这个品质, 请先在物品管理中把它们改为其他品质。` :
              '删除后无法恢复; 已开局对局中这个品质的藏品保留原等级, 按未知品质展示。'}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={pending}>{inUse ? '知道了' : '取消'}</AlertDialogCancel>
          {!inUse && (
            <AlertDialogAction variant="destructive" disabled={pending} onClick={handleConfirm}>
              {pending && <LoaderCircle className="animate-spin" />}
              删除
            </AlertDialogAction>
          )}
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
