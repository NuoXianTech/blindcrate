import {Backpack, Ban, Pencil, Power, ShieldCheck, Wallet} from 'lucide-react';
import {Badge} from '@/components/ui/badge';
import {Button} from '@/components/ui/button';
import {TableCell, TableRow} from '@/components/ui/table';
import {UserAvatar} from '@/components/common/user';
import type {AdminUser} from '@/lib/services/admin';
import {formatDateTime} from '@/lib/utils';
import {isBanned} from './constants';

/**
 * 用户行的Props接口
 */
interface UserRowProps {
  user: AdminUser;
  /** 是否是当前登录的管理员自己 */
  self: boolean;
  onEdit: () => void;
  onWallet: () => void;
  onInventory: () => void;
  onBan: () => void;
  onUnban: () => void;
  onDisable: () => void;
  onEnable: () => void;
}

/**
 * 用户列表的一行: 资料、状态和操作按钮
 */
export function UserRow({user: u, self, onEdit, onWallet, onInventory, onBan, onUnban, onDisable, onEnable}: UserRowProps) {
  const banned = isBanned(u);
  return (
    <TableRow>
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
      <TableCell className="text-muted-foreground">{u.email || '-'}</TableCell>
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
          <Button variant="ghost" size="icon" title="编辑资料" aria-label="编辑资料" onClick={onEdit}>
            <Pencil />
          </Button>
          {/* 后端不允许管理员封禁、停用自己, 也不能给自己调整钱包和背包 */}
          {!self && (
            <>
              <Button variant="ghost" size="icon" title="钱包" aria-label="钱包" onClick={onWallet}>
                <Wallet />
              </Button>
              <Button variant="ghost" size="icon" title="背包" aria-label="背包" onClick={onInventory}>
                <Backpack />
              </Button>
              <Button
                variant="ghost"
                size="icon"
                title={banned ? '解除封禁' : '封禁'}
                aria-label={banned ? '解除封禁' : '封禁'}
                className={banned ? 'text-destructive' : undefined}
                onClick={banned ? onUnban : onBan}
              >
                <Ban />
              </Button>
              <Button
                variant="ghost"
                size="icon"
                title={u.is_disabled ? '启用' : '停用'}
                aria-label={u.is_disabled ? '启用' : '停用'}
                className={u.is_disabled ? 'text-destructive' : undefined}
                onClick={u.is_disabled ? onEnable : onDisable}
              >
                <Power />
              </Button>
            </>
          )}
        </div>
      </TableCell>
    </TableRow>
  );
}
