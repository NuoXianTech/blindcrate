'use client';

import {useState} from 'react';
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
import {Label} from '@/components/ui/label';
import {Textarea} from '@/components/ui/textarea';
import {check, useSubmit} from '@/hooks/use-submit';
import {modeLabel} from '@/lib/auction';
import services from '@/lib/services';
import type {AdminMatch} from '@/lib/services/admin';

/**
 * 取消进行中的对局的二次确认, 需要填写原因
 */
export function CancelMatchDialog({match, onClose, onSaved}: {
  match: AdminMatch | null;
  onClose: () => void;
  onSaved: () => void;
}) {
  return (
    <AlertDialog open={!!match} onOpenChange={(open) => !open && onClose()}>
      <AlertDialogContent>
        {match && <CancelMatchForm key={match.id} match={match} onClose={onClose} onSaved={onSaved} />}
      </AlertDialogContent>
    </AlertDialog>
  );
}

/**
 * 取消对局的表单
 */
function CancelMatchForm({match, onClose, onSaved}: {
  match: AdminMatch;
  onClose: () => void;
  onSaved: () => void;
}) {
  const {pending, submit} = useSubmit();
  const [reason, setReason] = useState('');

  const handleConfirm = submit(async () => {
    check(reason.trim() ? null : '请填写取消原因');
    await services.admin.cancelMatch(match.id, reason.trim());
    toast.success(`已取消对局 #${match.id}`);
    onSaved();
    onClose();
  });

  return (
    <>
      <AlertDialogHeader>
        <AlertDialogTitle>
          取消对局 #{match.id} {modeLabel(match.mode)}
        </AlertDialogTitle>
        <AlertDialogDescription>
          用于对局卡住等异常情况。取消后对局立即结束, 不结算, 玩家的入场代价退还、冻结的资金解冻; 无法撤销。
        </AlertDialogDescription>
      </AlertDialogHeader>
      <div className="flex flex-col gap-2">
        <Label htmlFor="cancel-match-reason">原因</Label>
        <Textarea
          id="cancel-match-reason"
          maxLength={255}
          placeholder="必填, 记录在操作日志中, 最多 255 个字符"
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          disabled={pending}
        />
      </div>
      <AlertDialogFooter>
        <AlertDialogCancel disabled={pending}>返回</AlertDialogCancel>
        <AlertDialogAction variant="destructive" disabled={pending || !reason.trim()} onClick={handleConfirm}>
          {pending && <LoaderCircle className="animate-spin" />}
          取消对局
        </AlertDialogAction>
      </AlertDialogFooter>
    </>
  );
}
