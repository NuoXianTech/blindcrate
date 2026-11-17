'use client';

import {useEffect, useState} from 'react';
import {nanoid} from 'nanoid';
import {Sparkles} from 'lucide-react';
import {cn} from 'cn';
import {Button} from '@/components/ui/button';
import {Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle} from '@/components/ui/dialog';
import {Input} from '@/components/ui/input';
import {Label} from '@/components/ui/label';
import {Skeleton} from '@/components/ui/skeleton';
import {CrateStage} from '@/components/common/crate';
import {SubmitButton} from '@/components/common/form';
import {RewardList} from '@/components/common/reward';
import {check, useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';
import {toErrorMessage} from '@/lib/services/core';
import {USE_MODE, type ItemUseResult, type ItemUseView} from '@/lib/services/inventory';

/**
 * 使用对话框的Props接口
 */
interface UseDialogProps {
  /** 要使用的物品, null 时关闭 */
  item: { item_id: number; name: string } | null;
  onClose: () => void;
  /** 每使用一次后调用, 用于刷新背包 */
  onUsed: () => void;
}

/**
 * 在背包中使用物品的对话框: 按后端给出的使用方式展示, 不依赖效果代码, 后端新增的效果不用改这里
 * 按概率得到一项的(箱子)播放开箱动画, 其他的展示会得到什么, 需要时让玩家选一项或填写内容
 */
export function UseDialog({item, onClose, onUsed}: UseDialogProps) {
  // 结果揭晓之前不能关闭, 否则已经发放的结果没有展示
  const [locked, setLocked] = useState(false);

  return (
    <Dialog open={!!item} onOpenChange={(open) => !open && !locked && onClose()}>
      {/* 每次打开都重新挂载, 重新读取物品的内容 */}
      {item && <UseContent key={item.item_id} item={item} locked={locked} onClose={onClose} onUsed={onUsed} onLockedChange={setLocked} />}
    </Dialog>
  );
}

/** 对话框的内容: 读取物品的使用方式后按方式展示 */
function UseContent({item, locked, onClose, onUsed, onLockedChange}: {
  item: { item_id: number; name: string };
  locked: boolean;
  onClose: () => void;
  onUsed: () => void;
  onLockedChange: (locked: boolean) => void;
}) {
  const [view, setView] = useState<ItemUseView | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let alive = true;
    services.inventory.getUse(item.item_id)
        .then((data) => alive && setView(data))
        .catch((err) => alive && setError(toErrorMessage(err)));
    return () => {
      alive = false;
    };
  }, [item.item_id]);

  const random = view?.mode === USE_MODE.random;
  return (
    <DialogContent className={cn('max-h-[90vh] overflow-y-auto', random ? 'sm:max-w-2xl' : 'sm:max-w-md')} showCloseButton={!locked}>
      <DialogHeader>
        <DialogTitle>{view ? `${view.label} · ${item.name}` : item.name}</DialogTitle>
        <DialogDescription>{view?.summary ?? (error ? '这个物品现在不能使用' : '正在读取…')}</DialogDescription>
      </DialogHeader>
      {error ? (
        <>
          <p className="rounded-lg border bg-card px-4 py-6 text-center text-sm text-muted-foreground">{error}</p>
          <DialogFooter>
            <Button variant="outline" onClick={onClose}>
              关闭
            </Button>
          </DialogFooter>
        </>
      ) : !view ? (
        <>
          <Skeleton className="h-24 rounded-lg" />
          <Skeleton className="h-10 rounded-lg" />
        </>
      ) : random ? (
        <CrateStage view={view} onClose={onClose} onUsed={onUsed} onLockedChange={onLockedChange} />
      ) : (
        <UseStage view={view} onClose={onClose} onUsed={onUsed} />
      )}
    </DialogContent>
  );
}

/**
 * 不按概率的使用界面: 展示会得到的内容, 自选时选一项, 需要时填写内容; 使用后展示得到的内容和结果
 */
