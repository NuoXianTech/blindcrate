'use client';

import {useState} from 'react';
import {toast} from 'sonner';
import {Button} from '@/components/ui/button';
import {Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle} from '@/components/ui/dialog';
import {Input} from '@/components/ui/input';
import {Label} from '@/components/ui/label';
import {SubmitButton} from '@/components/common/form';
import {TierName} from '@/components/common/ranked';
import {check, useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';
import type {TierRequest} from '@/lib/services/admin';
import type {Tier} from '@/lib/services/ranked';
import {parseAmount} from '@/lib/utils';
import {COLOR_PATTERN, DEFAULT_COLOR, MAX_TIER_RATING} from './constants';

/**
 * 段位对话框的Props接口
 */
interface TierDialogProps {
  /** null: 关闭; "new": 新建; 段位: 编辑 */
  tier: Tier | 'new' | null;
  onClose: () => void;
  /** 保存成功后刷新列表 */
  onSaved: () => void;
}

/**
 * 新建或编辑段位
 */
export function TierDialog({tier, ...props}: TierDialogProps) {
  const editing = tier && tier !== 'new' ? tier : null;
  return (
    <Dialog open={!!tier} onOpenChange={(open) => !open && props.onClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{editing ? `编辑 ${editing.name}` : '新建段位'}</DialogTitle>
          <DialogDescription>段位按段位分实时计算, 保存后所有人的段位随即变化, 段位分不受影响</DialogDescription>
        </DialogHeader>
        {/* key 保证切换段位时表单重置 */}
        {tier && <TierForm key={editing?.id ?? 'new'} tier={editing} {...props} />}
      </DialogContent>
    </Dialog>
  );
}

/**
 * 段位表单, tier 为 null 时新建
 */
function TierForm({tier, onClose, onSaved}: Omit<TierDialogProps, 'tier'> & { tier: Tier | null }) {
  const {pending, submit} = useSubmit();
  const [name, setName] = useState(tier?.name ?? '');
  const [minRating, setMinRating] = useState(String(tier?.min_rating ?? ''));
  const [color, setColor] = useState(tier?.color ?? DEFAULT_COLOR);

  const handleSubmit = submit(async () => {
    const rating = parseAmount(minRating);
    check(rating === null || rating < 0 || rating > MAX_TIER_RATING ? `分数线需为 0-${MAX_TIER_RATING}` : null);
    const body: TierRequest = {name: name.trim(), min_rating: rating as number, color: color.trim().toLowerCase()};
    check(body.name.length === 0 || [...body.name].length > 32 ? '段位名称需为 1-32 个字符' : null);
    check(COLOR_PATTERN.test(body.color) ? null : '颜色需为 #RRGGBB 格式');
    if (tier) {
      await services.admin.updateTier(tier.id, body);
      toast.success(`已保存 ${body.name}`);
    } else {
      await services.admin.createTier(body);
      toast.success(`已新建 ${body.name}`);
    }
    onSaved();
    onClose();
  });

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4" noValidate>
      <div className="grid grid-cols-2 gap-3">
        <div className="flex flex-col gap-2">
          <Label htmlFor="tier-name">名称</Label>
          <Input id="tier-name" maxLength={32} placeholder="如 白银" value={name} onChange={(e) => setName(e.target.value)} disabled={pending} />
        </div>
        <div className="flex flex-col gap-2">
          <Label htmlFor="tier-min-rating">分数线</Label>
          <Input
            id="tier-min-rating"
            inputMode="numeric"
            placeholder="如 1100"
            className="font-mono tabular-nums"
            value={minRating}
            onChange={(e) => setMinRating(e.target.value)}
            disabled={pending}
          />
        </div>
      </div>
      <span className="-mt-2 text-xs text-muted-foreground">段位分不低于分数线时属于这个段位, 分数线不能与其他段位相同</span>

      <div className="flex flex-col gap-2">
        <Label htmlFor="tier-color">颜色</Label>
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
            id="tier-color"
            maxLength={7}
            className="font-mono"
            value={color}
            onChange={(e) => setColor(e.target.value)}
            disabled={pending}
          />
        </div>
      </div>

      <div className="flex items-center justify-between gap-4 rounded-lg border p-3">
        <span className="text-sm text-muted-foreground">预览</span>
        <TierName tier={{name: name.trim() || '段位名称', color: COLOR_PATTERN.test(color) ? color : DEFAULT_COLOR}} />
      </div>

      <DialogFooter>
        <Button type="button" variant="outline" onClick={onClose} disabled={pending}>
          取消
        </Button>
        <SubmitButton pending={pending}>{tier ? '保存' : '新建'}</SubmitButton>
      </DialogFooter>
    </form>
  );
}
