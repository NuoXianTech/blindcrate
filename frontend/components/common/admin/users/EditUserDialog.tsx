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
import type {AdminUser, UpdateUserRequest} from '@/lib/services/admin';
import {checkNickname, checkUsername} from '@/lib/validation';

/**
 * 用户对话框的Props接口
 */
export interface UserDialogProps {
  /** 要操作的用户, null 时关闭 */
  user: AdminUser | null;
  /** 关闭回调 */
  onClose: () => void;
  /** 保存成功后刷新列表 */
  onSaved: () => void;
}

/**
 * 编辑用户名、昵称和头像
 */
export function EditUserDialog({user, onClose, onSaved}: UserDialogProps) {
  return (
    <Dialog open={!!user} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>编辑资料</DialogTitle>
          <DialogDescription>修改 #{user?.id} 的用户名、昵称和头像</DialogDescription>
        </DialogHeader>
        {/* key 保证切换用户时表单重置 */}
        {user && <EditUserForm key={user.id} user={user} onClose={onClose} onSaved={onSaved} />}
      </DialogContent>
    </Dialog>
  );
}

/**
 * 编辑资料表单, 只提交改动的字段
 */
function EditUserForm({user, onClose, onSaved}: UserDialogProps & { user: AdminUser }) {
  const {user: me, refresh} = useAuth();
  const {pending, submit} = useSubmit();
  const [username, setUsername] = useState(user.username);
  const [nickname, setNickname] = useState(user.nickname);
  const [avatarURL, setAvatarURL] = useState(user.avatar_url);

  const handleSubmit = submit(async () => {
    const body: UpdateUserRequest = {};
    if (username.trim() !== user.username) body.username = username.trim();
    if (nickname.trim() !== user.nickname) body.nickname = nickname.trim();
    if (avatarURL.trim() !== user.avatar_url) body.avatar_url = avatarURL.trim();
    check(Object.keys(body).length === 0 ? '资料没有变化, 无需保存' : null);
    if (body.username !== undefined) check(checkUsername(body.username));
    if (body.nickname !== undefined) check(checkNickname(body.nickname));
    await services.admin.updateUser(user.id, body);
    toast.success(`已保存 ${body.username ?? user.username} 的资料`);
    // 改的是自己时同步 dock 上的信息
    if (user.id === me?.id) await refresh();
    onSaved();
    onClose();
  });

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4" noValidate>
      <div className="flex flex-col gap-2">
        <Label htmlFor="edit-username">用户名</Label>
        <Input id="edit-username" value={username} onChange={(e) => setUsername(e.target.value)} disabled={pending} />
      </div>
      <div className="flex flex-col gap-2">
        <Label htmlFor="edit-nickname">昵称</Label>
        <Input id="edit-nickname" value={nickname} onChange={(e) => setNickname(e.target.value)} disabled={pending} />
      </div>
      <div className="flex flex-col gap-2">
        <Label htmlFor="edit-avatar">头像地址</Label>
        <Input
          id="edit-avatar"
          type="url"
          placeholder="留空则清除头像"
          value={avatarURL}
          onChange={(e) => setAvatarURL(e.target.value)}
          disabled={pending}
        />
      </div>
      <DialogFooter>
        <Button type="button" variant="outline" onClick={onClose} disabled={pending}>
          取消
        </Button>
        <SubmitButton pending={pending}>保存资料</SubmitButton>
      </DialogFooter>
    </form>
  );
}
