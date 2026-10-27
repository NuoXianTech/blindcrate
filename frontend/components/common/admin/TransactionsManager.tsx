"use client";

import { useCallback, useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import { Backpack, LoaderCircle, Search, Wallet } from "lucide-react";
import { toast } from "sonner";
import { cn } from "cn";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useAuth } from "@/components/common/auth/AuthProvider";
import { Pagination } from "@/components/common/layout/Pagination";
import { CurrencyIcon } from "@/components/common/wallet/CurrencyIcon";
import {
  AdminService,
  type AdminInventoryTransaction,
  type AdminWalletTransaction,
  type Currency,
} from "@/lib/services/admin";
import { toErrorMessage } from "@/lib/services/core";
import { INVENTORY_BIZ_LABELS, WALLET_BIZ_LABELS } from "@/lib/biz";
import { formatAmount, formatDateTime } from "@/lib/format";

const PAGE_SIZE = 20;
const ALL = "all";

/** 数字 ID 筛选, 空表示不筛选 */
type IDFilters = Record<string, number | undefined>;

function parseID(value: string | null | undefined) {
  return value && /^\d+$/.test(value) ? Number(value) : undefined;
}

/** 流水查询: 全部用户的余额流水和背包流水, 可从用户管理带 ?tab=&user_id= 跳转过来 */
export function TransactionsManager() {
  const searchParams = useSearchParams();
  const [tab, setTab] = useState(searchParams.get("tab") === "inventory" ? "inventory" : "wallet");
  const userID = parseID(searchParams.get("user_id"));

  return (
    <div className="flex flex-col gap-4">
      <h1 className="text-2xl font-bold">流水查询</h1>
      <Tabs value={tab} onValueChange={setTab}>
        <TabsList>
          <TabsTrigger value="wallet">
            <Wallet />
            余额流水
          </TabsTrigger>
          <TabsTrigger value="inventory">
            <Backpack />
            背包流水
          </TabsTrigger>
        </TabsList>
        <TabsContent value="wallet" className="mt-2">
          <WalletTransactions initialUserID={userID} />
        </TabsContent>
        <TabsContent value="inventory" className="mt-2">
          <InventoryTransactions initialUserID={userID} />
        </TabsContent>
      </Tabs>
    </div>
  );
}

/** 用户、操作人等数字 ID 的筛选表单, 提交时才生效 */
function IDFilterForm({
  fields,
  values,
  onSubmit,
}: {
  fields: { key: string; label: string }[];
  values: IDFilters;
  onSubmit: (values: IDFilters) => void;
}) {
  const [inputs, setInputs] = useState<Record<string, string>>(() =>
    Object.fromEntries(fields.map((f) => [f.key, values[f.key]?.toString() ?? ""])),
  );

  const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const next: IDFilters = {};
    for (const f of fields) {
      const value = inputs[f.key].trim();
      if (value && !/^\d+$/.test(value)) {
        toast.error(`${f.label}只能填数字`);
        return;
      }
      next[f.key] = parseID(value);
    }
    onSubmit(next);
  };

  return (
    <form onSubmit={handleSubmit} className="flex flex-1 flex-wrap gap-2">
      {fields.map((f) => (
        <Input
          key={f.key}
          inputMode="numeric"
          placeholder={f.label}
          aria-label={f.label}
          value={inputs[f.key]}
          onChange={(e) => setInputs((prev) => ({ ...prev, [f.key]: e.target.value }))}
          className="w-28"
        />
      ))}
      <Button type="submit" variant="secondary">
        <Search />
        筛选
      </Button>
    </form>
  );
}

