"use client";

import { useCallback, useEffect, useState } from "react";
import { LoaderCircle, Pencil, Plus, Power, Search, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { cn } from "cn";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Textarea } from "@/components/ui/textarea";
import { useAuth } from "@/components/common/auth/AuthProvider";
import { SubmitButton } from "@/components/common/form/SubmitButton";
import { check, useSubmit } from "@/components/common/form/use-submit";
import { ItemImage } from "@/components/common/item/ItemImage";
import { Pagination } from "@/components/common/layout/Pagination";
import { CurrencyIcon } from "@/components/common/wallet/CurrencyIcon";
import { AdminService, type AdminItem, type Currency, type ItemRequest } from "@/lib/services/admin";
import { toErrorMessage } from "@/lib/services/core";
import { amountInput, formatAmount, formatDateTime, parseAmount } from "@/lib/format";
import { QUALITIES, qualityOf } from "@/lib/quality";

const PAGE_SIZE = 20;

type StatusFilter = "all" | "enabled" | "disabled";

const STATUS_FILTERS: { value: StatusFilter; label: string }[] = [
  { value: "all", label: "全部状态" },
  { value: "enabled", label: "上架中" },
  { value: "disabled", label: "已下架" },
];

/** 与后端 ItemRequest 的校验一致 */
const MAX_SELL_PRICE = 1_000_000_000_000;
const MAX_WEIGHT = 1_000_000;
const MAX_SIZE = 10;

function toRequest(item: AdminItem): ItemRequest {
  const { name, description, quality, image_url, sell_currency, sell_price, weight, width, height, is_enabled } = item;
  return { name, description, quality, image_url, sell_currency, sell_price, weight, width, height, is_enabled };
}

