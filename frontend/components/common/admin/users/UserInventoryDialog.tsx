'use client';

import {useCallback, useEffect, useState} from 'react';
import Link from 'next/link';
import {ScrollText} from 'lucide-react';
import {nanoid} from 'nanoid';
import {toast} from 'sonner';
import {cn} from 'cn';
import {Badge} from '@/components/ui/badge';
import {Button} from '@/components/ui/button';
import {DialogFooter} from '@/components/ui/dialog';
import {Input} from '@/components/ui/input';
import {Label} from '@/components/ui/label';
import {Skeleton} from '@/components/ui/skeleton';
import {SubmitButton} from '@/components/common/form';
import {ItemImage} from '@/components/common/item';
import {check, useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';
import type {AdminItem, AdminUser} from '@/lib/services/admin';
import {toErrorMessage} from '@/lib/services/core';
import type {UserItem} from '@/lib/services/inventory';
import {qualityOf} from '@/lib/quality';
import {parseAmount} from '@/lib/utils';
import {DirectionToggle} from './DirectionToggle';
import {ItemCombobox} from './ItemCombobox';
import {UserAssetDialog} from './UserAssetDialog';
import {ITEM_LIMIT, MAX_QUANTITY, transactionsHref, type Direction} from './constants';

/**
 * 查看和调整用户的背包物品
 */
export function UserInventoryDialog({user, onClose}: { user: AdminUser | null; onClose: () => void }) {
  return (
    <UserAssetDialog user={user} title="背包" onClose={onClose}>
      {/* key 保证切换用户时表单重置 */}
      {user && <InventoryPanel key={user.id} user={user} />}
    </UserAssetDialog>
  );
}

/**
 * 背包物品列表 + 发放/扣除表单
 */
function InventoryPanel({user}: { user: AdminUser }) {
  const {pending, submit} = useSubmit();
  const [owned, setOwned] = useState<{ total: number; items: UserItem[] } | null>(null);
  // 可发放的物品(已上架)
  const [grantable, setGrantable] = useState<AdminItem[]>([]);
  const [direction, setDirection] = useState<Direction>('add');
  const [itemID, setItemID] = useState<number | null>(null);
  const [quantity, setQuantity] = useState('1');
  const [remark, setRemark] = useState('');
  // 幂等键, 每次打开表单和每次成功后生成新的
  const [key, setKey] = useState(() => nanoid());

  const load = useCallback(async () => {
    try {
      setOwned(await services.admin.listUserInventory(user.id, {current: 1, size: ITEM_LIMIT}));
    } catch (err) {
      toast.error(toErrorMessage(err));
    }
  }, [user.id]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 打开时拉取背包
    void load();
  }, [load]);

  useEffect(() => {
    services.admin.listItems({current: 1, size: ITEM_LIMIT, is_enabled: true})
        .then((data) => setGrantable(data.items))
        .catch((err) => toast.error(toErrorMessage(err)));
  }, []);

  // 发放只能选已上架的物品; 扣除选用户已有的, 包括已下架和已删除的
  const options =
    direction === 'add' ?
      grantable.map((i) => ({id: i.id, name: i.name, quality: i.quality})) :
      (owned?.items ?? []).map((i) => ({id: i.item_id, name: i.name, quality: i.quality}));
  const ownedQuantity = owned?.items.find((i) => i.item_id === itemID)?.quantity ?? 0;
  const parsed = parseAmount(quantity);
  const delta = parsed && parsed > 0 ? (direction === 'add' ? parsed : -parsed) : null;
  const after = itemID !== null && delta !== null ? ownedQuantity + delta : null;
  const selectedName = options.find((o) => o.id === itemID)?.name;

  const changeDirection = (value: Direction) => {
    setDirection(value);
    setItemID(null);
  };

  // 点背包里的物品, 直接选中扣除
  const pickDeduct = (item: UserItem) => {
    setDirection('sub');
    setItemID(item.item_id);
  };

  const handleSubmit = submit(async () => {
    check(itemID !== null ? null : '请选择物品');
    check(parsed !== null && parsed > 0 && parsed <= MAX_QUANTITY ? null : `数量需为 1-${MAX_QUANTITY} 的整数`);
    check(after !== null && after < 0 ? '扣除后数量不能为负' : null);
    check(remark.trim() ? null : '请填写备注');
    await services.admin.adjustUserInventory(user.id, {
      item_id: itemID!,
      quantity: delta!,
      remark: remark.trim(),
      idempotency_key: key,
    });
    toast.success(`已给 ${user.username} ${direction === 'add' ? '发放' : '扣除'} ${selectedName} ×${parsed}`);
    setQuantity('1');
    setRemark('');
    setKey(nanoid());
    // 扣完了物品会从列表消失, 不能再选
    if (after === 0) setItemID(null);
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
                      'flex items-center gap-3 border-b px-3 py-2 text-left text-sm last:border-b-0 hover:bg-accent',
                      direction === 'sub' && i.item_id === itemID && 'bg-accent',
                  )}
                >
                  <span className="relative flex size-8 shrink-0 items-center justify-center overflow-hidden rounded-md border bg-background">
                    <ItemImage item={i} className={cn(i.is_placeholder && 'opacity-60')} />
                    <span className={cn('absolute inset-x-0 top-0 h-0.5', q.bar)} />
                  </span>
                  <span className={cn('min-w-0 flex-1 truncate font-medium', q.text)}>{i.name}</span>
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
        <Skeleton className="h-20 rounded-lg" />
      )}

      <form onSubmit={handleSubmit} className="flex flex-col gap-4" noValidate>
        <DirectionToggle value={direction} onChange={changeDirection} labels={['发放', '扣除']} disabled={pending} />
        <div className="grid grid-cols-[1fr_7rem] gap-3">
          <div className="flex min-w-0 flex-col gap-2">
            <Label>物品</Label>
            <ItemCombobox
              options={options}
              value={itemID}
              onChange={setItemID}
              placeholder={direction === 'add' ? '选择已上架的物品' : '选择背包中的物品'}
              disabled={pending}
            />
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
          <span className={cn('mr-auto text-sm text-muted-foreground', after !== null && after < 0 && 'text-destructive')}>
            {after !== null && `调整后 ×${after}`}
          </span>
          <Button type="button" variant="outline" asChild>
            <Link href={transactionsHref('inventory', user.id)}>
              <ScrollText />
              查看记录
            </Link>
          </Button>
          <SubmitButton pending={pending} variant={direction === 'sub' ? 'destructive' : 'default'}>
            {direction === 'add' ? '发放' : '扣除'}
          </SubmitButton>
        </DialogFooter>
      </form>
    </div>
  );
}
