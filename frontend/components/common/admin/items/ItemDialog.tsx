'use client';

import {useState} from 'react';
import {toast} from 'sonner';
import {Button} from '@/components/ui/button';
import {Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle} from '@/components/ui/dialog';
import {Input} from '@/components/ui/input';
import {Label} from '@/components/ui/label';
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select';
import {Switch} from '@/components/ui/switch';
import {Textarea} from '@/components/ui/textarea';
import {ImageUploadField, SubmitButton} from '@/components/common/form';
import {ItemImage} from '@/components/common/item';
import {CurrencyIcon} from '@/components/common/wallet';
import {useQualities, useSite} from '@/components/common/site';
import {check, useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';
import type {AdminItem, Currency, ItemCategory, ItemRequest} from '@/lib/services/admin';
import {amountInput, parseAmount} from '@/lib/utils';
import {checkImageURL} from '@/lib/validation';
import {CODE_PATTERN, MAX_SELL_PRICE, MAX_SIZE} from './constants';

/**
 * 物品对话框的Props接口
 */
interface ItemDialogProps {
  /** null: 关闭; "new": 新建; 物品: 编辑 */
  item: AdminItem | 'new' | null;
  /** 可选的回收货币 */
  currencies: Currency[];
  /** 可选的物品类别 */
  categories: ItemCategory[];
  onClose: () => void;
  /** 保存成功后刷新列表 */
  onSaved: () => void;
}

/**
 * 新建或编辑物品
 */
export function ItemDialog({item, ...props}: ItemDialogProps) {
  const editing = item && item !== 'new' ? item : null;
  return (
    <Dialog open={!!item} onOpenChange={(open) => !open && props.onClose()}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{editing ? `编辑 ${editing.name}` : '新建物品'}</DialogTitle>
          <DialogDescription>
            {editing ? '修改名称不影响已有记录中的名称' : '上架后可以发放给玩家, 加入掉落池后参与掉落'}
          </DialogDescription>
        </DialogHeader>
        {/* key 保证切换物品时表单重置 */}
        {item && <ItemForm key={editing?.id ?? 'new'} item={editing} {...props} />}
      </DialogContent>
    </Dialog>
  );
}

/**
 * 物品表单, item 为 null 时新建
 */
function ItemForm({item, currencies, categories, onClose, onSaved}: Omit<ItemDialogProps, 'item'> & {
  item: AdminItem | null;
}) {
  const {pending, submit} = useSubmit();
  const {qualities} = useQualities();
  const {economy} = useSite();
  const [code, setCode] = useState(item?.code ?? '');
  const [name, setName] = useState(item?.name ?? '');
  const [description, setDescription] = useState(item?.description ?? '');
  const [quality, setQuality] = useState(String(item?.quality ?? qualities[0]?.level ?? 1));
  const [category, setCategory] = useState(item?.category ?? categories[0]?.code ?? 'collectible');
  const [imageURL, setImageURL] = useState(item?.image_url ?? '');
  const [sellCurrency, setSellCurrency] = useState(item?.sell_currency ?? economy.base_currency);
  const currency = currencies.find((c) => c.code === sellCurrency);
  const decimals = currency?.decimals ?? 0;
  const [sellPrice, setSellPrice] = useState(item ? amountInput(item.sell_price, decimals) : '0');
  const [width, setWidth] = useState(String(item?.width ?? 1));
  const [height, setHeight] = useState(String(item?.height ?? 1));
  const [enabled, setEnabled] = useState(item?.is_enabled ?? true);

  const handleSubmit = submit(async () => {
    const body: ItemRequest = {
      code: code.trim(),
      name: name.trim(),
      description: description.trim(),
      quality: Number(quality),
      category,
      image_url: imageURL.trim(),
      sell_currency: sellCurrency,
      sell_price: parseAmount(sellPrice, decimals) ?? -1,
      width: parseAmount(width) ?? 0,
      height: parseAmount(height) ?? 0,
      is_enabled: enabled,
    };
    check(CODE_PATTERN.test(body.code) ? null : '物品代码需为 1-64 个小写字母、数字或下划线');
    check(body.name.length === 0 || [...body.name].length > 64 ? '物品名称需为 1-64 个字符' : null);
    check([...body.description].length > 255 ? '描述最多 255 个字符' : null);
    check(checkImageURL(body.image_url));
    check(body.sell_price < 0 || body.sell_price > MAX_SELL_PRICE ? '回收价格式不正确' : null);
    check(
      body.width < 1 || body.width > MAX_SIZE || body.height < 1 || body.height > MAX_SIZE ?
        `宽和高需为 1-${MAX_SIZE} 的整数` :
        null,
    );
    if (item) {
      await services.admin.updateItem(item.id, body);
      toast.success(`已保存 ${body.name}`);
    } else {
      await services.admin.createItem(body);
      toast.success(`已新建 ${body.name}`);
    }
    onSaved();
    onClose();
  });

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4" noValidate>
      <div className="grid grid-cols-2 gap-3">
        <div className="flex flex-col gap-2">
          <Label htmlFor="item-code">代码</Label>
          <Input
            id="item-code"
            maxLength={64}
            placeholder="如 bronze_mirror"
            className="font-mono"
            value={code}
            onChange={(e) => setCode(e.target.value)}
            disabled={pending}
          />
        </div>
        <div className="flex flex-col gap-2">
          <Label htmlFor="item-name">名称</Label>
          <Input id="item-name" maxLength={64} value={name} onChange={(e) => setName(e.target.value)} disabled={pending} />
        </div>
      </div>

      <div className="grid grid-cols-2 gap-3">
        <div className="flex flex-col gap-2">
          <Label>品质</Label>
          <Select value={quality} onValueChange={setQuality} disabled={pending}>
            <SelectTrigger className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {qualities.map((q) => (
                <SelectItem key={q.level} value={String(q.level)}>
                  <span className={q.text}>{q.name}</span>
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="flex flex-col gap-2">
          <Label>类别</Label>
          <Select value={category} onValueChange={setCategory} disabled={pending}>
            <SelectTrigger className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {categories.map((c) => (
                <SelectItem key={c.code} value={c.code}>
                  {c.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>
      <span className="-mt-2 text-xs text-muted-foreground">仓库只抽藏品, 其他类别的物品通过奖励、商城等获得</span>

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

      <ImageUploadField
        id="item-image"
        label="图片地址"
        purpose="item"
        placeholder="选填, 留空显示默认图标"
        value={imageURL}
        onChange={setImageURL}
        preview={
          <span className="flex size-9 shrink-0 items-center justify-center overflow-hidden rounded-md border bg-background">
            <ItemImage item={{name: name || '预览', image_url: imageURL.trim()}} />
          </span>
        }
        disabled={pending}
      />

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
            inputMode={decimals > 0 ? 'decimal' : 'numeric'}
            placeholder="0 表示不可出售"
            value={sellPrice}
            onChange={(e) => setSellPrice(e.target.value)}
            disabled={pending}
          />
        </div>
      </div>

      <div className="grid grid-cols-2 gap-3">
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
        <SubmitButton pending={pending}>{item ? '保存' : '新建'}</SubmitButton>
      </DialogFooter>
    </form>
  );
}
