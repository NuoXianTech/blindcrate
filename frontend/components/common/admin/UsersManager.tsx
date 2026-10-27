"use client";

import { useCallback, useEffect, useState } from "react";
import { Backpack, Ban, LoaderCircle, Pencil, Power, Search, ShieldCheck, Wallet } from "lucide-react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { UserInventoryDialog, UserWalletDialog } from "@/components/common/admin/UserAssetsDialogs";
import { useAuth } from "@/components/common/auth/AuthProvider";
import { SubmitButton } from "@/components/common/form/SubmitButton";
import { check, useSubmit } from "@/components/common/form/use-submit";
import { Pagination } from "@/components/common/layout/Pagination";
import { UserAvatar } from "@/components/common/user/UserAvatar";
import { AdminService, type AdminUser, type ListUsersParams } from "@/lib/services/admin";
import { toErrorMessage } from "@/lib/services/core";
import { formatDateTime } from "@/lib/format";
import { checkNickname, checkUsername } from "@/lib/validation";

const PAGE_SIZE = 20;

type StatusFilter = "all" | "admin" | "banned" | "disabled";

const STATUS_FILTERS: { value: StatusFilter; label: string }[] = [
  { value: "all", label: "全部用户" },
  { value: "admin", label: "管理员" },
  { value: "banned", label: "封禁中" },
  { value: "disabled", label: "已停用" },
];

const BAN_DURATIONS = [
  { value: "1h", label: "1 小时", ms: 3600_000 },
  { value: "1d", label: "1 天", ms: 86400_000 },
  { value: "7d", label: "7 天", ms: 7 * 86400_000 },
  { value: "30d", label: "30 天", ms: 30 * 86400_000 },
];

/** 从现在起 ms 毫秒后的时间 */
function banUntil(ms: number) {
  return new Date(Date.now() + ms).toISOString();
}

function isBanned(user: AdminUser) {
  return !!user.banned_until && new Date(user.banned_until) > new Date();
}

function statusParams(status: StatusFilter): Partial<ListUsersParams> {
  switch (status) {
    case "admin":
      return { is_admin: true };
    case "banned":
      return { is_banned: true };
    case "disabled":
      return { is_disabled: true };
    default:
      return {};
  }
}

