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
import type {AdminVenue} from '@/lib/services/admin';

/**
 * 删除地图的二次确认
 */
export function DeleteVenueDialog({venue, onClose, onSaved}: {
  venue: AdminVenue | null;
  onClose: () => void;
  onSaved: () => void;
}) {
  const {pending, submit} = useSubmit();

  const handleConfirm = submit(async () => {
    if (!venue) return;
    await services.admin.deleteVenue(venue.id);
    toast.success(`已删除 ${venue.name}`);
    onSaved();
    onClose();
  });

  return (
    <AlertDialog open={!!venue} onOpenChange={(open) => !open && !pending && onClose()}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>删除 {venue?.name}</AlertDialogTitle>
          <AlertDialogDescription>
            删除后大厅中不再显示, 进行中的对局照常打完, 历史对局保留。代码 {venue?.code} 之后也不能再用。只是暂时不想开放的话, 停用即可。
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
