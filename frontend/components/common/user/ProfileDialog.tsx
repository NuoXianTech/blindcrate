"use client";

import { useEffect, useState } from "react";
import { LoaderCircle } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useAuth } from "@/components/common/auth/AuthProvider";
import { UserAvatar } from "@/components/common/user/UserAvatar";
import { AuthService, type UserInfo } from "@/lib/services/auth";
import { ApiError } from "@/lib/services/core";

/** 发送验证码的冷却秒数, 与后端一致 */
const EMAIL_CODE_COOLDOWN = 60;

function errorMessage(err: unknown) {
  return err instanceof ApiError ? err.message : "请求失败, 请稍后再试";
}

/** 个人资料面板: 资料、密码、邮箱 */
export function ProfileDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (open: boolean) => void }) {
  const { user } = useAuth();
  if (!user) return null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader className="flex-row items-center gap-3 text-left">
          <UserAvatar user={user} className="size-12 text-lg" />
          <div className="flex min-w-0 flex-col gap-1">
            <DialogTitle className="truncate">{user.nickname}</DialogTitle>
            <DialogDescription className="truncate">
              @{user.username}
              {user.is_admin && <span className="ml-2 text-primary">管理员</span>}
            </DialogDescription>
          </div>
        </DialogHeader>
        {/* 每次打开重新挂载, 表单回到当前资料 */}
        {open && (
          <Tabs defaultValue="profile">
            <TabsList className="w-full">
              <TabsTrigger value="profile">资料</TabsTrigger>
              <TabsTrigger value="password">密码</TabsTrigger>
              <TabsTrigger value="email">邮箱</TabsTrigger>
            </TabsList>
            <TabsContent value="profile">
              <ProfileForm user={user} />
            </TabsContent>
            <TabsContent value="password">
              <PasswordForm />
            </TabsContent>
            <TabsContent value="email">
              <EmailForm user={user} />
            </TabsContent>
          </Tabs>
        )}
      </DialogContent>
    </Dialog>
  );
}