export function UsersManager() {
  const { user: me } = useAuth();
  const isAdmin = !!me?.is_admin;
  const [keywordInput, setKeywordInput] = useState("");
  const [keyword, setKeyword] = useState("");
  const [status, setStatus] = useState<StatusFilter>("all");
  const [current, setCurrent] = useState(1);
  const [data, setData] = useState<{ total: number; users: AdminUser[] } | null>(null);
  const [loading, setLoading] = useState(true);

  const [editing, setEditing] = useState<AdminUser | null>(null);
  const [banning, setBanning] = useState<AdminUser | null>(null);
  const [disabling, setDisabling] = useState<AdminUser | null>(null);
  const [wallet, setWallet] = useState<AdminUser | null>(null);
  const [inventory, setInventory] = useState<AdminUser | null>(null);

  const load = useCallback(async () => {
    // 确认是管理员后再拉取, 否则 AdminGuard 会跳走
    if (!isAdmin) return;
    setLoading(true);
    try {
      setData(await AdminService.listUsers({ current, size: PAGE_SIZE, keyword: keyword || undefined, ...statusParams(status) }));
    } catch (err) {
      toast.error(toErrorMessage(err));
    } finally {
      setLoading(false);
    }
  }, [current, keyword, status, isAdmin]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 筛选条件变化时重新拉取
    void load();
  }, [load]);

  const handleSearch = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setCurrent(1);
    setKeyword(keywordInput.trim());
  };

  const runAction = async (action: () => Promise<unknown>, success: string) => {
    try {
      await action();
      toast.success(success);
      await load();
    } catch (err) {
      toast.error(toErrorMessage(err));
    }
  };

  return (
    <div className="flex flex-col gap-4">
      <h1 className="text-2xl font-bold">用户管理</h1>

      <div className="flex flex-col gap-2 sm:flex-row">
        <form onSubmit={handleSearch} className="flex flex-1 gap-2">
          <Input
            placeholder="用户名或邮箱前缀"
            value={keywordInput}
            onChange={(e) => setKeywordInput(e.target.value)}
            className="sm:max-w-xs"
          />
          <Button type="submit" variant="secondary">
            <Search />
            搜索
          </Button>
        </form>
        <Select
          value={status}
          onValueChange={(value) => {
            setCurrent(1);
            setStatus(value as StatusFilter);
          }}
        >
          <SelectTrigger className="sm:w-36">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {STATUS_FILTERS.map((f) => (
              <SelectItem key={f.value} value={f.value}>
                {f.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="rounded-xl border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>用户</TableHead>
              <TableHead>邮箱</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>最后登录</TableHead>
              <TableHead>注册时间</TableHead>
              <TableHead className="text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading && !data ? (
              <TableRow>
                <TableCell colSpan={6} className="h-24 text-center">
                  <LoaderCircle className="mx-auto size-5 animate-spin text-primary" />
                </TableCell>
              </TableRow>
            ) : data?.users.length ? (
              data.users.map((u) => {
                const banned = isBanned(u);
                const self = u.id === me?.id;
                return (
                  <TableRow key={u.id}>
                    <TableCell>
                      <div className="flex items-center gap-3">
                        <UserAvatar user={u} className="size-8 text-xs" />
                        <div className="flex flex-col">
                          <span className="font-medium">{u.nickname}</span>
                          <span className="flex gap-2 text-xs text-muted-foreground">
                            <span>@{u.username}</span>
                            <span>#{u.id}</span>
                          </span>
                        </div>
                      </div>
                    </TableCell>
                    <TableCell className="text-muted-foreground">{u.email || "-"}</TableCell>
                    <TableCell>
                      <div className="flex flex-wrap gap-1">
                        {u.is_admin && (
                          <Badge>
                            <ShieldCheck />
                            管理员
                          </Badge>
                        )}
                        {u.is_disabled && <Badge variant="destructive">已停用</Badge>}
                        {banned && (
                          <Badge variant="outline" className="border-destructive/50 text-destructive" title={u.ban_reason || undefined}>
                            封禁至 {formatDateTime(u.banned_until)}
                          </Badge>
                        )}
                        {!u.is_admin && !u.is_disabled && !banned && <Badge variant="secondary">正常</Badge>}
                      </div>
                    </TableCell>
                    <TableCell className="text-muted-foreground">{formatDateTime(u.last_login_at)}</TableCell>
                    <TableCell className="text-muted-foreground">{formatDateTime(u.created_at)}</TableCell>
                    <TableCell>
                      <div className="flex justify-end gap-1">
                        <Button variant="ghost" size="icon" title="编辑资料" aria-label="编辑资料" onClick={() => setEditing(u)}>
                          <Pencil />
                        </Button>
                        {/* 后端不允许管理员封禁、停用自己, 也不能给自己调整钱包和背包 */}
                        {!self && (
                          <>
                            <Button variant="ghost" size="icon" title="钱包" aria-label="钱包" onClick={() => setWallet(u)}>
                              <Wallet />
                            </Button>
                            <Button variant="ghost" size="icon" title="背包" aria-label="背包" onClick={() => setInventory(u)}>
                              <Backpack />
                            </Button>
                            <Button
                              variant="ghost"
                              size="icon"
                              title={banned ? "解除封禁" : "封禁"}
                              aria-label={banned ? "解除封禁" : "封禁"}
                              className={banned ? "text-destructive" : undefined}
                              onClick={() =>
                                banned ? runAction(() => AdminService.unbanUser(u.id), `已解除 ${u.username} 的封禁`) : setBanning(u)
                              }
                            >
                              <Ban />
                            </Button>
                            <Button
                              variant="ghost"
                              size="icon"
                              title={u.is_disabled ? "启用" : "停用"}
                              aria-label={u.is_disabled ? "启用" : "停用"}
                              className={u.is_disabled ? "text-destructive" : undefined}
                              onClick={() =>
                                u.is_disabled
                                  ? runAction(() => AdminService.setUserDisabled(u.id, false), `已启用 ${u.username}`)
                                  : setDisabling(u)
                              }
                            >
                              <Power />
                            </Button>
                          </>
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                );
              })
            ) : (
              <TableRow>
                <TableCell colSpan={6} className="h-24 text-center text-muted-foreground">
                  没有找到用户
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>

      <Pagination current={current} size={PAGE_SIZE} total={data?.total ?? 0} onChange={setCurrent} />

      <EditUserDialog user={editing} onClose={() => setEditing(null)} onSaved={load} />
      <BanUserDialog user={banning} onClose={() => setBanning(null)} onSaved={load} />
      <DisableUserDialog user={disabling} onClose={() => setDisabling(null)} onSaved={load} />
      <UserWalletDialog user={wallet} onClose={() => setWallet(null)} />
      <UserInventoryDialog user={inventory} onClose={() => setInventory(null)} />
    </div>
  );
}

function EditUserDialog({ user, onClose, onSaved }: { user: AdminUser | null; onClose: () => void; onSaved: () => void }) {
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

function EditUserForm({ user, onClose, onSaved }: { user: AdminUser; onClose: () => void; onSaved: () => void }) {
  const { user: me, refresh } = useAuth();
  const { pending, submit } = useSubmit();
  const [username, setUsername] = useState(user.username);
  const [nickname, setNickname] = useState(user.nickname);
  const [avatarURL, setAvatarURL] = useState(user.avatar_url);

  const handleSubmit = submit(async () => {
    const body: { username?: string; nickname?: string; avatar_url?: string } = {};
    if (username.trim() !== user.username) body.username = username.trim();
    if (nickname.trim() !== user.nickname) body.nickname = nickname.trim();
    if (avatarURL.trim() !== user.avatar_url) body.avatar_url = avatarURL.trim();
    check(Object.keys(body).length === 0 ? "资料没有变化, 无需保存" : null);
    if (body.username !== undefined) check(checkUsername(body.username));
    if (body.nickname !== undefined) check(checkNickname(body.nickname));
    await AdminService.updateUser(user.id, body);
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

function BanUserDialog({ user, onClose, onSaved }: { user: AdminUser | null; onClose: () => void; onSaved: () => void }) {
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

function BanUserForm({ user, onClose, onSaved }: { user: AdminUser; onClose: () => void; onSaved: () => void }) {
  const { pending, submit } = useSubmit();
  const [duration, setDuration] = useState(BAN_DURATIONS[1].value);
  const [reason, setReason] = useState("");

  const handleSubmit = submit(async () => {
    const ms = BAN_DURATIONS.find((d) => d.value === duration)!.ms;
    await AdminService.banUser(user.id, banUntil(ms), reason.trim());
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

function DisableUserDialog({ user, onClose, onSaved }: { user: AdminUser | null; onClose: () => void; onSaved: () => void }) {
  const { pending, submit } = useSubmit();

  const handleSubmit = submit(async () => {
    if (!user) return;
    await AdminService.setUserDisabled(user.id, true);
    toast.success(`已停用 ${user.username}`);
    onSaved();
    onClose();
  });

  return (
    <Dialog open={!!user} onOpenChange={(open) => !open && !pending && onClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>停用 {user?.username}</DialogTitle>
          <DialogDescription>停用后该账号无法登录, 所有设备会立即下线, 直到重新启用。</DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit}>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose} disabled={pending}>
              取消
            </Button>
            <SubmitButton pending={pending} variant="destructive">
              停用
            </SubmitButton>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
