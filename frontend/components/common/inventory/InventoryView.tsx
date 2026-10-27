"use client";

import { useCallback, useEffect, useState } from "react";
import { Backpack, LoaderCircle, ScrollText, X } from "lucide-react";
import { toast } from "sonner";
import { cn } from "cn";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useAuth } from "@/components/common/auth/AuthProvider";
import { ItemImage } from "@/components/common/item/ItemImage";
import { Pagination } from "@/components/common/layout/Pagination";
import { CurrencyIcon } from "@/components/common/wallet/CurrencyIcon";
import { toErrorMessage } from "@/lib/services/core";
import { InventoryService, type InventoryTransaction, type UserItem } from "@/lib/services/inventory";
import { WalletService, type BalanceItem } from "@/lib/services/wallet";
import { INVENTORY_BIZ_LABELS } from "@/lib/biz";
import { formatAmount, formatDateTime } from "@/lib/format";
import { QUALITIES, qualityOf } from "@/lib/quality";

const ITEM_PAGE_SIZE = 24;
const RECORD_PAGE_SIZE = 20;

/** 记录页只看某个物品时的筛选条件 */
type ItemFilter = { id: number; name: string };

/** 背包: 物品网格 + 背包记录 */
export function InventoryView() {
  const { user } = useAuth();
  const [tab, setTab] = useState("items");
  const [itemFilter, setItemFilter] = useState<ItemFilter | null>(null);
  // 货币名称和小数位, 用于展示售价
  const [currencies, setCurrencies] = useState<BalanceItem[]>([]);

  useEffect(() => {
    if (!user) return;
    WalletService.listBalances()
      .then((data) => setCurrencies(data.balances))
      .catch(() => {
        // 只影响售价的货币名称, 失败时显示货币代码
      });
  }, [user]);

  const viewRecords = (item: UserItem) => {
    setItemFilter({ id: item.item_id, name: item.name });
    setTab("records");
  };

  return (
    <div className="flex flex-col gap-4">
      <h1 className="text-2xl font-bold">背包</h1>
      <Tabs value={tab} onValueChange={setTab}>
        <TabsList>
          <TabsTrigger value="items">
            <Backpack />
            物品
          </TabsTrigger>
          <TabsTrigger value="records">
            <ScrollText />
            记录
          </TabsTrigger>
        </TabsList>
        <TabsContent value="items" className="mt-2">
          <ItemGrid currencies={currencies} onViewRecords={viewRecords} />
        </TabsContent>
        <TabsContent value="records" className="mt-2">
          <RecordList itemFilter={itemFilter} onClearFilter={() => setItemFilter(null)} />
        </TabsContent>
      </Tabs>
    </div>
  );
}

function ItemGrid({ currencies, onViewRecords }: { currencies: BalanceItem[]; onViewRecords: (item: UserItem) => void }) {
  const { user } = useAuth();
  const [quality, setQuality] = useState<number | undefined>();
  const [current, setCurrent] = useState(1);
  const [data, setData] = useState<{ total: number; items: UserItem[] } | null>(null);
  const [loading, setLoading] = useState(true);
  const [selected, setSelected] = useState<UserItem | null>(null);

  const load = useCallback(async () => {
    if (!user) return;
    setLoading(true);
    try {
      setData(await InventoryService.listItems({ current, size: ITEM_PAGE_SIZE, quality }));
    } catch (err) {
      toast.error(toErrorMessage(err));
    } finally {
      setLoading(false);
    }
  }, [current, quality, user]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 筛选条件变化时重新拉取
    void load();
  }, [load]);

  const changeQuality = (value: number | undefined) => {
    setCurrent(1);
    setQuality(value);
  };

  return (
    <div className="flex flex-col gap-4">
      <div className="hide-scrollbar flex gap-1 overflow-x-auto">
        <Button variant={quality === undefined ? "secondary" : "ghost"} size="sm" onClick={() => changeQuality(undefined)}>
          全部
        </Button>
        {QUALITIES.map((q) => (
          <Button key={q.value} variant={quality === q.value ? "secondary" : "ghost"} size="sm" onClick={() => changeQuality(q.value)}>
            <span className={q.text}>{q.label}</span>
          </Button>
        ))}
      </div>

      {loading && !data ? (
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4 lg:grid-cols-6">
          {Array.from({ length: 6 }, (_, i) => (
            <div key={i} className="aspect-[3/4] animate-pulse rounded-xl border bg-card" />
          ))}
        </div>
      ) : data?.items.length ? (
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4 lg:grid-cols-6">
          {data.items.map((item) => (
            <ItemCard key={item.item_id} item={item} onClick={() => setSelected(item)} />
          ))}
        </div>
      ) : (
        <div className="flex h-48 flex-col items-center justify-center gap-2 rounded-xl border bg-card text-muted-foreground">
          <Backpack className="size-8" />
          <span className="text-sm">{quality === undefined ? "背包里还没有物品" : "没有这个品质的物品"}</span>
        </div>
      )}

      <Pagination current={current} size={ITEM_PAGE_SIZE} total={data?.total ?? 0} onChange={setCurrent} />

      <ItemDetailDialog
        item={selected}
        currencies={currencies}
        onClose={() => setSelected(null)}
        onViewRecords={(item) => {
          setSelected(null);
          onViewRecords(item);
        }}
      />
    </div>
  );
}

