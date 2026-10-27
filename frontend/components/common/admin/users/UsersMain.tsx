'use client';

import {useCallback, useEffect, useState} from 'react';
import {Search} from 'lucide-react';
import {toast} from 'sonner';
import {Button} from '@/components/ui/button';
import {Input} from '@/components/ui/input';
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select';
import {Table, TableBody, TableHead, TableHeader, TableRow} from '@/components/ui/table';
import {useAuth} from '@/components/common/auth';
import {Pagination, TableEmpty, TableSkeleton} from '@/components/common/layout';
import services from '@/lib/services';
import type {AdminUser} from '@/lib/services/admin';
import {toErrorMessage} from '@/lib/services/core';
import {BanUserDialog} from './BanUserDialog';
import {DisableUserDialog} from './DisableUserDialog';
import {EditUserDialog} from './EditUserDialog';
import {UserInventoryDialog} from './UserInventoryDialog';
import {UserRow} from './UserRow';
import {UserWalletDialog} from './UserWalletDialog';
import {PAGE_SIZE, STATUS_FILTERS, statusParams, type StatusFilter} from './constants';

/**
 * 用户管理: 搜索、状态筛选, 编辑资料、封禁、停用, 调整钱包和背包
 */
export function UsersMain() {
  const {user: me} = useAuth();
  const isAdmin = !!me?.is_admin;
  const [keywordInput, setKeywordInput] = useState('');
  const [keyword, setKeyword] = useState('');
  const [status, setStatus] = useState<StatusFilter>('all');
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
      setData(await services.admin.listUsers({current, size: PAGE_SIZE, keyword: keyword || undefined, ...statusParams(status)}));
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
              <TableSkeleton cols={6} />
            ) : data?.users.length ? (
              data.users.map((u) => (
                <UserRow
                  key={u.id}
                  user={u}
                  self={u.id === me?.id}
                  onEdit={() => setEditing(u)}
                  onWallet={() => setWallet(u)}
                  onInventory={() => setInventory(u)}
                  onBan={() => setBanning(u)}
                  onUnban={() => runAction(() => services.admin.unbanUser(u.id), `已解除 ${u.username} 的封禁`)}
                  onDisable={() => setDisabling(u)}
                  onEnable={() => runAction(() => services.admin.setUserDisabled(u.id, false), `已启用 ${u.username}`)}
                />
              ))
            ) : (
              <TableEmpty colSpan={6}>没有找到用户</TableEmpty>
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