export function ItemsManager() {
  const { user: me } = useAuth();
  const isAdmin = !!me?.is_admin;
  const [keywordInput, setKeywordInput] = useState("");
  const [keyword, setKeyword] = useState("");
  const [quality, setQuality] = useState("all");
  const [status, setStatus] = useState<StatusFilter>("all");
  const [current, setCurrent] = useState(1);
  const [data, setData] = useState<{ total: number; items: AdminItem[] } | null>(null);
  const [loading, setLoading] = useState(true);
  const [currencies, setCurrencies] = useState<Currency[]>([]);

  // null: 关闭; "new": 新建; 物品: 编辑
  const [editing, setEditing] = useState<AdminItem | "new" | null>(null);
  const [deleting, setDeleting] = useState<AdminItem | null>(null);

  useEffect(() => {
    if (!isAdmin) return;
    AdminService.listCurrencies()
      .then((data) => setCurrencies(data.currencies))
      .catch((err) => toast.error(toErrorMessage(err)));
  }, [isAdmin]);

  const load = useCallback(async () => {
    // 确认是管理员后再拉取, 否则 AdminGuard 会跳走
    if (!isAdmin) return;
    setLoading(true);
    try {
      setData(
        await AdminService.listItems({
          current,
          size: PAGE_SIZE,
          keyword: keyword || undefined,
          quality: quality === "all" ? undefined : Number(quality),
          is_enabled: status === "all" ? undefined : status === "enabled",
        }),
      );
    } catch (err) {
      toast.error(toErrorMessage(err));
    } finally {
      setLoading(false);
    }
  }, [current, keyword, quality, status, isAdmin]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 筛选条件变化时重新拉取
    void load();
  }, [load]);

  const handleSearch = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setCurrent(1);
    setKeyword(keywordInput.trim());
  };

  // 上下架也走修改接口, 需提交全部字段
  const toggleEnabled = async (item: AdminItem) => {
    try {
      await AdminService.updateItem(item.id, { ...toRequest(item), is_enabled: !item.is_enabled });
      toast.success(item.is_enabled ? `已下架 ${item.name}` : `已上架 ${item.name}`);
      await load();
    } catch (err) {
      toast.error(toErrorMessage(err));
    }
  };

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between gap-2">
        <h1 className="text-2xl font-bold">物品管理</h1>
        <Button onClick={() => setEditing("new")}>
          <Plus />
          新建物品
        </Button>
      </div>

      <div className="flex flex-col gap-2 sm:flex-row">
        <form onSubmit={handleSearch} className="flex flex-1 gap-2">
          <Input
            placeholder="物品名称"
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
          value={quality}
          onValueChange={(value) => {
            setCurrent(1);
            setQuality(value);
          }}
        >
          <SelectTrigger className="sm:w-32">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部品质</SelectItem>
            {QUALITIES.map((q) => (
              <SelectItem key={q.value} value={String(q.value)}>
                <span className={q.text}>{q.label}</span>
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select
          value={status}
          onValueChange={(value) => {
            setCurrent(1);
            setStatus(value as StatusFilter);
          }}
        >
          <SelectTrigger className="sm:w-32">
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
              <TableHead>物品</TableHead>
              <TableHead>品质</TableHead>
              <TableHead className="text-right">回收价</TableHead>
              <TableHead className="text-right">掉落权重</TableHead>
              <TableHead>尺寸</TableHead>
              <TableHead className="text-right">持有人数</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>更新时间</TableHead>
              <TableHead className="text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading && !data ? (
              <TableRow>
                <TableCell colSpan={9} className="h-24 text-center">
                  <LoaderCircle className="mx-auto size-5 animate-spin text-primary" />
                </TableCell>
              </TableRow>
            ) : data?.items.length ? (
              data.items.map((item) => {
                const q = qualityOf(item.quality);
                const currency = currencies.find((c) => c.code === item.sell_currency);
                return (
                  <TableRow key={item.id}>
                    <TableCell>
                      <div className="flex items-center gap-3">
                        <span className="relative flex size-10 shrink-0 items-center justify-center overflow-hidden rounded-md border bg-background">
                          <ItemImage item={item} />
                          <span className={cn("absolute inset-x-0 top-0 h-0.5", q.bar)} />
                        </span>
                        <div className="flex min-w-0 flex-col">
                          <span className={cn("font-medium", q.text)}>{item.name}</span>
                          <span className="max-w-56 truncate text-xs text-muted-foreground" title={item.description || undefined}>
                            #{item.id}
                            {item.description && ` · ${item.description}`}
                          </span>
                        </div>
                      </div>
                    </TableCell>
                    <TableCell>
                      <span className={q.text}>{q.label}</span>
                    </TableCell>
                    <TableCell className="text-right">
                      {item.sell_price > 0 ? (
                        <span className="inline-flex items-center gap-1.5" title={currency?.name ?? item.sell_currency}>
                          <CurrencyIcon currency={item.sell_currency} className="size-4" />
                          <span className="font-mono tabular-nums">{formatAmount(item.sell_price, currency?.decimals ?? 0)}</span>
                        </span>
                      ) : (
                        <span className="text-muted-foreground">不可出售</span>
                      )}
                    </TableCell>
                    <TableCell className="text-right">
                      {item.weight > 0 ? (
                        <span className="font-mono tabular-nums">{item.weight}</span>
                      ) : (
                        <span className="text-muted-foreground">不掉落</span>
                      )}
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {item.width}×{item.height}
                    </TableCell>
                    <TableCell className="text-right font-mono tabular-nums">{item.holder_count}</TableCell>
                    <TableCell>
                      {item.is_enabled ? <Badge variant="secondary">上架中</Badge> : <Badge variant="outline">已下架</Badge>}
                    </TableCell>
                    <TableCell className="text-muted-foreground">{formatDateTime(item.updated_at)}</TableCell>
                    <TableCell>
                      <div className="flex justify-end gap-1">
                        <Button variant="ghost" size="icon" title="编辑" aria-label="编辑" onClick={() => setEditing(item)}>
                          <Pencil />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          title={item.is_enabled ? "下架" : "上架"}
                          aria-label={item.is_enabled ? "下架" : "上架"}
                          className={item.is_enabled ? undefined : "text-destructive"}
                          onClick={() => toggleEnabled(item)}
                        >
                          <Power />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          title="删除"
                          aria-label="删除"
                          className="hover:text-destructive"
                          onClick={() => setDeleting(item)}
                        >
                          <Trash2 />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                );
              })
            ) : (
              <TableRow>
                <TableCell colSpan={9} className="h-24 text-center text-muted-foreground">
                  没有找到物品
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>

      <Pagination current={current} size={PAGE_SIZE} total={data?.total ?? 0} onChange={setCurrent} />

      <ItemDialog item={editing} currencies={currencies} onClose={() => setEditing(null)} onSaved={load} />
      <DeleteItemDialog item={deleting} onClose={() => setDeleting(null)} onSaved={load} />
    </div>
  );
}

function ItemDialog({
  item,
  currencies,
  onClose,
  onSaved,
}: {
  item: AdminItem | "new" | null;
  currencies: Currency[];
  onClose: () => void;
  onSaved: () => void;
}) {
  const editing = item && item !== "new" ? item : null;
  return (
    <Dialog open={!!item} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{editing ? `编辑 ${editing.name}` : "新建物品"}</DialogTitle>
          <DialogDescription>
            {editing ? "修改名称不影响已有记录中的名称" : "上架后可以发放给玩家, 掉落权重大于 0 时参与掉落"}
          </DialogDescription>
        </DialogHeader>
        {/* key 保证切换物品时表单重置 */}
        {item && <ItemForm key={editing?.id ?? "new"} item={editing} currencies={currencies} onClose={onClose} onSaved={onSaved} />}
      </DialogContent>
    </Dialog>
  );
}

function ItemForm({
  item,
  currencies,
  onClose,
  onSaved,
}: {
  item: AdminItem | null;
  currencies: Currency[];
  onClose: () => void;
  onSaved: () => void;
}) {
  const { pending, submit } = useSubmit();
  const [name, setName] = useState(item?.name ?? "");
  const [description, setDescription] = useState(item?.description ?? "");
  const [quality, setQuality] = useState(String(item?.quality ?? 1));
  const [imageURL, setImageURL] = useState(item?.image_url ?? "");
  const [sellCurrency, setSellCurrency] = useState(item?.sell_currency ?? currencies[0]?.code ?? "chip");
  const currency = currencies.find((c) => c.code === sellCurrency);
  const decimals = currency?.decimals ?? 0;
  const [sellPrice, setSellPrice] = useState(item ? amountInput(item.sell_price, decimals) : "0");
  const [weight, setWeight] = useState(String(item?.weight ?? 0));
  const [width, setWidth] = useState(String(item?.width ?? 1));
  const [height, setHeight] = useState(String(item?.height ?? 1));
  const [enabled, setEnabled] = useState(item?.is_enabled ?? true);

  const handleSubmit = submit(async () => {
    const body: ItemRequest = {
      name: name.trim(),
      description: description.trim(),
      quality: Number(quality),
      image_url: imageURL.trim(),
      sell_currency: sellCurrency,
      sell_price: parseAmount(sellPrice, decimals) ?? -1,
      weight: parseAmount(weight) ?? -1,
      width: parseAmount(width) ?? 0,
      height: parseAmount(height) ?? 0,
      is_enabled: enabled,
    };
    check(body.name.length === 0 || [...body.name].length > 64 ? "物品名称需为 1-64 个字符" : null);
    check([...body.description].length > 255 ? "描述最多 255 个字符" : null);
    check(body.image_url && !/^https?:\/\/\S+$/.test(body.image_url) ? "图片地址需以 http:// 或 https:// 开头" : null);
    check(body.sell_price < 0 || body.sell_price > MAX_SELL_PRICE ? "回收价格式不正确" : null);
    check(body.weight < 0 || body.weight > MAX_WEIGHT ? `掉落权重需为 0-${MAX_WEIGHT} 的整数` : null);
    check(
      body.width < 1 || body.width > MAX_SIZE || body.height < 1 || body.height > MAX_SIZE
        ? `宽和高需为 1-${MAX_SIZE} 的整数`
        : null,
    );
    if (item) {
      await AdminService.updateItem(item.id, body);
      toast.success(`已保存 ${body.name}`);
    } else {
      await AdminService.createItem(body);
      toast.success(`已新建 ${body.name}`);
    }
    onSaved();
    onClose();
  });

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4" noValidate>
      <div className="grid grid-cols-[1fr_8rem] gap-3">
        <div className="flex flex-col gap-2">
          <Label htmlFor="item-name">名称</Label>
          <Input id="item-name" maxLength={64} value={name} onChange={(e) => setName(e.target.value)} disabled={pending} />
        </div>
        <div className="flex flex-col gap-2">
          <Label>品质</Label>
          <Select value={quality} onValueChange={setQuality} disabled={pending}>
            <SelectTrigger className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {QUALITIES.map((q) => (
                <SelectItem key={q.value} value={String(q.value)}>
                  <span className={q.text}>{q.label}</span>
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      <div className="flex flex-col gap-2">
        <Label htmlFor="item-description">描述</Label>
        <Textarea
          id="item-description"
          maxLength={255}
          placeholder="选填, 最多 255 个字符"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          disabled={pending}
        />
      </div>

      <div className="flex flex-col gap-2">
        <Label htmlFor="item-image">图片地址</Label>
        <div className="flex gap-3">
          <span className="flex size-9 shrink-0 items-center justify-center overflow-hidden rounded-md border bg-background">
            <ItemImage item={{ name: name || "预览", image_url: imageURL.trim() }} />
          </span>
          <Input
            id="item-image"
            type="url"
            placeholder="选填, 留空显示默认图标"
            value={imageURL}
            onChange={(e) => setImageURL(e.target.value)}
            disabled={pending}
          />
        </div>
      </div>

      <div className="grid grid-cols-[8rem_1fr] gap-3">
        <div className="flex flex-col gap-2">
          <Label>回收货币</Label>
          <Select value={sellCurrency} onValueChange={setSellCurrency} disabled={pending}>
            <SelectTrigger className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {currencies.map((c) => (
                <SelectItem key={c.code} value={c.code}>
                  <CurrencyIcon currency={c.code} />
                  {c.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="flex flex-col gap-2">
          <Label htmlFor="item-price">回收价</Label>
          <Input
            id="item-price"
            inputMode={decimals > 0 ? "decimal" : "numeric"}
            placeholder="0 表示不可出售"
            value={sellPrice}
            onChange={(e) => setSellPrice(e.target.value)}
            disabled={pending}
          />
        </div>
      </div>

      <div className="grid grid-cols-3 gap-3">
        <div className="flex flex-col gap-2">
          <Label htmlFor="item-weight">掉落权重</Label>
          <Input
            id="item-weight"
            inputMode="numeric"
            placeholder="0 表示不掉落"
            value={weight}
            onChange={(e) => setWeight(e.target.value)}
            disabled={pending}
          />
        </div>
        <div className="flex flex-col gap-2">
          <Label htmlFor="item-width">宽(格)</Label>
          <Input id="item-width" inputMode="numeric" value={width} onChange={(e) => setWidth(e.target.value)} disabled={pending} />
        </div>
        <div className="flex flex-col gap-2">
          <Label htmlFor="item-height">高(格)</Label>
          <Input id="item-height" inputMode="numeric" value={height} onChange={(e) => setHeight(e.target.value)} disabled={pending} />
        </div>
      </div>

      <div className="flex items-center justify-between gap-4 rounded-lg border p-3">
        <div className="flex flex-col gap-0.5">
          <Label htmlFor="item-enabled">上架</Label>
          <span className="text-xs text-muted-foreground">下架后不会掉落, 也不能再发放, 玩家已有的保留</span>
        </div>
        <Switch id="item-enabled" checked={enabled} onCheckedChange={setEnabled} disabled={pending} />
      </div>

      <DialogFooter>
        <Button type="button" variant="outline" onClick={onClose} disabled={pending}>
          取消
        </Button>
        <SubmitButton pending={pending}>{item ? "保存" : "新建"}</SubmitButton>
      </DialogFooter>
    </form>
  );
}

function DeleteItemDialog({ item, onClose, onSaved }: { item: AdminItem | null; onClose: () => void; onSaved: () => void }) {
  const { pending, submit } = useSubmit();

  const handleSubmit = submit(async () => {
    if (!item) return;
    const { holder_count } = await AdminService.deleteItem(item.id);
    toast.success(holder_count > 0 ? `已删除 ${item.name}, ${holder_count} 名玩家的背包中保留为未知物品` : `已删除 ${item.name}`);
    onSaved();
    onClose();
  });

  return (
    <Dialog open={!!item} onOpenChange={(open) => !open && !pending && onClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>删除 {item?.name}</DialogTitle>
          <DialogDescription>
            删除后无法恢复, 物品 ID 不会被复用。
            {item && item.holder_count > 0
              ? ` 当前有 ${item.holder_count} 名玩家持有, 他们背包中的物品会保留并显示为「未知物品 #${item.id}」。`
              : " 当前没有玩家持有。"}
            只是不想再掉落或发放的话, 下架即可。
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit}>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose} disabled={pending}>
              取消
            </Button>
            <SubmitButton pending={pending} variant="destructive">
              删除
            </SubmitButton>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
