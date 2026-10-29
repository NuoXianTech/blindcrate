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
import type {Season} from '@/lib/services/season';

/** 需要二次确认的操作: 删除未开始的赛季或提前结束进行中的赛季 */
export type SeasonAction = 'delete' | 'end';

const ACTIONS: Record<SeasonAction, { label: string; description: string }> = {
  delete: {
    label: '删除',
    description: '删除后无法恢复; 赛季还没有开始, 不会影响任何数据。',
  },
  end: {
    label: '提前结束',
    description: '结束时间改为现在, 之后的对局不再计入这个赛季, 赛季排行榜的名次随即固定; 结束后不能再恢复, 也不能修改时间。',
  },
};

/**
 * 删除或提前结束赛季的二次确认
 */
export function SeasonActionDialog({target, onClose, onSaved}: {
  target: { season: Season; action: SeasonAction } | null;
  onClose: () => void;
  onSaved: () => void;
}) {
  const {pending, submit} = useSubmit();
  const action = target ? ACTIONS[target.action] : null;

  const handleConfirm = submit(async () => {
    if (!target) return;
    if (target.action === 'delete') {
      await services.admin.deleteSeason(target.season.id);
      toast.success(`已删除 ${target.season.name}`);
    } else {
      await services.admin.endSeason(target.season.id);
      toast.success(`已结束 ${target.season.name}`);
    }
    onSaved();
    onClose();
  });

  return (
    <AlertDialog open={!!target} onOpenChange={(open) => !open && !pending && onClose()}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>
            {action?.label} {target?.season.name}
          </AlertDialogTitle>
          <AlertDialogDescription>{action?.description}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={pending}>取消</AlertDialogCancel>
          <AlertDialogAction variant="destructive" disabled={pending} onClick={handleConfirm}>
            {pending && <LoaderCircle className="animate-spin" />}
            {action?.label}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
