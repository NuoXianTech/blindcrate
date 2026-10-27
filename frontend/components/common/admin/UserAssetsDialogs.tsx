"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { LoaderCircle, Minus, Plus, ScrollText, Snowflake } from "lucide-react";
import { toast } from "sonner";
import { cn } from "cn";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { SubmitButton } from "@/components/common/form/SubmitButton";
import { check, useSubmit } from "@/components/common/form/use-submit";
import { ItemImage } from "@/components/common/item/ItemImage";
import { CurrencyIcon } from "@/components/common/wallet/CurrencyIcon";
import { AdminService, type AdminItem, type AdminUser } from "@/lib/services/admin";
import { toErrorMessage } from "@/lib/services/core";
import type { UserItem } from "@/lib/services/inventory";
import type { BalanceItem } from "@/lib/services/wallet";
import { ROUTES } from "@/lib/access";
import { formatAmount, parseAmount } from "@/lib/format";
import { qualityOf } from "@/lib/quality";

/** 与后端 wallet.MaxAmount、inventory.MaxQuantity 一致 */
const MAX_AMOUNT = 1_000_000_000_000;
const MAX_QUANTITY = 1_000_000;
/** 一次拉取的物品种类上限, 与后端分页上限一致 */
const ITEM_LIMIT = 100;

type Direction = "add" | "sub";

/** 幂等键, 每次打开表单和每次成功后生成新的; randomUUID 只在 https 和 localhost 可用, 局域网 http 访问也要能用 */
function newIdempotencyKey() {
  return Array.from(crypto.getRandomValues(new Uint8Array(16)), (b) => b.toString(16).padStart(2, "0")).join("");
}

/** 流水查询页, 带上用户筛选 */
function transactionsHref(tab: "wallet" | "inventory", userID: number) {
  return `${ROUTES.adminTransactions}?tab=${tab}&user_id=${userID}`;
}

/** 加款/扣款、发放/扣除的切换 */
function DirectionToggle({
  value,
  onChange,
  labels,
  disabled,
}: {
  value: Direction;
  onChange: (value: Direction) => void;
  labels: [string, string];
  disabled: boolean;
}) {
  return (
    <div className="grid grid-cols-2 gap-1 rounded-lg bg-muted p-1">
      {(["add", "sub"] as const).map((d, i) => (
        <Button
          key={d}
          type="button"
          size="sm"
          variant={value === d ? "secondary" : "ghost"}
          className={cn(value === d && "bg-background shadow-xs hover:bg-background", value === d && d === "sub" && "text-destructive")}
          onClick={() => onChange(d)}
          disabled={disabled}
        >
          {d === "add" ? <Plus /> : <Minus />}
          {labels[i]}
        </Button>
      ))}
    </div>
  );
}

function UserDialog({
  user,
  title,
  onClose,
  children,
}: {
  user: AdminUser | null;
  title: string;
  onClose: () => void;
  children: React.ReactNode;
}) {
  return (
    <Dialog open={!!user} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>
            {user?.nickname} 的{title}
          </DialogTitle>
          <DialogDescription>
            @{user?.username} #{user?.id} · 调整会写入流水, 用户可以看到备注
          </DialogDescription>
        </DialogHeader>
        {children}
      </DialogContent>
    </Dialog>
  );
}

export function UserWalletDialog({ user, onClose }: { user: AdminUser | null; onClose: () => void }) {
  return (
    <UserDialog user={user} title="钱包" onClose={onClose}>
      {/* key 保证切换用户时表单重置 */}
      {user && <WalletPanel key={user.id} user={user} />}
    </UserDialog>
  );
}