function UseStage({view, onClose, onUsed}: {
  view: ItemUseView;
  onClose: () => void;
  onUsed: () => void;
}) {
  const {pending, submit} = useSubmit();
  const [quantity, setQuantity] = useState(view.quantity);
  // 幂等键在使用成功后才换新的, 请求失败后重试不会多用一个
  const [idempotencyKey, setIdempotencyKey] = useState(() => nanoid());
  const [choice, setChoice] = useState<number | null>(null);
  const [input, setInput] = useState('');
  const [result, setResult] = useState<ItemUseResult | null>(null);
  const choosing = view.mode === USE_MODE.choose;

  const handleUse = submit(async () => {
    check(choosing && choice === null ? '请先选择一项' : null);
    const text = input.trim();
    check(view.input && (!text || [...text].length > view.input.max_length) ? `${view.input.label}需为 1-${view.input.max_length} 个字符` : null);
    const used = await services.inventory.use(view.item_id, {
      idempotency_key: idempotencyKey,
      choice: choosing ? choice ?? 0 : undefined,
      input: view.input ? text : undefined,
    });
    setIdempotencyKey(nanoid());
    setQuantity(used.quantity);
    setResult(used);
    onUsed();
  });

  if (result) {
    return (
      <>
        <div role="status" aria-live="polite" className="flex flex-col gap-3 rounded-lg border bg-card p-4">
          <span className="text-sm font-medium">{result.message || (result.received.length > 0 ? '已获得' : '已使用')}</span>
          {result.received.length > 0 && <RewardList lines={result.received} />}
        </div>
        <DialogFooter className="sm:items-center">
          <Remaining quantity={quantity} />
          <Button variant="outline" onClick={onClose}>
            关闭
          </Button>
          {quantity > 0 && (
            <Button onClick={() => setResult(null)}>
              再用一个
            </Button>
          )}
        </DialogFooter>
      </>
    );
  }

  return (
    <>
      {view.mode === USE_MODE.grant && view.options[0] && (
        <div className="flex flex-col gap-2 rounded-lg border bg-card p-4">
          <span className="text-sm font-medium">使用后得到</span>
          <RewardList lines={view.options[0].lines} />
        </div>
      )}
      {choosing && (
        <div role="radiogroup" aria-label="选择一项" className="flex max-h-72 flex-col gap-2 overflow-y-auto">
          {view.options.map((option, i) => (
            <button
              key={i}
              type="button"
              role="radio"
              aria-checked={choice === i}
              onClick={() => setChoice(i)}
              className={cn(
                  'rounded-lg border bg-card p-3 text-left transition-colors outline-none hover:bg-accent focus-visible:ring-[3px] focus-visible:ring-ring/50',
                  choice === i && 'border-primary ring-2 ring-primary/40',
              )}
            >
              <RewardList lines={option.lines} />
            </button>
          ))}
        </div>
      )}
      {view.input && (
        <div className="flex flex-col gap-2">
          <Label htmlFor="use-input">{view.input.label}</Label>
          <Input
            id="use-input"
            value={input}
            maxLength={view.input.max_length}
            placeholder={view.input.placeholder}
            onChange={(e) => setInput(e.target.value)}
          />
        </div>
      )}
      <DialogFooter className="sm:items-center">
        <Remaining quantity={quantity} />
        <Button variant="outline" onClick={onClose} disabled={pending}>
          取消
        </Button>
        <SubmitButton type="button" pending={pending} disabled={quantity < 1} onClick={handleUse}>
          {!pending && <Sparkles />}
          使用
        </SubmitButton>
      </DialogFooter>
    </>
  );
}

/** 背包中剩余的数量 */
function Remaining({quantity}: { quantity: number }) {
  return (
    <span className="text-sm text-muted-foreground sm:mr-auto">
      剩余 <span className="font-mono tabular-nums text-foreground">×{quantity}</span>
    </span>
  );
}
