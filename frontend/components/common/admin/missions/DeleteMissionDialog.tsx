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
import type {AdminMission} from '@/lib/services/mission';

/**
 * 删除任务的二次确认
 */
export function DeleteMissionDialog({mission, onClose, onSaved}: {
  mission: AdminMission | null;
  onClose: () => void;
  onSaved: () => void;
}) {
  const {pending, submit} = useSubmit();

  const handleConfirm = submit(async () => {
    if (!mission) return;
    await services.missionAdmin.deleteMission(mission.id);
    toast.success(`已删除 ${mission.name}`);
    onSaved();
    onClose();
  });

  return (
    <AlertDialog open={!!mission} onOpenChange={(open) => !open && !pending && onClose()}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>删除 {mission?.name}</AlertDialogTitle>
          <AlertDialogDescription>
            删除后玩家看不到这个任务, 已领取的奖励不收回。代码 {mission?.code} 之后也不能再用。只是暂时不想开放的话, 停用即可。
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
