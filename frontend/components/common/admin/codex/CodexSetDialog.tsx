'use client';

import {useState} from 'react';
import {toast} from 'sonner';
import {Button} from '@/components/ui/button';
import {Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle} from '@/components/ui/dialog';
import {Label} from '@/components/ui/label';
import {Skeleton} from '@/components/ui/skeleton';
import {Textarea} from '@/components/ui/textarea';
import {BundleEditor, fromBundleDrafts, toBundleDrafts} from '@/components/common/admin/editors';
import {FieldGroup, InputField, SwitchField} from '@/components/common/admin/venues/Fields';
import {SubmitButton} from '@/components/common/form';
import {check, useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';
import type {AdminCodexSet, CodexSetRequest} from '@/lib/services/codex';
import type {AdminItem} from '@/lib/services/item';
import type {LootPoolListItem} from '@/lib/services/loot';
import type {Currency} from '@/lib/services/wallet';
import {parseAmount} from '@/lib/utils';
import {SetItemsEditor} from './SetItemsEditor';
import {CODE_PATTERN, MAX_BUNDLE_LINES, MAX_SET_ITEMS, MAX_SORT_ORDER} from './constants';

/** 编辑收集组用到的物品、货币和掉落池 */
export interface CodexSetFormOptions {
  items: AdminItem[];
  currencies: Currency[];
  pools: LootPoolListItem[];
  loaded: boolean;
}

/** 对话框的目标: new 新建; edit 编辑; copy 以已有收集组为模板新建 */
export type CodexSetTarget = { mode: 'new' } | { mode: 'edit' | 'copy'; set: AdminCodexSet };

/**
 * 收集组对话框的Props接口
 */
interface CodexSetDialogProps {
  /** null 时关闭 */
  target: CodexSetTarget | null;
  options: CodexSetFormOptions;
  onClose: () => void;
  /** 保存成功后刷新列表 */
  onSaved: () => void;
}

/**
 * 新建、编辑或复制收集组
 */
export function CodexSetDialog({target, options, onClose, onSaved}: CodexSetDialogProps) {
  const title = !target ?
    '' :
    target.mode === 'edit' ? `编辑 ${target.set.name}` : target.mode === 'copy' ? `复制 ${target.set.name}` : '新建收集组';
  return (
    <Dialog open={!!target} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>玩家集齐组内全部上架的藏品后可以领取一次奖励; 修改藏品后已领取的不受影响</DialogDescription>
        </DialogHeader>
        {target && !options.loaded && (
          <div className="flex flex-col gap-2">
            <Skeleton className="h-9" />
            <Skeleton className="h-9" />
            <Skeleton className="h-9" />
          </div>
        )}
        {/* key 保证切换收集组时表单重置 */}
        {target && options.loaded && (
          <CodexSetForm
            key={target.mode === 'new' ? 'new' : `${target.mode}:${target.set.id}`}
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

/**
 * 收集组表单
 */
function CodexSetForm({target, options, onClose, onSaved}: Omit<CodexSetDialogProps, 'target'> & { target: CodexSetTarget }) {
  const {pending, submit} = useSubmit();
  // 编辑和复制时以已有收集组为初始值, 复制时代码留空
  const source = target.mode === 'new' ? null : target.set;
  const editing = target.mode === 'edit' ? target.set : null;
  const {items, currencies, pools} = options;

  const [code, setCode] = useState(editing?.code ?? '');
  const [name, setName] = useState(source?.name ?? '');
  const [description, setDescription] = useState(source?.description ?? '');
  const [itemIDs, setItemIDs] = useState<number[]>(source?.item_ids ?? []);
  const [sortOrder, setSortOrder] = useState(String(source?.sort_order ?? 0));
  const [isEnabled, setIsEnabled] = useState(source?.is_enabled ?? true);
  const [reward, setReward] = useState(() => toBundleDrafts(source?.reward ?? {}, currencies));

  const handleSubmit = submit(async () => {
    const sort = parseAmount(sortOrder.trim());
    check(sort === null || Math.abs(sort) > MAX_SORT_ORDER ? `排序需为 -${MAX_SORT_ORDER} 到 ${MAX_SORT_ORDER} 的整数` : null);
    const body: CodexSetRequest = {
      code: editing?.code ?? code.trim(),
      name: name.trim(),
      description: description.trim(),
      item_ids: itemIDs,
      reward: fromBundleDrafts(reward, currencies, '奖励'),
      sort_order: sort!,
      is_enabled: isEnabled,
    };
    check(CODE_PATTERN.test(body.code) ? null : '代码需为 1-32 个小写字母、数字或下划线');
    check(body.name.length === 0 || [...body.name].length > 64 ? '名称需为 1-64 个字符' : null);
    check(body.item_ids.length === 0 ? '收集组至少包含一种藏品' : null);
    check(reward.length === 0 ? '奖励不能为空' : null);
    if (editing) {
      await services.codexAdmin.updateSet(editing.id, body);
      toast.success(`已保存 ${body.name}`);
    } else {
      await services.codexAdmin.createSet(body);
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
            id="codex-set-code"
            label="代码"
            placeholder="如 dock_all"
            maxLength={32}
            value={code}
            onChange={setCode}
            help="领取记录按收集组计, 创建后不能修改"
            disabled={pending || !!editing}
          />
          <InputField id="codex-set-name" label="名称" placeholder="如 码头全收集" maxLength={64} value={name} onChange={setName} disabled={pending} />
        </div>
        <div className="flex flex-col gap-2">
          <Label htmlFor="codex-set-description">说明</Label>
          <Textarea
            id="codex-set-description"
            maxLength={255}
            placeholder="选填, 最多 255 个字符"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            disabled={pending}
          />
        </div>
        <div className="grid grid-cols-2 gap-3">
          <InputField
            id="codex-set-sort"
            label="排序"
            inputMode="numeric"
            value={sortOrder}
            onChange={setSortOrder}
            help="从小到大排列"
            disabled={pending}
          />
        </div>
        <SwitchField
          id="codex-set-enabled"
          label="启用"
          help="停用后玩家看不到, 也不能领取"
          checked={isEnabled}
          onChange={setIsEnabled}
          disabled={pending}
        />
      </FieldGroup>

      <FieldGroup
        title="藏品"
        help={`只能加入藏品, 最多 ${MAX_SET_ITEMS} 种; 已下架、已删除或不再是藏品的不计入进度, 这期间集齐其余藏品的玩家就可以领取; 批量添加只加入上架中的藏品`}
      >
        <SetItemsEditor value={itemIDs} onChange={setItemIDs} items={items} pools={pools} max={MAX_SET_ITEMS} disabled={pending} />
      </FieldGroup>

      <FieldGroup title="奖励" help={`集齐后发放的货币和物品, 不能为空; 已下架的物品按回收价折成货币; 最多 ${MAX_BUNDLE_LINES} 项`}>
        <BundleEditor
          drafts={reward}
          onChange={setReward}
          items={items}
          currencies={currencies}
          max={MAX_BUNDLE_LINES}
          empty="还没有奖励"
          disabled={pending}
        />
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