function WalletPanel({ user }: { user: AdminUser }) {
  const { pending, submit } = useSubmit();
  const [balances, setBalances] = useState<BalanceItem[] | null>(null);
  const [direction, setDirection] = useState<Direction>("add");
  const [currency, setCurrency] = useState("");
  const [amount, setAmount] = useState("");
  const [remark, setRemark] = useState("");
  const [key, setKey] = useState(newIdempotencyKey);

  const load = useCallback(async () => {
    try {
      const data = await AdminService.listUserBalances(user.id);
      setBalances(data.balances);
      // 默认选第一个可用货币
      setCurrency((prev) => prev || (data.balances.find((b) => b.is_enabled)?.currency ?? ""));
    } catch (err) {
      toast.error(toErrorMessage(err));
    }
  }, [user.id]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 打开时拉取余额
    void load();
  }, [load]);

  const selected = balances?.find((b) => b.currency === currency);
  const decimals = selected?.decimals ?? 0;
  const parsed = parseAmount(amount, decimals);
  const delta = parsed && parsed > 0 ? (direction === "add" ? parsed : -parsed) : null;
  const after = selected && delta !== null ? selected.balance + delta : null;

  const handleSubmit = submit(async () => {
    check(selected ? null : "请选择货币");
    check(parsed !== null && parsed > 0 && parsed <= MAX_AMOUNT ? null : "金额格式不正确");
    check(after !== null && after < 0 ? "扣款后余额不能为负" : null);
    check(remark.trim() ? null : "请填写备注");
    await AdminService.adjustUserBalance(user.id, {
      currency,
      amount: delta!,
      remark: remark.trim(),
      idempotency_key: key,
    });
    toast.success(
      `已给 ${user.username} ${direction === "add" ? "加款" : "扣款"} ${formatAmount(parsed!, decimals)} ${selected!.name}`,
    );
    setAmount("");
    setRemark("");
    setKey(newIdempotencyKey());
    await load();
  });

  return (
    <div className="flex flex-col gap-4">
      {balances ? (
        <div className="grid grid-cols-2 gap-2">
          {balances.map((b) => (
            <div key={b.currency} className={cn("flex flex-col gap-1 rounded-lg border p-3", !b.is_enabled && "opacity-60")}>
              <span className="flex items-center gap-1.5 text-xs text-muted-foreground">
                <CurrencyIcon currency={b.currency} className="size-3.5" />
                {b.name}
                {!b.is_enabled && (
                  <Badge variant="outline" className="px-1 py-0 text-[10px]">
                    已停用
                  </Badge>
                )}
              </span>
              <span className="font-mono text-lg font-semibold tabular-nums">{formatAmount(b.balance, b.decimals)}</span>
              {b.frozen > 0 && (
                <span className="flex items-center gap-1 text-xs text-muted-foreground">
                  <Snowflake className="size-3" />
                  冻结 {formatAmount(b.frozen, b.decimals)}
                </span>
              )}
            </div>
          ))}
        </div>
      ) : (
        <div className="flex h-20 items-center justify-center rounded-lg border">
          <LoaderCircle className="size-5 animate-spin text-primary" />
        </div>
      )}

      <form onSubmit={handleSubmit} className="flex flex-col gap-4" noValidate>
        <DirectionToggle value={direction} onChange={setDirection} labels={["加款", "扣款"]} disabled={pending} />
        <div className="grid grid-cols-[8rem_1fr] gap-3">
          <div className="flex flex-col gap-2">
            <Label>货币</Label>
            <Select value={currency} onValueChange={setCurrency} disabled={pending}>
              <SelectTrigger className="w-full">
                <SelectValue placeholder="选择货币" />
              </SelectTrigger>
              <SelectContent>
                {/* 已停用的货币不能调账 */}
                {balances
                  ?.filter((b) => b.is_enabled)
                  .map((b) => (
                    <SelectItem key={b.currency} value={b.currency}>
                      <CurrencyIcon currency={b.currency} />
                      {b.name}
                    </SelectItem>
                  ))}
              </SelectContent>
            </Select>
          </div>
          <div className="flex flex-col gap-2">
            <Label htmlFor="adjust-amount">金额</Label>
            <Input
              id="adjust-amount"
              inputMode={decimals > 0 ? "decimal" : "numeric"}
              placeholder="大于 0"
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
              disabled={pending}
            />
          </div>
        </div>
        <div className="flex flex-col gap-2">
          <Label htmlFor="adjust-remark">备注</Label>
          <Input
            id="adjust-remark"
            maxLength={255}
            placeholder="调整原因, 用户可以看到"
            value={remark}
            onChange={(e) => setRemark(e.target.value)}
            disabled={pending}
          />
        </div>
        <DialogFooter className="items-center">
          <span className={cn("mr-auto text-sm text-muted-foreground", after !== null && after < 0 && "text-destructive")}>
            {after !== null && `调整后 ${formatAmount(after, decimals)} ${selected!.name}`}
          </span>
          <Button type="button" variant="outline" asChild>
            <Link href={transactionsHref("wallet", user.id)}>
              <ScrollText />
              查看流水
            </Link>
          </Button>
          <SubmitButton pending={pending} variant={direction === "sub" ? "destructive" : "default"}>
            {direction === "add" ? "加款" : "扣款"}
          </SubmitButton>
        </DialogFooter>
      </form>
    </div>
  );
}

