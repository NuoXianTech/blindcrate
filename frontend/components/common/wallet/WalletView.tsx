"use client";

import { useCallback, useEffect, useState } from "react";
import { LoaderCircle, Snowflake } from "lucide-react";
import { toast } from "sonner";
import { cn } from "cn";
import { Badge } from "@/components/ui/badge";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { useAuth } from "@/components/common/auth/AuthProvider";
import { Pagination } from "@/components/common/layout/Pagination";
import { CurrencyIcon } from "@/components/common/wallet/CurrencyIcon";
import { toErrorMessage } from "@/lib/services/core";
import { WalletService, type BalanceItem, type WalletTransaction } from "@/lib/services/wallet";
import { formatAmount, formatDateTime } from "@/lib/format";

const PAGE_SIZE = 20;
const ALL = "all";

/** 业务类型名称, 与后端 internal/apps/wallet/constants.go 的 Biz 常量对应 */
const WALLET_BIZ_LABELS: Record<string, string> = {
  admin_adjust: "管理员调整",
};

/** 钱包: 各货币余额 + 余额流水 */
export function WalletView() {
  const { user } = useAuth();
  const [balances, setBalances] = useState<BalanceItem[] | null>(null);

  useEffect(() => {
    if (!user) return;
    WalletService.listBalances()
      .then((data) => setBalances(data.balances))
      .catch((err) => toast.error(toErrorMessage(err)));
  }, [user]);

  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-2xl font-bold">钱包</h1>

      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        {balances
          ? balances.map((b) => <BalanceCard key={b.currency} balance={b} />)
          : Array.from({ length: 2 }, (_, i) => <div key={i} className="h-28 animate-pulse rounded-xl border bg-card" />)}
      </div>

      <TransactionList balances={balances ?? []} />
    </div>
  );
}

function BalanceCard({ balance: b }: { balance: BalanceItem }) {
  return (
    <div className="flex flex-col gap-3 rounded-xl border bg-card p-4">
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <CurrencyIcon currency={b.currency} className="size-5" />
        <span>{b.name}</span>
        {!b.is_enabled && <Badge variant="outline">已停用</Badge>}
      </div>
      <span className="font-mono text-3xl font-bold tabular-nums">{formatAmount(b.balance, b.decimals)}</span>
      {b.frozen > 0 && (
        <span className="flex items-center gap-1 text-xs text-muted-foreground" title="冻结中的余额(如竞拍押金), 暂不可用">
          <Snowflake className="size-3" />
          冻结 {formatAmount(b.frozen, b.decimals)}
        </span>
      )}
    </div>
  );
}

function TransactionList({ balances }: { balances: BalanceItem[] }) {
  const { user } = useAuth();
  const [currency, setCurrency] = useState<string>(ALL);
  const [current, setCurrent] = useState(1);
  const [data, setData] = useState<{ total: number; transactions: WalletTransaction[] } | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    if (!user) return;
    setLoading(true);
    try {
      setData(
        await WalletService.listTransactions({
          current,
          size: PAGE_SIZE,
          currency: currency === ALL ? undefined : currency,
        }),
      );
    } catch (err) {
      toast.error(toErrorMessage(err));
    } finally {
      setLoading(false);
    }
  }, [current, currency, user]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 筛选条件变化时重新拉取
    void load();
  }, [load]);

  // 流水里可能有已停用且余额为 0 的货币, 余额列表里没有时直接显示代码
  const currencyOf = (code: string) => balances.find((b) => b.currency === code);

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between gap-2">
        <h2 className="text-lg font-semibold">余额流水</h2>
        <Select
          value={currency}
          onValueChange={(value) => {
            setCurrent(1);
            setCurrency(value);
          }}
        >
          <SelectTrigger className="w-32">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL}>全部货币</SelectItem>
            {balances.map((b) => (
              <SelectItem key={b.currency} value={b.currency}>
                {b.name}
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
              <TableHead>货币</TableHead>
              <TableHead className="text-right">变动</TableHead>
              <TableHead className="text-right">变动后余额</TableHead>
              <TableHead>类型</TableHead>
              <TableHead>备注</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading && !data ? (
              <TableRow>
                <TableCell colSpan={6} className="h-24 text-center">
                  <LoaderCircle className="mx-auto size-5 animate-spin text-primary" />
                </TableCell>
              </TableRow>
            ) : data?.transactions.length ? (
              data.transactions.map((t) => {
                const c = currencyOf(t.currency);
                const decimals = c?.decimals ?? 0;
                return (
                  <TableRow key={t.id}>
                    <TableCell className="text-muted-foreground">{formatDateTime(t.created_at)}</TableCell>
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
                    <TableCell className="max-w-64 truncate text-muted-foreground" title={t.remark || undefined}>
                      {t.remark || "-"}
                    </TableCell>
                  </TableRow>
                );
              })
            ) : (
              <TableRow>
                <TableCell colSpan={6} className="h-24 text-center text-muted-foreground">
                  还没有流水
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
