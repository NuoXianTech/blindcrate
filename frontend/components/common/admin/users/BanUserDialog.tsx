'use client';

import {useState} from 'react';
import {toast} from 'sonner';
import {Button} from '@/components/ui/button';
import {Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle} from '@/components/ui/dialog';
import {Input} from '@/components/ui/input';
import {Label} from '@/components/ui/label';
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select';
import {SubmitButton} from '@/components/common/form';
import {useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';
import type {AdminUser} from '@/lib/services/admin';
import {BAN_DURATIONS, banUntil} from './constants';
import type {UserDialogProps} from './EditUserDialog';

/**
 * 封禁用户: 选择时长和原因, 到期自动解除
 */
export function BanUserDialog({user, onClose, onSaved}: UserDialogProps) {
  return (
    <Dialog open={!!user} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>封禁 {user?.username}</DialogTitle>
          <DialogDescription>封禁后其所有设备会下线, 到期自动解除。永久封禁请使用停用。</DialogDescription>
        </DialogHeader>
        {user && <BanUserForm key={user.id} user={user} onClose={onClose} onSaved={onSaved} />}
      </DialogContent>
    </Dialog>
  );
}

/**
 * 封禁表单
 */
function BanUserForm({user, onClose, onSaved}: UserDialogProps & { user: AdminUser }) {
  const {pending, submit} = useSubmit();
  const [duration, setDuration] = useState(BAN_DURATIONS[1].value);
  const [reason, setReason] = useState('');

  const handleSubmit = submit(async () => {
    const ms = BAN_DURATIONS.find((d) => d.value === duration)!.ms;
    await services.admin.banUser(user.id, banUntil(ms), reason.trim());
    toast.success(`已封禁 ${user.username}`);
    onSaved();
    onClose();
  });

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4" noValidate>
      <div className="flex flex-col gap-2">
        <Label>封禁时长</Label>
        <Select value={duration} onValueChange={setDuration} disabled={pending}>
          <SelectTrigger className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {BAN_DURATIONS.map((d) => (
              <SelectItem key={d.value} value={d.value}>
                {d.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <div className="flex flex-col gap-2">
        <Label htmlFor="ban-reason">原因</Label>
        <Input
          id="ban-reason"
          maxLength={255}
          placeholder="选填, 最多 255 个字符"
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          disabled={pending}
        />
      </div>
      <DialogFooter>
        <Button type="button" variant="outline" onClick={onClose} disabled={pending}>
          取消
        </Button>
        <SubmitButton pending={pending} variant="destructive">
          封禁
        </SubmitButton>
      </DialogFooter>
    </form>
  );
}
