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
import {useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';
import type {Suspect} from '@/lib/services/admin';

/**
 * 把一对可疑账号标记为已处理, 可以填写处理结果
 */
export function ReviewSuspectDialog({suspect, onClose, onSaved}: {
  suspect: Suspect | null;
  onClose: () => void;
  onSaved: () => void;
}) {
  return (
    <AlertDialog open={!!suspect} onOpenChange={(open) => !open && onClose()}>
      <AlertDialogContent>
        {suspect && (
          <ReviewSuspectForm
            key={`${suspect.user_a.id}:${suspect.user_b.id}`}
            suspect={suspect}
            onClose={onClose}
            onSaved={onSaved}
          />
        )}
      </AlertDialogContent>
    </AlertDialog>
  );
}

/**
 * 处理可疑账号的表单, 默认带上一次的处理结果
 */
function ReviewSuspectForm({suspect: s, onClose, onSaved}: {
  suspect: Suspect;
  onClose: () => void;
  onSaved: () => void;
}) {
  const {pending, submit} = useSubmit();
  const [note, setNote] = useState(s.review?.note ?? '');

  const handleConfirm = submit(async () => {
    await services.admin.reviewSuspect({
      user_a: s.user_a.id,
      user_b: s.user_b.id,
      last_match_id: s.last_match_id,
      note: note.trim(),
    });
    toast.success(`已处理 ${s.user_a.username} 和 ${s.user_b.username}`);
    onSaved();
    onClose();
  });

  return (
    <>
      <AlertDialogHeader>
        <AlertDialogTitle>
          处理 {s.user_a.username} 和 {s.user_b.username}
        </AlertDialogTitle>
        <AlertDialogDescription>
          标记后移出待处理, 之后两人又一起打了排位时重新进入待处理。封禁、停用请在用户管理中操作。
        </AlertDialogDescription>
      </AlertDialogHeader>
      <div className="flex flex-col gap-2">
        <Label htmlFor="review-suspect-note">处理结果</Label>
        <Textarea
          id="review-suspect-note"
          maxLength={255}
          placeholder="选填, 如已封禁、正常组队, 最多 255 个字符"
          value={note}
          onChange={(e) => setNote(e.target.value)}
          disabled={pending}
        />
      </div>
      <AlertDialogFooter>
        <AlertDialogCancel disabled={pending}>取消</AlertDialogCancel>
        <AlertDialogAction disabled={pending} onClick={handleConfirm}>
          {pending && <LoaderCircle className="animate-spin" />}
          标记已处理
        </AlertDialogAction>
      </AlertDialogFooter>
    </>
  );
}
