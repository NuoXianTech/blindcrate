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
import type {Tier} from '@/lib/services/ranked';

/**
 * 删除段位的二次确认
 */
export function DeleteTierDialog({tier, onClose, onSaved}: {
  tier: Tier | null;
  onClose: () => void;
  onSaved: () => void;
}) {
  const {pending, submit} = useSubmit();

  const handleConfirm = submit(async () => {
    if (!tier) return;
    await services.admin.deleteTier(tier.id);
    toast.success(`已删除 ${tier.name}`);
    onSaved();
    onClose();
  });

  return (
    <AlertDialog open={!!tier} onOpenChange={(open) => !open && !pending && onClose()}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>删除 {tier?.name}</AlertDialogTitle>
          <AlertDialogDescription>
            删除后无法恢复; 段位分不受影响, 原来在这个段位的人按剩下的分数线重新计算段位。
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
