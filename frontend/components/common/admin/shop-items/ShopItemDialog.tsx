'use client';

import {useId, useState} from 'react';
import {toast} from 'sonner';
import {Button} from '@/components/ui/button';
import {Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle} from '@/components/ui/dialog';
import {Input} from '@/components/ui/input';
import {Label} from '@/components/ui/label';
import {Skeleton} from '@/components/ui/skeleton';
import {Textarea} from '@/components/ui/textarea';
import {
  BundleEditor,
  ConditionEditor,
  fromBundleDrafts,
  fromConditionDrafts,
  toBundleDrafts,
  toConditionDrafts,
  type ScopeOption,
} from '@/components/common/admin/editors';
import {fromLocalInput, toLocalInput} from '@/components/common/admin/seasons/constants';
import {FieldGroup, InputField, SwitchField} from '@/components/common/admin/venues/Fields';
import {SubmitButton} from '@/components/common/form';
import {ItemImage} from '@/components/common/item';
import {check, useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';
import type {AdminItem, AdminShopItem, ConditionType, Currency, ShopItemRequest} from '@/lib/services/admin';
import {parseAmount} from '@/lib/utils';
import {CODE_PATTERN, MAX_BUNDLE_LINES, MAX_LIMIT, MAX_REQUIREMENTS, MAX_TAB_LENGTH} from './constants';

/** 编辑商品用到的物品、货币、条件类型、统计范围和已有页签 */
export interface ShopItemOptions {
  items: AdminItem[];
  currencies: Currency[];
  types: ConditionType[];
  scopes: ScopeOption[];
  tabs: string[];
  loaded: boolean;
}

/** 对话框的目标: new 新建; edit 编辑; copy 以已有商品为模板新建 */
export type ShopItemTarget = { mode: 'new' } | { mode: 'edit' | 'copy'; item: AdminShopItem };

/**
 * 商品对话框的Props接口
 */
interface ShopItemDialogProps {
  /** null 时关闭 */
  target: ShopItemTarget | null;
  options: ShopItemOptions;
  onClose: () => void;
  /** 保存成功后刷新列表 */
  onSaved: () => void;
}

/**
 * 新建、编辑或复制商品; 修改不影响已有订单
 */
export function ShopItemDialog({target, options, onClose, onSaved}: ShopItemDialogProps) {
  const title = !target ? '' : target.mode === 'edit' ? `编辑 ${target.item.name}` : target.mode === 'copy' ? `复制 ${target.item.name}` : '新建商品';
  return (
    <Dialog open={!!target} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>购买时一次扣除价格并发放商品; 修改价格、商品后只影响之后的订单</DialogDescription>
        </DialogHeader>
        {target && !options.loaded && (
          <div className="flex flex-col gap-2">
            <Skeleton className="h-9" />
            <Skeleton className="h-9" />
            <Skeleton className="h-9" />
          </div>
        )}
        {/* key 保证切换商品时表单重置 */}
        {target && options.loaded && (
          <ShopItemForm
            key={target.mode === 'new' ? 'new' : `${target.mode}:${target.item.id}`}
            target={target}
            options={options}
            onClose={onClose}
            onSaved={onSaved}
          />
        )}
      </DialogContent>
    </Dialog>
  );
}

/** 限购或库存的输入框原文转为数字, 留空为 0 或不限, 格式不对时提示 */
function parseLimit(text: string, label: string): number {
  const value = parseAmount(text.trim() || '0');
  check(value === null || value < 0 || value > MAX_LIMIT ? `${label}需为 0-${MAX_LIMIT} 的整数` : null);
  return value!;
}

/**
 * 商品表单
 */
function ShopItemForm({target, options, onClose, onSaved}: Omit<ShopItemDialogProps, 'target'> & { target: ShopItemTarget }) {
  const tabListID = useId();
  const {pending, submit} = useSubmit();
  // 编辑和复制时以已有商品为初始值, 复制时代码留空
  const source = target.mode === 'new' ? null : target.item;
  const editing = target.mode === 'edit' ? target.item : null;
  const {items, currencies, types, scopes} = options;

  const [code, setCode] = useState(editing?.code ?? '');
  const [name, setName] = useState(source?.name ?? '');
  const [description, setDescription] = useState(source?.description ?? '');
  const [imageURL, setImageURL] = useState(source?.image_url ?? '');
  const [tab, setTab] = useState(source?.tab ?? '');
  const [sortOrder, setSortOrder] = useState(String(source?.sort_order ?? 0));
  const [isEnabled, setIsEnabled] = useState(source?.is_enabled ?? true);
  const [goods, setGoods] = useState(() => toBundleDrafts(source?.goods ?? {}, currencies));
  const [price, setPrice] = useState(() => toBundleDrafts(source?.price ?? {}, currencies));
  const [originalPrice, setOriginalPrice] = useState(() => toBundleDrafts(source?.original_price ?? {}, currencies));
  const [requirements, setRequirements] = useState(() => toConditionDrafts(source?.requirements ?? []));
  const [limitTotal, setLimitTotal] = useState(String(source?.limits.total ?? 0));
  const [limitDaily, setLimitDaily] = useState(String(source?.limits.daily ?? 0));
  const [limitSeason, setLimitSeason] = useState(String(source?.limits.season ?? 0));
  const [stock, setStock] = useState(source?.stock == null ? '' : String(source.stock));
  const [startsAt, setStartsAt] = useState(source?.starts_at ? toLocalInput(source.starts_at) : '');
  const [endsAt, setEndsAt] = useState(source?.ends_at ? toLocalInput(source.ends_at) : '');

  // 没有修改的时间原样提交, 后端按分钟比较; 留空为不限
  const timeOf = (input: string, original: string | null | undefined, label: string) => {
    if (!input) return null;
    if (original && input === toLocalInput(original)) return original;
    const value = fromLocalInput(input);
    check(value ? null : `${label}格式不正确`);
    return value;
  };

  const handleSubmit = submit(async () => {
    const sort = parseAmount(sortOrder.trim());
    check(sort === null || Math.abs(sort) > 1_000_000 ? '排序需为 -1000000 到 1000000 的整数' : null);
    const body: ShopItemRequest = {
      code: editing?.code ?? code.trim(),
      name: name.trim(),
      description: description.trim(),
      image_url: imageURL.trim(),
      tab: tab.trim(),
      sort_order: sort!,
      goods: fromBundleDrafts(goods, currencies, '商品'),
      price: fromBundleDrafts(price, currencies, '价格'),
      original_price: fromBundleDrafts(originalPrice, currencies, '原价'),
      requirements: fromConditionDrafts(requirements, types, '购买条件'),
      limits: {
        total: parseLimit(limitTotal, '每人限购'),
        daily: parseLimit(limitDaily, '每天限购'),
        season: parseLimit(limitSeason, '每赛季限购'),
      },
      stock: stock.trim() ? parseLimit(stock, '库存') : null,
      starts_at: timeOf(startsAt, editing?.starts_at, '上架时间'),
      ends_at: timeOf(endsAt, editing?.ends_at, '下架时间'),
      is_enabled: isEnabled,
    };
    check(CODE_PATTERN.test(body.code) ? null : '代码需为 1-32 个小写字母、数字或下划线');
    check(body.name.length === 0 || [...body.name].length > 64 ? '名称需为 1-64 个字符' : null);
    check([...body.tab].length > MAX_TAB_LENGTH ? `页签最多 ${MAX_TAB_LENGTH} 个字符` : null);
    check(body.image_url && !/^https?:\/\/\S+$/.test(body.image_url) ? '图片地址需以 http:// 或 https:// 开头' : null);
    check(goods.length === 0 ? '商品不能为空' : null);
    check(body.starts_at && body.ends_at && new Date(body.ends_at) <= new Date(body.starts_at) ? '下架时间需晚于上架时间' : null);
    if (editing) {
      await services.admin.updateShopItem(editing.id, body);
      toast.success(`已保存 ${body.name}`);
    } else {
      await services.admin.createShopItem(body);
      toast.success(`已新建 ${body.name}`);
    }
    onSaved();
    onClose();
  });

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-6" noValidate>
      <FieldGroup title="基本信息">
        <div className="grid grid-cols-2 gap-3">
          <InputField
            id="shop-item-code"
            label="代码"
            placeholder="如 potion_pack"
            maxLength={32}
            value={code}
            onChange={setCode}
            help="限购按代码计数, 创建后不能修改"
            disabled={pending || !!editing}
          />
          <InputField id="shop-item-name" label="名称" placeholder="如 药水礼包" maxLength={64} value={name} onChange={setName} disabled={pending} />
        </div>
        <div className="grid grid-cols-2 gap-3">
          <div className="flex min-w-0 flex-col gap-2">
            <Label htmlFor="shop-item-tab">页签</Label>
            <Input
              id="shop-item-tab"
              list={tabListID}
              maxLength={MAX_TAB_LENGTH}
              placeholder="如 道具, 留空归入默认页签"
              value={tab}
              onChange={(e) => setTab(e.target.value)}
              disabled={pending}
            />
            <datalist id={tabListID}>
              {options.tabs.map((t) => (
                <option key={t} value={t} />
              ))}
            </datalist>
          </div>
          <InputField
            id="shop-item-sort"
            label="排序"
            inputMode="numeric"
            value={sortOrder}
            onChange={setSortOrder}
            help="商城中从小到大排列"
            disabled={pending}
          />
        </div>
        <div className="flex flex-col gap-2">
          <Label htmlFor="shop-item-description">描述</Label>
          <Textarea
            id="shop-item-description"
            maxLength={255}
            placeholder="选填, 最多 255 个字符"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            disabled={pending}
          />
        </div>
        <div className="flex flex-col gap-2">
          <Label htmlFor="shop-item-image">图片地址</Label>
          <div className="flex gap-3">
            <span className="flex size-9 shrink-0 items-center justify-center overflow-hidden rounded-md border bg-background">
              <ItemImage item={{name: name || '预览', image_url: imageURL.trim()}} />
            </span>
            <Input
              id="shop-item-image"
              type="url"
              placeholder="选填, 留空时显示第一件商品的图片"
              value={imageURL}
              onChange={(e) => setImageURL(e.target.value)}
              disabled={pending}
            />
          </div>
        </div>
        <SwitchField
          id="shop-item-enabled"
          label="上架"
          help="下架后商城中不再显示, 不能购买"
          checked={isEnabled}
          onChange={setIsEnabled}
          disabled={pending}
        />
      </FieldGroup>

      <FieldGroup title="商品" help={`购买一份得到的货币和物品, 不能为空; 已下架的物品不能购买; 最多 ${MAX_BUNDLE_LINES} 项`}>
        <BundleEditor
          drafts={goods}
          onChange={setGoods}
          items={items}
          currencies={currencies}
          max={MAX_BUNDLE_LINES}
          empty="还没有商品内容"
          disabled={pending}
        />
      </FieldGroup>

      <FieldGroup title="价格" help={`一份的价格, 可以是货币, 也可以用物品兑换; 留空为免费; 最多 ${MAX_BUNDLE_LINES} 项`}>
        <BundleEditor
          drafts={price}
          onChange={setPrice}
          items={items}
          currencies={currencies}
          max={MAX_BUNDLE_LINES}
          empty="免费"
          disabled={pending}
        />
      </FieldGroup>

      <FieldGroup title="原价" help="只用于在商城中展示折扣, 不影响实际价格">
        <BundleEditor
          drafts={originalPrice}
          onChange={setOriginalPrice}
          items={items}
          currencies={currencies}
          max={MAX_BUNDLE_LINES}
          empty="不展示原价"
          disabled={pending}
        />
      </FieldGroup>

      <FieldGroup title="购买条件" help={`需要全部满足才能购买; 最多 ${MAX_REQUIREMENTS} 个`}>
        <ConditionEditor
          drafts={requirements}
          onChange={setRequirements}
          types={types}
          items={items}
          currencies={currencies}
          scopes={scopes}
          max={MAX_REQUIREMENTS}
          empty="没有购买条件, 所有玩家都能购买"
          disabled={pending}
        />
      </FieldGroup>

      <FieldGroup title="限购和库存" help="0 为不限; 每天按服务器时区计算, 每赛季限购在没有进行中的赛季时不能购买; 退款不恢复限购数量">
        <div className="grid grid-cols-3 gap-3">
          <InputField id="shop-item-limit-total" label="每人限购" inputMode="numeric" suffix="份" value={limitTotal} onChange={setLimitTotal} disabled={pending} />
          <InputField id="shop-item-limit-daily" label="每天限购" inputMode="numeric" suffix="份" value={limitDaily} onChange={setLimitDaily} disabled={pending} />
          <InputField id="shop-item-limit-season" label="每赛季限购" inputMode="numeric" suffix="份" value={limitSeason} onChange={setLimitSeason} disabled={pending} />
        </div>
        <InputField
          id="shop-item-stock"
          label="全服剩余库存"
          inputMode="numeric"
          placeholder="留空为不限"
          suffix="份"
          value={stock}
          onChange={setStock}
          help={editing ? `已售出 ${editing.sold} 份; 保存时直接覆盖剩余库存, 打开表单后又售出的份数不会扣除` : '售完后不能购买, 退款时恢复'}
          disabled={pending}
        />
      </FieldGroup>

      <FieldGroup title="上架时间" help="按本地时间, 精确到分钟; 包含上架时间, 不包含下架时间; 留空为不限">
        <div className="grid grid-cols-2 gap-3">
          <div className="flex flex-col gap-2">
            <Label htmlFor="shop-item-starts-at">上架时间</Label>
            <Input id="shop-item-starts-at" type="datetime-local" value={startsAt} onChange={(e) => setStartsAt(e.target.value)} disabled={pending} />
          </div>
          <div className="flex flex-col gap-2">
            <Label htmlFor="shop-item-ends-at">下架时间</Label>
            <Input id="shop-item-ends-at" type="datetime-local" value={endsAt} onChange={(e) => setEndsAt(e.target.value)} disabled={pending} />
          </div>
        </div>
      </FieldGroup>

      <DialogFooter>
        <Button type="button" variant="outline" onClick={onClose} disabled={pending}>
          取消
        </Button>
        <SubmitButton pending={pending}>{editing ? '保存' : '新建'}</SubmitButton>
      </DialogFooter>
    </form>
  );
}
