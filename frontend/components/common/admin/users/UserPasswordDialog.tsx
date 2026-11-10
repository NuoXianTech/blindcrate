'use client';

import {useState} from 'react';
import {toast} from 'sonner';
import {Button} from '@/components/ui/button';
import {Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle} from '@/components/ui/dialog';
import {Input} from '@/components/ui/input';
import {Label} from '@/components/ui/label';
import {useAuth} from '@/components/common/auth';
import {SubmitButton} from '@/components/common/form';
import {check, useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';
import type {AdminUser} from '@/lib/services/auth';
import {checkPassword, checkPasswordConfirm} from '@/lib/validation';

/**
 * 修改密码: 其他用户不需要原密码, 修改后其所有设备下线; 自己的需要原密码, 当前设备保持登录
 */
export function UserPasswordDialog({user, onClose}: { user: AdminUser | null; onClose: () => void }) {
  const {user: me} = useAuth();
  const self = !!user && user.id === me?.id;
  return (
    <Dialog open={!!user} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>修改 {user?.username} 的密码</DialogTitle>
          <DialogDescription>
            {self ? '修改自己的密码需要原密码, 修改后当前设备保持登录, 其他设备下线' : '不需要原密码, 修改后其所有设备会下线'}
          </DialogDescription>
        </DialogHeader>
        {/* key 保证切换用户时表单重置 */}
        {user && <PasswordForm key={user.id} user={user} self={self} onClose={onClose} />}
      </DialogContent>
    </Dialog>
  );
}

/**
 * 修改密码表单, 自己的密码走个人接口核对原密码
 */
function PasswordForm({user, self, onClose}: { user: AdminUser; self: boolean; onClose: () => void }) {
  const {pending, submit} = useSubmit();
  const [oldPassword, setOldPassword] = useState('');
  const [password, setPassword] = useState('');
  const [confirm, setConfirm] = useState('');

  const handleSubmit = submit(async () => {
    check(checkPassword(password));
    check(checkPasswordConfirm(password, confirm));
    if (self) {
      await services.auth.changePassword({old_password: oldPassword, new_password: password});
      toast.success('密码已修改, 其他设备已下线');
    } else {
      await services.authAdmin.setUserPassword(user.id, password);
      toast.success(`已修改 ${user.username} 的密码, 其所有设备已下线`);
    }
    onClose();
  });

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4" noValidate>
      {self && (
        <div className="flex flex-col gap-2">
          <Label htmlFor="password-old">原密码</Label>
          <Input
            id="password-old"
            type="password"
            autoComplete="current-password"
            value={oldPassword}
            onChange={(e) => setOldPassword(e.target.value)}
            disabled={pending}
          />
        </div>
      )}
      <div className="flex flex-col gap-2">
        <Label htmlFor="password-new">新密码</Label>
        <Input
          id="password-new"
          type="password"
          autoComplete="new-password"
          placeholder="8-72 位"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          disabled={pending}
        />
      </div>
      <div className="flex flex-col gap-2">
        <Label htmlFor="password-confirm">确认新密码</Label>
        <Input
          id="password-confirm"
          type="password"
          autoComplete="new-password"
          value={confirm}
          onChange={(e) => setConfirm(e.target.value)}
          disabled={pending}
        />
      </div>
      <DialogFooter>
        <Button type="button" variant="outline" onClick={onClose} disabled={pending}>
          取消
        </Button>
        <SubmitButton pending={pending}>修改密码</SubmitButton>
      </DialogFooter>
    </form>
  );
}