function BizTypeSelect({ value, labels, onChange }: { value: string; labels: Record<string, string>; onChange: (value: string) => void }) {
  return (
    <Select value={value} onValueChange={onChange}>
      <SelectTrigger className="sm:w-36">
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value={ALL}>全部类型</SelectItem>
        {Object.entries(labels).map(([v, label]) => (
          <SelectItem key={v} value={v}>
            {label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

/** 用户或操作人; 用户已不存在时只有 ID, 操作人为 0 表示系统操作 */
function Actor({ id, username, system }: { id: number; username: string; system?: boolean }) {
  if (system && id === 0) return <span className="text-muted-foreground">系统</span>;
  return (
    <span className="flex flex-col">
      <span>{username ? `@${username}` : "已删除用户"}</span>
      <span className="text-xs text-muted-foreground">#{id}</span>
    </span>
  );
}

function StateRow({ colSpan, loading, empty }: { colSpan: number; loading: boolean; empty: string }) {
  return (
    <TableRow>
      <TableCell colSpan={colSpan} className={cn("h-24 text-center", !loading && "text-muted-foreground")}>
        {loading ? <LoaderCircle className="mx-auto size-5 animate-spin text-primary" /> : empty}
      </TableCell>
    </TableRow>
  );
}

function WalletTransactions({ initialUserID }: { initialUserID?: number }) {
  const isAdmin = !!useAuth().user?.is_admin;
  const [ids, setIDs] = useState<IDFilters>({ user_id: initialUserID });
  const [currency, setCurrency] = useState(ALL);
  const [bizType, setBizType] = useState(ALL);
  const [current, setCurrent] = useState(1);
  const [data, setData] = useState<{ total: number; transactions: AdminWalletTransaction[] } | null>(null);
  const [loading, setLoading] = useState(true);
  const [currencies, setCurrencies] = useState<Currency[]>([]);

  useEffect(() => {
    if (!isAdmin) return;
    AdminService.listCurrencies()
      .then((data) => setCurrencies(data.currencies))
      .catch(() => {
        // 只影响货币名称和小数位, 失败时显示货币代码
      });
  }, [isAdmin]);

  const load = useCallback(async () => {
    // 确认是管理员后再拉取, 否则 AdminGuard 会跳走
    if (!isAdmin) return;
    setLoading(true);
    try {
      setData(
        await AdminService.listWalletTransactions({
          current,
          size: PAGE_SIZE,
          ...ids,
          currency: currency === ALL ? undefined : currency,
          biz_type: bizType === ALL ? undefined : bizType,
        }),
      );
    } catch (err) {
      toast.error(toErrorMessage(err));
    } finally {
      setLoading(false);
    }
  }, [current, ids, currency, bizType, isAdmin]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 筛选条件变化时重新拉取
    void load();
  }, [load]);

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-2 sm:flex-row">
        <IDFilterForm
          fields={[
            { key: "user_id", label: "用户 ID" },
            { key: "operator_id", label: "操作人 ID" },
          ]}
          values={ids}
          onSubmit={(next) => {
            setCurrent(1);
            setIDs(next);
          }}
        />
        <Select
          value={currency}
          onValueChange={(value) => {
            setCurrent(1);
            setCurrency(value);
          }}
        >
          <SelectTrigger className="sm:w-32">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL}>全部货币</SelectItem>
            {currencies.map((c) => (
              <SelectItem key={c.code} value={c.code}>
                <CurrencyIcon currency={c.code} />
                {c.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <BizTypeSelect
          value={bizType}
          labels={WALLET_BIZ_LABELS}
          onChange={(value) => {
            setCurrent(1);
            setBizType(value);
          }}
        />
      </div>

      <div className="rounded-xl border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>时间</TableHead>
              <TableHead>用户</TableHead>
              <TableHead>货币</TableHead>
              <TableHead className="text-right">变动</TableHead>
              <TableHead className="text-right">变动后余额</TableHead>
              <TableHead>类型</TableHead>
              <TableHead>操作人</TableHead>
              <TableHead>备注</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading && !data ? (
              <StateRow colSpan={8} loading empty="" />
            ) : data?.transactions.length ? (
              data.transactions.map((t) => {
                const c = currencies.find((c) => c.code === t.currency);
                const decimals = c?.decimals ?? 0;
                return (
                  <TableRow key={t.id}>
                    <TableCell className="text-muted-foreground">{formatDateTime(t.created_at)}</TableCell>
                    <TableCell>
                      <Actor id={t.user_id} username={t.username} />
                    </TableCell>
                    <TableCell>
                      <span className="flex items-center gap-1.5">
                        <CurrencyIcon currency={t.currency} className="size-4" />
                        {c?.name ?? t.currency}
                      </span>
                    </TableCell>
                    <TableCell className="text-right font-mono tabular-nums">
                      {t.amount !== 0 && (
                        <div className={cn(t.amount > 0 ? "text-quality-green" : "text-destructive")}>
                          {formatAmount(t.amount, decimals, { signed: true })}
                        </div>
                      )}
                      {t.frozen_amount !== 0 && (
                        <div className="text-xs text-muted-foreground">冻结 {formatAmount(t.frozen_amount, decimals, { signed: true })}</div>
                      )}
                    </TableCell>
                    <TableCell className="text-right font-mono text-muted-foreground tabular-nums">
                      {formatAmount(t.balance_after, decimals)}
                    </TableCell>
                    <TableCell>{WALLET_BIZ_LABELS[t.biz_type] ?? t.biz_type}</TableCell>
                    <TableCell>
                      <Actor id={t.operator_id} username={t.operator_username} system />
                    </TableCell>
                    <TableCell className="max-w-64 truncate text-muted-foreground" title={t.remark || undefined}>
                      {t.remark || "-"}
                    </TableCell>
                  </TableRow>
                );
              })
            ) : (
              <StateRow colSpan={8} loading={false} empty="没有找到流水" />
            )}
          </TableBody>
        </Table>
      </div>

      <Pagination current={current} size={PAGE_SIZE} total={data?.total ?? 0} onChange={setCurrent} />
    </div>
  );
}

function InventoryTransactions({ initialUserID }: { initialUserID?: number }) {
  const isAdmin = !!useAuth().user?.is_admin;
  const [ids, setIDs] = useState<IDFilters>({ user_id: initialUserID });
  const [bizType, setBizType] = useState(ALL);
  const [current, setCurrent] = useState(1);
  const [data, setData] = useState<{ total: number; transactions: AdminInventoryTransaction[] } | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    if (!isAdmin) return;
    setLoading(true);
    try {
      setData(
        await AdminService.listInventoryTransactions({
          current,
          size: PAGE_SIZE,
          ...ids,
          biz_type: bizType === ALL ? undefined : bizType,
        }),
      );
    } catch (err) {
      toast.error(toErrorMessage(err));
    } finally {
      setLoading(false);
    }
  }, [current, ids, bizType, isAdmin]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 筛选条件变化时重新拉取
    void load();
  }, [load]);

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-2 sm:flex-row">
        <IDFilterForm
          fields={[
            { key: "user_id", label: "用户 ID" },
            { key: "operator_id", label: "操作人 ID" },
            { key: "item_id", label: "物品 ID" },
          ]}
          values={ids}
          onSubmit={(next) => {
            setCurrent(1);
            setIDs(next);
          }}
        />
        <BizTypeSelect
          value={bizType}
          labels={INVENTORY_BIZ_LABELS}
          onChange={(value) => {
            setCurrent(1);
            setBizType(value);
          }}
        />
      </div>

      <div className="rounded-xl border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>时间</TableHead>
              <TableHead>用户</TableHead>
              <TableHead>物品</TableHead>
              <TableHead className="text-right">变动</TableHead>
              <TableHead className="text-right">变动后数量</TableHead>
              <TableHead>类型</TableHead>
              <TableHead>操作人</TableHead>
              <TableHead>备注</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading && !data ? (
              <StateRow colSpan={8} loading empty="" />
            ) : data?.transactions.length ? (
              data.transactions.map((t) => (
                <TableRow key={t.id}>
                  <TableCell className="text-muted-foreground">{formatDateTime(t.created_at)}</TableCell>
                  <TableCell>
                    <Actor id={t.user_id} username={t.username} />
                  </TableCell>
                  <TableCell>
                    <span className="flex flex-col">
                      <span>{t.item_name}</span>
                      <span className="text-xs text-muted-foreground">#{t.item_id}</span>
                    </span>
                  </TableCell>
                  <TableCell
                    className={cn("text-right font-mono tabular-nums", t.quantity > 0 ? "text-quality-green" : "text-destructive")}
                  >
                    {t.quantity > 0 ? `+${t.quantity}` : t.quantity}
                  </TableCell>
                  <TableCell className="text-right font-mono text-muted-foreground tabular-nums">{t.quantity_after}</TableCell>
                  <TableCell>{INVENTORY_BIZ_LABELS[t.biz_type] ?? t.biz_type}</TableCell>
                  <TableCell>
                    <Actor id={t.operator_id} username={t.operator_username} system />
                  </TableCell>
                  <TableCell className="max-w-64 truncate text-muted-foreground" title={t.remark || undefined}>
                    {t.remark || "-"}
                  </TableCell>
                </TableRow>
              ))
            ) : (
              <StateRow colSpan={8} loading={false} empty="没有找到记录" />
            )}
          </TableBody>
        </Table>
      </div>

      <Pagination current={current} size={PAGE_SIZE} total={data?.total ?? 0} onChange={setCurrent} />
    </div>
  );
}