function ProfileForm({ user }: { user: UserInfo }) {
  const { setUser } = useAuth();
  const [nickname, setNickname] = useState(user.nickname);
  const [avatarURL, setAvatarURL] = useState(user.avatar_url);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    // 只提交改动过的字段
    const body: { nickname?: string; avatar_url?: string } = {};
    if (nickname.trim() !== user.nickname) body.nickname = nickname.trim();
    if (avatarURL.trim() !== user.avatar_url) body.avatar_url = avatarURL.trim();
    if (Object.keys(body).length === 0) {
      setError("没有修改");
      return;
    }
    if (body.nickname !== undefined && (body.nickname.length < 1 || [...body.nickname].length > 32)) {
      setError("昵称需为 1-32 个字符");
      return;
    }

    setError(null);
    setSaving(true);
    try {
      setUser(await AuthService.updateUserInfo(body));
      toast.success("资料已更新");
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setSaving(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4 pt-2" noValidate>
      <div className="flex flex-col gap-2">
        <Label htmlFor="profile-username">用户名</Label>
        <Input id="profile-username" value={user.username} disabled />
      </div>
      <div className="flex flex-col gap-2">
        <Label htmlFor="profile-nickname">昵称</Label>
        <Input
          id="profile-nickname"
          placeholder="1-32 个字符"
          value={nickname}
          onChange={(e) => setNickname(e.target.value)}
          disabled={saving}
        />
      </div>
      <div className="flex flex-col gap-2">
        <Label htmlFor="profile-avatar">头像地址</Label>
        <div className="flex items-center gap-3">
          <UserAvatar user={{ nickname: nickname || user.nickname, avatar_url: avatarURL.trim() }} className="size-9" />
          <Input
            id="profile-avatar"
            type="url"
            placeholder="https://..., 留空则清除头像"
            value={avatarURL}
            onChange={(e) => setAvatarURL(e.target.value)}
            disabled={saving}
          />
        </div>
      </div>
      <FormFooter error={error} pending={saving} label="保存" />
    </form>
  );
}

function PasswordForm() {
  const [oldPassword, setOldPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (newPassword.length < 8 || newPassword.length > 72) {
      setError("新密码长度需为 8-72 位");
      return;
    }
    if (newPassword !== confirm) {
      setError("两次输入的密码不一致");
      return;
    }

    setError(null);
    setSaving(true);
    try {
      await AuthService.changePassword({ old_password: oldPassword, new_password: newPassword });
      toast.success("密码已修改, 其他设备已下线");
      setOldPassword("");
      setNewPassword("");
      setConfirm("");
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setSaving(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4 pt-2" noValidate>
      <div className="flex flex-col gap-2">
        <Label htmlFor="old-password">原密码</Label>
        <Input
          id="old-password"
          type="password"
          autoComplete="current-password"
          value={oldPassword}
          onChange={(e) => setOldPassword(e.target.value)}
          disabled={saving}
        />
      </div>
      <div className="flex flex-col gap-2">
        <Label htmlFor="new-password">新密码</Label>
        <Input
          id="new-password"
          type="password"
          autoComplete="new-password"
          placeholder="8-72 位"
          value={newPassword}
          onChange={(e) => setNewPassword(e.target.value)}
          disabled={saving}
        />
      </div>
      <div className="flex flex-col gap-2">
        <Label htmlFor="confirm-password">确认新密码</Label>
        <Input
          id="confirm-password"
          type="password"
          autoComplete="new-password"
          value={confirm}
          onChange={(e) => setConfirm(e.target.value)}
          disabled={saving}
        />
      </div>
      <FormFooter error={error} pending={saving} label="修改密码" />
    </form>
  );
}

function EmailForm({ user }: { user: UserInfo }) {
  const { refresh } = useAuth();
  const [email, setEmail] = useState("");
  const [code, setCode] = useState("");
  const [cooldown, setCooldown] = useState(0);
  const [sending, setSending] = useState(false);
  const [verifying, setVerifying] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (cooldown <= 0) return;
    const timer = setTimeout(() => setCooldown(cooldown - 1), 1000);
    return () => clearTimeout(timer);
  }, [cooldown]);

  const handleSend = async () => {
    if (!email.trim()) {
      setError("请填写邮箱");
      return;
    }
    setError(null);
    setSending(true);
    try {
      await AuthService.sendEmailCode(email.trim());
      setCooldown(EMAIL_CODE_COOLDOWN);
      toast.success("验证码已发送, 10 分钟内有效");
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setSending(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (!email.trim() || !code.trim()) {
      setError("请填写邮箱和验证码");
      return;
    }
    setError(null);
    setVerifying(true);
    try {
      await AuthService.verifyEmail(email.trim(), code.trim());
      await refresh();
      toast.success("邮箱已绑定");
      setEmail("");
      setCode("");
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setVerifying(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4 pt-2" noValidate>
      <p className="text-sm text-muted-foreground">
        当前邮箱: <span className="text-foreground">{user.email || "未绑定"}</span>
        <br />
        绑定后可以用邮箱登录。
      </p>
      <div className="flex flex-col gap-2">
        <Label htmlFor="email">{user.email ? "新邮箱" : "邮箱"}</Label>
        <div className="flex gap-2">
          <Input
            id="email"
            type="email"
            autoComplete="email"
            placeholder="name@example.com"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            disabled={verifying}
          />
          <Button type="button" variant="secondary" onClick={handleSend} disabled={sending || cooldown > 0}>
            {sending && <LoaderCircle className="animate-spin" />}
            {cooldown > 0 ? `${cooldown}s` : "发送验证码"}
          </Button>
        </div>
      </div>
      <div className="flex flex-col gap-2">
        <Label htmlFor="email-code">验证码</Label>
        <Input
          id="email-code"
          inputMode="numeric"
          autoComplete="one-time-code"
          maxLength={6}
          placeholder="6 位数字"
          value={code}
          onChange={(e) => setCode(e.target.value)}
          disabled={verifying}
        />
      </div>
      <FormFooter error={error} pending={verifying} label="绑定邮箱" />
    </form>
  );
}

function FormFooter({ error, pending, label }: { error: string | null; pending: boolean; label: string }) {
  return (
    <>
      {error && (
        <p role="alert" className="text-sm text-destructive">
          {error}
        </p>
      )}
      <Button type="submit" className="w-full" disabled={pending}>
        {pending && <LoaderCircle className="animate-spin" />}
        {label}
      </Button>
    </>
  );
}
