'use client';

import {useState} from 'react';
import {toast} from 'sonner';
import {Button} from '@/components/ui/button';
import {Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle} from '@/components/ui/dialog';
import {Input} from '@/components/ui/input';
import {Label} from '@/components/ui/label';
import {Switch} from '@/components/ui/switch';
import {Textarea} from '@/components/ui/textarea';
import {SubmitButton} from '@/components/common/form';
import {check, useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';
import type {LootPoolListItem, LootPoolRequest} from '@/lib/services/admin';
import {CODE_PATTERN} from './constants';

/**
 * 掉落池对话框的Props接口
 */
interface LootPoolDialogProps {
  /** null: 关闭; "new": 新建; 掉落池: 编辑 */
  pool: LootPoolListItem | 'new' | null;
  onClose: () => void;
  /** 保存成功后刷新列表 */
  onSaved: () => void;
}

/**
 * 新建或编辑掉落池的基本信息, 条目在条目对话框中编辑
 */
export function LootPoolDialog({pool, onClose, onSaved}: LootPoolDialogProps) {
  const editing = pool && pool !== 'new' ? pool : null;
  return (
    <Dialog open={!!pool} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{editing ? `编辑 ${editing.name}` : '新建掉落池'}</DialogTitle>
          <DialogDescription>
            {editing ? '地图的档位按代码引用掉落池, 修改代码后需要同步修改地图' : '新建后在条目中添加可以掉落的物品和货币'}
          </DialogDescription>
        </DialogHeader>
        {/* key 保证切换掉落池时表单重置 */}
        {pool && <LootPoolForm key={editing?.id ?? 'new'} pool={editing} onClose={onClose} onSaved={onSaved} />}
      </DialogContent>
    </Dialog>
  );
}

/**
 * 掉落池表单, pool 为 null 时新建
 */
function LootPoolForm({pool, onClose, onSaved}: Omit<LootPoolDialogProps, 'pool'> & { pool: LootPoolListItem | null }) {
  const {pending, submit} = useSubmit();
  const [code, setCode] = useState(pool?.code ?? '');
  const [name, setName] = useState(pool?.name ?? '');
  const [description, setDescription] = useState(pool?.description ?? '');
  const [enabled, setEnabled] = useState(pool?.is_enabled ?? true);

  const handleSubmit = submit(async () => {
    const body: LootPoolRequest = {
      code: code.trim(),
      name: name.trim(),
      description: description.trim(),
      is_enabled: enabled,
    };
    check(CODE_PATTERN.test(body.code) ? null : '代码需为 1-64 个小写字母、数字或下划线');
    check(body.name.length === 0 || [...body.name].length > 64 ? '名称需为 1-64 个字符' : null);
    check([...body.description].length > 255 ? '描述最多 255 个字符' : null);
    if (pool) {
      await services.admin.updateLootPool(pool.id, body);
      toast.success(`已保存 ${body.name}`);
    } else {
      await services.admin.createLootPool(body);
      toast.success(`已新建 ${body.name}`);
    }
    onSaved();
    onClose();
  });

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4" noValidate>
      <div className="grid grid-cols-2 gap-3">
        <div className="flex flex-col gap-2">
          <Label htmlFor="pool-code">代码</Label>
          <Input
            id="pool-code"
            maxLength={64}
            placeholder="如 warehouse_common"
            className="font-mono"
            value={code}
            onChange={(e) => setCode(e.target.value)}
            disabled={pending}
          />
        </div>
        <div className="flex flex-col gap-2">
          <Label htmlFor="pool-name">名称</Label>
          <Input id="pool-name" maxLength={64} value={name} onChange={(e) => setName(e.target.value)} disabled={pending} />
        </div>
      </div>

      <div className="flex flex-col gap-2">
        <Label htmlFor="pool-description">描述</Label>
        <Textarea
          id="pool-description"
          maxLength={255}
          placeholder="选填, 最多 255 个字符"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          disabled={pending}
        />
      </div>

      <div className="flex items-center justify-between gap-4 rounded-lg border p-3">
        <div className="flex flex-col gap-0.5">
          <Label htmlFor="pool-enabled">启用</Label>
          <span className="text-xs text-muted-foreground">停用后不能再抽取, 引用它的地图无法开局</span>
        </div>
        <Switch id="pool-enabled" checked={enabled} onCheckedChange={setEnabled} disabled={pending} />
      </div>

      <DialogFooter>
        <Button type="button" variant="outline" onClick={onClose} disabled={pending}>
          取消
        </Button>
        <SubmitButton pending={pending}>{pool ? '保存' : '新建'}</SubmitButton>
      </DialogFooter>
    </form>
  );
}
