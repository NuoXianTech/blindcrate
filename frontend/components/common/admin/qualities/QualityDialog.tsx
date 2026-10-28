'use client';

import {useState} from 'react';
import {useRouter} from 'next/navigation';
import {toast} from 'sonner';
import {Button} from '@/components/ui/button';
import {Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle} from '@/components/ui/dialog';
import {Input} from '@/components/ui/input';
import {Label} from '@/components/ui/label';
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select';
import {SubmitButton} from '@/components/common/form';
import {check, useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';
import type {AdminQuality, QualityRequest} from '@/lib/services/admin';
import {refreshQualities} from '@/lib/site-actions';
import {parseAmount} from '@/lib/utils';
import {
  CODE_PATTERN,
  COLOR_PATTERN,
  EFFECTS,
  MAX_LEVEL,
  MAX_SCAN_MS,
  MAX_SORT_ORDER,
  MIN_SCAN_MS,
  NO_EFFECT,
} from './constants';
import {QualityPreview} from './QualityPreview';

/** 新建品质时的默认颜色 */
const DEFAULT_COLOR = '#a1a5ab';

/**
 * 品质对话框的Props接口
 */
interface QualityDialogProps {
  /** null: 关闭; "new": 新建; 品质: 编辑 */
  quality: AdminQuality | 'new' | null;
  /** 新建时默认填写的等级, 为现有最高等级加一 */
  nextLevel: number;
  onClose: () => void;
  /** 保存成功后刷新列表 */
  onSaved: () => void;
}

/**
 * 新建或编辑品质: 等级和代码创建后不能修改
 */
export function QualityDialog({quality, ...props}: QualityDialogProps) {
  const editing = quality && quality !== 'new' ? quality : null;
  return (
    <Dialog open={!!quality} onOpenChange={(open) => !open && props.onClose()}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{editing ? `编辑 ${editing.name}` : '新建品质'}</DialogTitle>
          <DialogDescription>保存后全站的品质名称和颜色立即更新; 新的扫描时长只影响之后开局的对局</DialogDescription>
        </DialogHeader>
        {/* key 保证切换品质时表单重置 */}
        {quality && <QualityForm key={editing?.level ?? 'new'} quality={editing} {...props} />}
      </DialogContent>
    </Dialog>
  );
}

/** 输入框的原文换算为整数, 格式不对或超出范围时中断提交 */
function readInteger(text: string, label: string, min: number, max: number): number {
  const value = parseAmount(text);
  check(value === null || value < min || value > max ? `${label}需为 ${min}-${max}` : null);
  return value as number;
}

/**
 * 品质表单, quality 为 null 时新建
 */
function QualityForm({quality, nextLevel, onClose, onSaved}: Omit<QualityDialogProps, 'quality'> & {
  quality: AdminQuality | null;
}) {
  const router = useRouter();
  const {pending, submit} = useSubmit();
  const [level, setLevel] = useState(String(quality?.level ?? nextLevel));
  const [code, setCode] = useState(quality?.code ?? '');
  const [name, setName] = useState(quality?.name ?? '');
  const [color, setColor] = useState(quality?.color ?? DEFAULT_COLOR);
  const [effect, setEffect] = useState(quality?.effect || NO_EFFECT);
  const [scanMs, setScanMs] = useState(String(quality?.scan_ms ?? 1000));
  const [sortOrder, setSortOrder] = useState(String(quality?.sort_order ?? nextLevel * 10));

  const handleSubmit = submit(async () => {
    const body: QualityRequest = {
      name: name.trim(),
      color: color.trim().toLowerCase(),
      effect: effect === NO_EFFECT ? '' : effect,
      scan_ms: readInteger(scanMs, '扫描时长', MIN_SCAN_MS, MAX_SCAN_MS),
      sort_order: readInteger(sortOrder, '排序', -MAX_SORT_ORDER, MAX_SORT_ORDER),
    };
    check(body.name.length === 0 || [...body.name].length > 32 ? '名称需为 1-32 个字符' : null);
    check(COLOR_PATTERN.test(body.color) ? null : '颜色需为 #RRGGBB 格式');
    if (quality) {
      await services.admin.updateQuality(quality.level, body);
      toast.success(`已保存 ${body.name}`);
    } else {
      const created = {...body, level: readInteger(level, '等级', 1, MAX_LEVEL), code: code.trim()};
      check(CODE_PATTERN.test(created.code) ? null : '代码需为 1-32 个小写字母、数字或下划线');
      await services.admin.createQuality(created);
      toast.success(`已新建 ${body.name}`);
    }
    // 刷新服务端缓存的品质, 全站的名称和颜色随即更新
    await refreshQualities();
    router.refresh();
    onSaved();
    onClose();
  });

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4" noValidate>
      <div className="grid grid-cols-2 gap-3">
        <div className="flex flex-col gap-2">
          <Label htmlFor="quality-level">等级</Label>
          <Input
            id="quality-level"
            inputMode="numeric"
            className="font-mono tabular-nums"
            value={level}
            onChange={(e) => setLevel(e.target.value)}
            disabled={pending || !!quality}
          />
        </div>
        <div className="flex flex-col gap-2">
          <Label htmlFor="quality-code">代码</Label>
          <Input
            id="quality-code"
            maxLength={32}
            placeholder="如 legendary"
            className="font-mono"
            value={code}
            onChange={(e) => setCode(e.target.value)}
            disabled={pending || !!quality}
          />
        </div>
      </div>
      <span className="-mt-2 text-xs text-muted-foreground">
        等级越大越稀有, 物品按等级引用品质; 等级和代码创建后不能修改
      </span>

      <div className="flex flex-col gap-2">
        <Label htmlFor="quality-name">名称</Label>
        <Input id="quality-name" maxLength={32} value={name} onChange={(e) => setName(e.target.value)} disabled={pending} />
      </div>

      <div className="grid grid-cols-2 gap-3">
        <div className="flex flex-col gap-2">
          <Label htmlFor="quality-color">颜色</Label>
          <div className="flex gap-2">
            <input
              type="color"
              aria-label="选择颜色"
              className="h-9 w-10 shrink-0 cursor-pointer rounded-md border bg-transparent p-1 disabled:cursor-not-allowed"
              value={COLOR_PATTERN.test(color) ? color : DEFAULT_COLOR}
              onChange={(e) => setColor(e.target.value)}
              disabled={pending}
            />
            <Input
              id="quality-color"
              maxLength={7}
              className="font-mono"
              value={color}
              onChange={(e) => setColor(e.target.value)}
              disabled={pending}
            />
          </div>
        </div>
        <div className="flex flex-col gap-2">
          <Label htmlFor="quality-effect">特效</Label>
          <Select value={effect} onValueChange={setEffect} disabled={pending}>
            <SelectTrigger id="quality-effect" className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {EFFECTS.map((e) => (
                <SelectItem key={e.value} value={e.value}>
                  {e.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>
      <span className="-mt-2 text-xs text-muted-foreground">炫彩渐变的文字和色条用流动渐变, 颜色只在不支持特效的地方使用</span>

      <div className="grid grid-cols-2 gap-3">
        <div className="flex flex-col gap-2">
          <Label htmlFor="quality-scan-ms">扫描时长(毫秒)</Label>
          <Input
            id="quality-scan-ms"
            inputMode="numeric"
            className="font-mono tabular-nums"
            value={scanMs}
            onChange={(e) => setScanMs(e.target.value)}
            disabled={pending}
          />
        </div>
        <div className="flex flex-col gap-2">
          <Label htmlFor="quality-sort-order">排序</Label>
          <Input
            id="quality-sort-order"
            inputMode="numeric"
            className="font-mono tabular-nums"
            value={sortOrder}
            onChange={(e) => setSortOrder(e.target.value)}
            disabled={pending}
          />
        </div>
      </div>
      <span className="-mt-2 text-xs text-muted-foreground">
        开箱时逐件扫描藏品, 品质越高可以扫得越久, 为 {MIN_SCAN_MS}-{MAX_SCAN_MS} 毫秒; 排序从小到大决定筛选和图例中的顺序
      </span>

      <div className="flex items-center justify-between gap-4 rounded-lg border p-3">
        <span className="text-sm text-muted-foreground">预览</span>
        <QualityPreview name={name.trim() || '品质名称'} color={color} effect={effect} />
      </div>

      <DialogFooter>
        <Button type="button" variant="outline" onClick={onClose} disabled={pending}>
          取消
        </Button>
        <SubmitButton pending={pending}>{quality ? '保存' : '新建'}</SubmitButton>
      </DialogFooter>
    </form>
  );
}