function ItemCard({ item, onClick }: { item: UserItem; onClick: () => void }) {
  const q = qualityOf(item.quality);
  return (
    <button
      type="button"
      onClick={onClick}
      className="flex flex-col overflow-hidden rounded-xl border bg-card text-left transition-colors outline-none hover:bg-accent focus-visible:ring-[3px] focus-visible:ring-ring/50"
    >
      <span className={cn("h-1 shrink-0", q.bar)} />
      <span className="relative flex aspect-square items-center justify-center p-4">
        <ItemImage item={item} className={cn(item.is_placeholder && "opacity-60")} />
        <span className="absolute right-2 bottom-2 rounded-md bg-background/80 px-1.5 py-0.5 font-mono text-xs tabular-nums">
          ×{item.quantity}
        </span>
      </span>
      <span className="flex flex-col gap-0.5 px-3 pb-3">
        <span className={cn("truncate font-medium", q.text)} title={item.name}>
          {item.name}
        </span>
        <span className="text-xs text-muted-foreground">
          {q.label} · {item.width}×{item.height}
        </span>
      </span>
    </button>
  );
}

function ItemDetailDialog({
  item,
  currencies,
  onClose,
  onViewRecords,
}: {
  item: UserItem | null;
  currencies: BalanceItem[];
  onClose: () => void;
  onViewRecords: (item: UserItem) => void;
}) {
  const q = qualityOf(item?.quality ?? 1);
  const currency = currencies.find((c) => c.currency === item?.sell_currency);

  return (
    <Dialog open={!!item} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-md">
        {item && (
          <>
            <DialogHeader className="flex-row items-center gap-4 text-left">
              <span className="flex size-16 shrink-0 items-center justify-center overflow-hidden rounded-lg border bg-card">
                <ItemImage item={item} />
              </span>
              <div className="flex min-w-0 flex-col gap-1">
                <DialogTitle className={cn("truncate", q.text)}>{item.name}</DialogTitle>
                <DialogDescription className="flex flex-wrap items-center gap-2">
                  <span>{q.label}品质</span>
                  {item.is_placeholder ? (
                    <Badge variant="outline">已移除</Badge>
                  ) : (
                    !item.is_enabled && <Badge variant="outline">已下架</Badge>
                  )}
                </DialogDescription>
              </div>
            </DialogHeader>

            {item.description && <p className="text-sm whitespace-pre-wrap text-muted-foreground">{item.description}</p>}
            {item.is_placeholder && <p className="text-sm text-muted-foreground">这个物品已被官方移除, 已有的仍保留在背包中。</p>}

            <dl className="grid grid-cols-2 gap-3 rounded-lg border bg-card p-4 text-sm">
              <Detail label="数量">
                <span className="font-mono tabular-nums">{item.quantity}</span>
              </Detail>
              <Detail label="尺寸">
                {item.width}×{item.height} 格
              </Detail>
              <Detail label="出售价格">
                {item.sell_price > 0 ? (
                  <span className="flex items-center gap-1.5">
                    <CurrencyIcon currency={item.sell_currency} className="size-4" />
                    <span className="font-mono tabular-nums">{formatAmount(item.sell_price, currency?.decimals ?? 0)}</span>
                    <span className="text-muted-foreground">{currency?.name ?? item.sell_currency}</span>
                  </span>
                ) : (
                  <span className="text-muted-foreground">不可出售</span>
                )}
              </Detail>
              <Detail label="首次获得">{formatDateTime(item.created_at)}</Detail>
            </dl>

            <DialogFooter>
              <Button variant="outline" onClick={() => onViewRecords(item)}>
                <ScrollText />
                查看记录
              </Button>
            </DialogFooter>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}

function Detail({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-1">
      <dt className="text-xs text-muted-foreground">{label}</dt>
      <dd>{children}</dd>
    </div>
  );
}

function RecordList({ itemFilter, onClearFilter }: { itemFilter: ItemFilter | null; onClearFilter: () => void }) {
  const { user } = useAuth();
  const [current, setCurrent] = useState(1);
  const [data, setData] = useState<{ total: number; transactions: InventoryTransaction[] } | null>(null);
  const [loading, setLoading] = useState(true);
  const itemID = itemFilter?.id;

  // 切换物品筛选时回到第一页
  const [prevItemID, setPrevItemID] = useState(itemID);
  if (prevItemID !== itemID) {
    setPrevItemID(itemID);
    setCurrent(1);
  }

  const load = useCallback(async () => {
    if (!user) return;
    setLoading(true);
    try {
      setData(await InventoryService.listTransactions({ current, size: RECORD_PAGE_SIZE, item_id: itemID }));
    } catch (err) {
      toast.error(toErrorMessage(err));
    } finally {
      setLoading(false);
    }
  }, [current, itemID, user]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 筛选条件变化时重新拉取
    void load();
  }, [load]);

  return (
    <div className="flex flex-col gap-4">
      {itemFilter && (
        <div className="flex items-center gap-2 text-sm">
          <span className="text-muted-foreground">只看</span>
          <Badge variant="secondary" className="gap-1 pr-1">
            {itemFilter.name}
            <button type="button" aria-label="取消筛选" className="rounded-full p-0.5 hover:bg-background/50" onClick={onClearFilter}>
              <X className="size-3" />
            </button>
          </Badge>
        </div>
      )}

      <div className="rounded-xl border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>时间</TableHead>
              <TableHead>物品</TableHead>
              <TableHead className="text-right">变动</TableHead>
              <TableHead className="text-right">变动后数量</TableHead>
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
              data.transactions.map((t) => (
                <TableRow key={t.id}>
                  <TableCell className="text-muted-foreground">{formatDateTime(t.created_at)}</TableCell>
                  <TableCell>{t.item_name}</TableCell>
                  <TableCell
                    className={cn("text-right font-mono tabular-nums", t.quantity > 0 ? "text-quality-green" : "text-destructive")}
                  >
                    {t.quantity > 0 ? `+${t.quantity}` : t.quantity}
                  </TableCell>
                  <TableCell className="text-right font-mono text-muted-foreground tabular-nums">{t.quantity_after}</TableCell>
                  <TableCell>{INVENTORY_BIZ_LABELS[t.biz_type] ?? t.biz_type}</TableCell>
                  <TableCell className="max-w-64 truncate text-muted-foreground" title={t.remark || undefined}>
                    {t.remark || "-"}
                  </TableCell>
                </TableRow>
              ))
            ) : (
              <TableRow>
                <TableCell colSpan={6} className="h-24 text-center text-muted-foreground">
                  还没有记录
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>

      <Pagination current={current} size={RECORD_PAGE_SIZE} total={data?.total ?? 0} onChange={setCurrent} />
    </div>
  );
}
