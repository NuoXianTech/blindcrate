"use client";

import { useCallback, useEffect, useState } from "react";
import { LoaderCircle, Search } from "lucide-react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { useAuth } from "@/components/common/auth/AuthProvider";
import { Pagination } from "@/components/common/layout/Pagination";
import { AdminService, type AuditLog, type AuditScope } from "@/lib/services/admin";
import { toErrorMessage } from "@/lib/services/core";
import { formatDateTime } from "@/lib/format";

const PAGE_SIZE = 20;
const ALL = "all";

/** 操作名称, 与后端 internal/audit 的 Action 常量对应 */
const ACTION_LABELS: Record<string, string> = {
  "auth.login": "登录",
  "auth.login_failed": "登录失败",
  "auth.register": "注册",
  "auth.logout": "退出登录",
  "auth.change_password": "修改密码",
  "auth.verify_email": "绑定邮箱",
  "auth.update_profile": "修改资料",
  "admin.user_update": "修改用户资料",
  "admin.user_ban": "封禁用户",
  "admin.user_unban": "解除封禁",
  "admin.user_disable": "停用用户",
  "admin.user_enable": "启用用户",
};

const SCOPE_LABELS: Record<AuditScope, string> = { user: "用户", admin: "管理员" };

export function AuditLogsManager() {
  const isAdmin = !!useAuth().user?.is_admin;
  const [scope, setScope] = useState<string>(ALL);
  const [action, setAction] = useState<string>(ALL);
  const [actorInput, setActorInput] = useState("");
  const [actorID, setActorID] = useState<number | undefined>();
  const [current, setCurrent] = useState(1);
  const [data, setData] = useState<{ total: number; logs: AuditLog[] } | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    // 确认是管理员后再拉取, 否则 AdminGuard 会跳走
    if (!isAdmin) return;
    setLoading(true);
    try {
      setData(
        await AdminService.listAuditLogs({
          current,
          size: PAGE_SIZE,
          scope: scope === ALL ? undefined : (scope as AuditScope),
          action: action === ALL ? undefined : action,
          actor_id: actorID,
        }),
      );
    } catch (err) {
      toast.error(toErrorMessage(err));
    } finally {
      setLoading(false);
    }
  }, [current, scope, action, actorID, isAdmin]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 筛选条件变化时重新拉取
    void load();
  }, [load]);

  const handleActorSearch = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const value = actorInput.trim();
    if (value && !/^\d+$/.test(value)) {
      toast.error("操作人 ID 只能填数字");
      return;
    }
    setCurrent(1);
    setActorID(value ? Number(value) : undefined);
  };

  return (
    <div className="flex flex-col gap-4">
      <h1 className="text-2xl font-bold">操作日志</h1>

      <div className="flex flex-col gap-2 sm:flex-row">
        <form onSubmit={handleActorSearch} className="flex flex-1 gap-2">
          <Input
            inputMode="numeric"
            placeholder="操作人 ID"
            value={actorInput}
            onChange={(e) => setActorInput(e.target.value)}
            className="sm:max-w-40"
          />
          <Button type="submit" variant="secondary">
            <Search />
            筛选
          </Button>
        </form>
        <Select
          value={scope}
          onValueChange={(value) => {
            setCurrent(1);
            setScope(value);
          }}
        >
          <SelectTrigger className="sm:w-32">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL}>全部来源</SelectItem>
            <SelectItem value="user">用户</SelectItem>
            <SelectItem value="admin">管理员</SelectItem>
          </SelectContent>
        </Select>
        <Select
          value={action}
          onValueChange={(value) => {
            setCurrent(1);
            setAction(value);
          }}
        >
          <SelectTrigger className="sm:w-40">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL}>全部操作</SelectItem>
            {Object.entries(ACTION_LABELS).map(([value, label]) => (
              <SelectItem key={value} value={value}>
                {label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="rounded-xl border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>时间</TableHead>
              <TableHead>操作人</TableHead>
              <TableHead>操作</TableHead>
              <TableHead>对象</TableHead>
              <TableHead>详情</TableHead>
              <TableHead>IP</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading && !data ? (
              <TableRow>
                <TableCell colSpan={6} className="h-24 text-center">
                  <LoaderCircle className="mx-auto size-5 animate-spin text-primary" />
                </TableCell>
              </TableRow>
            ) : data?.logs.length ? (
              data.logs.map((log) => (
                <TableRow key={log.id}>
                  <TableCell className="text-muted-foreground">{formatDateTime(log.created_at)}</TableCell>
                  <TableCell>
                    {log.actor_id ? (
                      <span>
                        {log.actor_username || "已删除"}
                        <span className="ml-1 text-xs text-muted-foreground">#{log.actor_id}</span>
                      </span>
                    ) : (
                      <span className="text-muted-foreground">未登录</span>
                    )}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <Badge variant={log.scope === "admin" ? "default" : "secondary"}>{SCOPE_LABELS[log.scope]}</Badge>
                      <span className={log.action === "auth.login_failed" ? "text-destructive" : undefined}>
                        {ACTION_LABELS[log.action] ?? log.action}
                      </span>
                    </div>
                  </TableCell>
                  <TableCell className="text-muted-foreground">
                    {log.target_type ? `${log.target_type} #${log.target_id}` : "-"}
                  </TableCell>
                  <TableCell className="max-w-64">
                    {log.detail && Object.keys(log.detail).length > 0 ? (
                      <code className="block truncate font-mono text-xs text-muted-foreground" title={JSON.stringify(log.detail, null, 2)}>
                        {JSON.stringify(log.detail)}
                      </code>
                    ) : (
                      <span className="text-muted-foreground">-</span>
                    )}
                  </TableCell>
                  <TableCell className="font-mono text-xs text-muted-foreground" title={log.user_agent}>
                    {log.ip || "-"}
                  </TableCell>
                </TableRow>
              ))
            ) : (
              <TableRow>
                <TableCell colSpan={6} className="h-24 text-center text-muted-foreground">
                  没有日志
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>

      <Pagination current={current} size={PAGE_SIZE} total={data?.total ?? 0} onChange={setCurrent} />
    </div>
  );
}