export function UserInventoryDialog({ user, onClose }: { user: AdminUser | null; onClose: () => void }) {
  return (
    <UserDialog user={user} title="背包" onClose={onClose}>
      {user && <InventoryPanel key={user.id} user={user} />}
    </UserDialog>
  );
}

function InventoryPanel({ user }: { user: AdminUser }) {
  const { pending, submit } = useSubmit();
  const [owned, setOwned] = useState<{ total: number; items: UserItem[] } | null>(null);
  // 可发放的物品(已上架)
  const [grantable, setGrantable] = useState<AdminItem[]>([]);
  const [direction, setDirection] = useState<Direction>("add");
  const [itemID, setItemID] = useState("");
  const [quantity, setQuantity] = useState("1");
  const [remark, setRemark] = useState("");
  const [key, setKey] = useState(newIdempotencyKey);

  const load = useCallback(async () => {
    try {
      setOwned(await AdminService.listUserInventory(user.id, { current: 1, size: ITEM_LIMIT }));
    } catch (err) {
      toast.error(toErrorMessage(err));
    }
  }, [user.id]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 打开时拉取背包
    void load();
  }, [load]);

  useEffect(() => {
    AdminService.listItems({ current: 1, size: ITEM_LIMIT, is_enabled: true })
      .then((data) => setGrantable(data.items))
      .catch((err) => toast.error(toErrorMessage(err)));
  }, []);

  // 发放只能选已上架的物品; 扣除选用户已有的, 包括已下架和已删除的
  const options =
    direction === "add"
      ? grantable.map((i) => ({ id: i.id, name: i.name, quality: i.quality }))
      : (owned?.items ?? []).map((i) => ({ id: i.item_id, name: i.name, quality: i.quality }));
  const ownedQuantity = owned?.items.find((i) => String(i.item_id) === itemID)?.quantity ?? 0;
  const parsed = parseAmount(quantity);
  const delta = parsed && parsed > 0 ? (direction === "add" ? parsed : -parsed) : null;
  const after = itemID && delta !== null ? ownedQuantity + delta : null;
  const selectedName = options.find((o) => String(o.id) === itemID)?.name;

  const changeDirection = (value: Direction) => {
    setDirection(value);
    setItemID("");
  };

  // 点背包里的物品, 直接选中扣除
  const pickDeduct = (item: UserItem) => {
    setDirection("sub");
    setItemID(String(item.item_id));
  };

  const handleSubmit = submit(async () => {
    check(itemID ? null : "请选择物品");
    check(parsed !== null && parsed > 0 && parsed <= MAX_QUANTITY ? null : `数量需为 1-${MAX_QUANTITY} 的整数`);
    check(after !== null && after < 0 ? "扣除后数量不能为负" : null);
    check(remark.trim() ? null : "请填写备注");
    await AdminService.adjustUserInventory(user.id, {
      item_id: Number(itemID),
      quantity: delta!,
      remark: remark.trim(),
      idempotency_key: key,
    });
    toast.success(`已给 ${user.username} ${direction === "add" ? "发放" : "扣除"} ${selectedName} ×${parsed}`);
    setQuantity("1");
    setRemark("");
    setKey(newIdempotencyKey());
    // 扣完了物品会从列表消失, 不能再选
    if (after === 0) setItemID("");
    await load();
  });

  return (
    <div className="flex flex-col gap-4">
      {owned ? (
        owned.items.length ? (
          <div className="flex max-h-56 flex-col overflow-y-auto rounded-lg border">
            {owned.items.map((i) => {
              const q = qualityOf(i.quality);
              return (
                <button
                  key={i.item_id}
                  type="button"
                  title="选中扣除"
                  onClick={() => pickDeduct(i)}
                  className={cn(
                    "flex items-center gap-3 border-b px-3 py-2 text-left text-sm last:border-b-0 hover:bg-accent",
                    direction === "sub" && String(i.item_id) === itemID && "bg-accent",
                  )}
                >
                  <span className="relative flex size-8 shrink-0 items-center justify-center overflow-hidden rounded-md border bg-background">
                    <ItemImage item={i} className={cn(i.is_placeholder && "opacity-60")} />
                    <span className={cn("absolute inset-x-0 top-0 h-0.5", q.bar)} />
                  </span>
                  <span className={cn("min-w-0 flex-1 truncate font-medium", q.text)}>{i.name}</span>
                  {i.is_placeholder ? (
                    <Badge variant="outline">已移除</Badge>
                  ) : (
                    !i.is_enabled && <Badge variant="outline">已下架</Badge>
                  )}
                  <span className="font-mono text-muted-foreground tabular-nums">×{i.quantity}</span>
                </button>
              );
            })}
            {owned.total > owned.items.length && (
              <span className="px-3 py-2 text-xs text-muted-foreground">只显示前 {owned.items.length} 种, 共 {owned.total} 种</span>
            )}
          </div>
        ) : (
          <div className="flex h-20 items-center justify-center rounded-lg border text-sm text-muted-foreground">背包是空的</div>
        )
      ) : (
        <div className="flex h-20 items-center justify-center rounded-lg border">
          <LoaderCircle className="size-5 animate-spin text-primary" />
        </div>
      )}

      <form onSubmit={handleSubmit} className="flex flex-col gap-4" noValidate>
        <DirectionToggle value={direction} onChange={changeDirection} labels={["发放", "扣除"]} disabled={pending} />
        <div className="grid grid-cols-[1fr_7rem] gap-3">
          <div className="flex min-w-0 flex-col gap-2">
            <Label>物品</Label>
            <Select value={itemID} onValueChange={setItemID} disabled={pending}>
              <SelectTrigger className="w-full">
                <SelectValue placeholder={direction === "add" ? "选择已上架的物品" : "选择背包中的物品"} />
              </SelectTrigger>
              <SelectContent>
                {options.map((o) => (
                  <SelectItem key={o.id} value={String(o.id)}>
                    <span className={qualityOf(o.quality).text}>{o.name}</span>
                    <span className="text-xs text-muted-foreground">#{o.id}</span>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="flex flex-col gap-2">
            <Label htmlFor="adjust-quantity">数量</Label>
            <Input
              id="adjust-quantity"
              inputMode="numeric"
              value={quantity}
              onChange={(e) => setQuantity(e.target.value)}
              disabled={pending}
            />
          </div>
        </div>
        <div className="flex flex-col gap-2">
          <Label htmlFor="adjust-item-remark">备注</Label>
          <Input
            id="adjust-item-remark"
            maxLength={255}
            placeholder="调整原因, 用户可以看到"
            value={remark}
            onChange={(e) => setRemark(e.target.value)}
            disabled={pending}
          />
        </div>
        <DialogFooter className="items-center">
          <span className={cn("mr-auto text-sm text-muted-foreground", after !== null && after < 0 && "text-destructive")}>
            {after !== null && `调整后 ×${after}`}
          </span>
          <Button type="button" variant="outline" asChild>
            <Link href={transactionsHref("inventory", user.id)}>
              <ScrollText />
              查看记录
            </Link>
          </Button>
          <SubmitButton pending={pending} variant={direction === "sub" ? "destructive" : "default"}>
            {direction === "add" ? "发放" : "扣除"}
          </SubmitButton>
        </DialogFooter>
      </form>
    </div>
  );
}
