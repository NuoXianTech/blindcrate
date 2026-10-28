'use client';

import {useId, useState} from 'react';
import {toast} from 'sonner';
import {Button} from '@/components/ui/button';
import {Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle} from '@/components/ui/dialog';
import {Input} from '@/components/ui/input';
import {Label} from '@/components/ui/label';
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select';
import {Switch} from '@/components/ui/switch';
import {Textarea} from '@/components/ui/textarea';
import {SubmitButton} from '@/components/common/form';
import type {ScopeOption} from '@/components/common/admin/editors';
import {check, useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';
import type {AdminLeaderboard, LeaderboardRequest, StatMetric} from '@/lib/services/admin';
import {LEADERBOARD_PERIOD, type LeaderboardPeriod} from '@/lib/services/leaderboard';
import {parseAmount} from '@/lib/utils';
import {CODE_PATTERN, DEFAULT_SIZE, MAX_SIZE, MIN_SIZE, PERIODS} from './constants';

/** 排序值的范围, 与后端一致 */
const MAX_SORT_ORDER = 1_000_000;
/** 上榜最少局数的上限 */
const MAX_MIN_MATCHES = 1_000_000;

/**
 * 榜单对话框的Props接口
 */
interface LeaderboardDialogProps {
  /** null: 关闭; "new": 新建; 榜单: 编辑 */
  board: AdminLeaderboard | 'new' | null;
  /** 可以排名的指标 */
  metrics: StatMetric[];
  /** 统计范围输入框的候选项, 也可以手动输入 */
  scopes: ScopeOption[];
  onClose: () => void;
  /** 保存成功后刷新列表 */
  onSaved: () => void;
}

/**
 * 新建或编辑榜单: 指标、统计范围和周期决定排名的数据, 上榜人数和最少局数决定谁能上榜
 */
export function LeaderboardDialog({board, ...props}: LeaderboardDialogProps) {
  const editing = board && board !== 'new' ? board : null;
  return (
    <Dialog open={!!board} onOpenChange={(open) => !open && props.onClose()}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{editing ? `编辑 ${editing.name}` : '新建榜单'}</DialogTitle>
          <DialogDescription>修改后玩家看到的前 N 名立即更新, 统计数据不受影响</DialogDescription>
        </DialogHeader>
        {/* key 保证切换榜单时表单重置 */}
        {board && <LeaderboardForm key={editing?.id ?? 'new'} board={editing} {...props} />}
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
 * 榜单表单, board 为 null 时新建
 */
function LeaderboardForm({board, metrics, scopes, onClose, onSaved}: Omit<LeaderboardDialogProps, 'board'> & {
  board: AdminLeaderboard | null;
}) {
  const scopeListID = useId();
  const {pending, submit} = useSubmit();
  const [code, setCode] = useState(board?.code ?? '');
  const [name, setName] = useState(board?.name ?? '');
  const [description, setDescription] = useState(board?.description ?? '');
  const [metric, setMetric] = useState(board?.metric ?? '');
  const [scope, setScope] = useState(board?.scope ?? 'all');
  const [period, setPeriod] = useState<LeaderboardPeriod>(board?.period ?? LEADERBOARD_PERIOD.total);
  const [size, setSize] = useState(String(board?.size ?? DEFAULT_SIZE));
  const [minMatches, setMinMatches] = useState(String(board?.min_matches ?? 0));
  const [sortOrder, setSortOrder] = useState(String(board?.sort_order ?? 0));
  const [enabled, setEnabled] = useState(board?.is_enabled ?? true);

  const handleSubmit = submit(async () => {
    const body: LeaderboardRequest = {
      code: code.trim(),
      name: name.trim(),
      description: description.trim(),
      metric,
      scope: scope.trim(),
      period,
      size: readInteger(size, '上榜人数', MIN_SIZE, MAX_SIZE),
      min_matches: readInteger(minMatches, '最少局数', 0, MAX_MIN_MATCHES),
      sort_order: readInteger(sortOrder, '排序', -MAX_SORT_ORDER, MAX_SORT_ORDER),
      is_enabled: enabled,
    };
    check(CODE_PATTERN.test(body.code) ? null : '代码需为 1-64 个小写字母、数字或下划线');
    check(body.name.length === 0 || [...body.name].length > 64 ? '名称需为 1-64 个字符' : null);
    check([...body.description].length > 255 ? '描述最多 255 个字符' : null);
    check(!body.metric ? '请选择指标' : null);
    check(!body.scope ? '请填写统计范围' : null);
    if (board) {
      await services.admin.updateLeaderboard(board.id, body);
      toast.success(`已保存 ${body.name}`);
    } else {
      await services.admin.createLeaderboard(body);
      toast.success(`已新建 ${body.name}`);
    }
    onSaved();
    onClose();
  });

  // 指标失效(如货币已删除)或候选项未加载时仍显示原值, 方便管理员改选
  const unknownMetric = metric && !metrics.some((m) => m.key === metric);

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4" noValidate>
      <div className="grid grid-cols-2 gap-3">
        <div className="flex flex-col gap-2">
          <Label htmlFor="board-code">代码</Label>
          <Input
            id="board-code"
            maxLength={64}
            placeholder="如 ranked_wins"
            className="font-mono"
            value={code}
            onChange={(e) => setCode(e.target.value)}
            disabled={pending}
          />
        </div>
        <div className="flex flex-col gap-2">
          <Label htmlFor="board-name">名称</Label>
          <Input id="board-name" maxLength={64} value={name} onChange={(e) => setName(e.target.value)} disabled={pending} />
        </div>
      </div>

      <div className="flex flex-col gap-2">
        <Label htmlFor="board-description">描述</Label>
        <Textarea
          id="board-description"
          maxLength={255}
          placeholder="选填, 显示在榜单上方"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          disabled={pending}
        />
      </div>

      <div className="flex flex-col gap-2">
        <Label htmlFor="board-metric">指标</Label>
        <Select value={metric} onValueChange={setMetric} disabled={pending}>
          <SelectTrigger id="board-metric" className="w-full">
            <SelectValue placeholder="选择排名的指标" />
          </SelectTrigger>
          <SelectContent>
            {unknownMetric && <SelectItem value={metric}>{metrics.length ? `未知指标 ${metric}` : metric}</SelectItem>}
            {metrics.map((m) => (
              <SelectItem key={m.key} value={m.key}>
                {m.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <span className="text-xs text-muted-foreground">从大到小排名, 同分时先达到的在前</span>
      </div>

      <div className="grid grid-cols-2 gap-3">
        <div className="flex flex-col gap-2">
          <Label htmlFor="board-scope">统计范围</Label>
          <Input
            id="board-scope"
            list={scopeListID}
            maxLength={64}
            placeholder="如 all、mode:ranked"
            className="font-mono"
            value={scope}
            onChange={(e) => setScope(e.target.value)}
            disabled={pending}
          />
          <datalist id={scopeListID}>
            {scopes.map((s) => (
              <option key={s.value} value={s.value}>
                {s.label}
              </option>
            ))}
          </datalist>
        </div>
        <div className="flex flex-col gap-2">
          <Label htmlFor="board-period">统计周期</Label>
          <Select value={period} onValueChange={(value) => setPeriod(value as LeaderboardPeriod)} disabled={pending}>
            <SelectTrigger id="board-period" className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {PERIODS.map((p) => (
                <SelectItem key={p.value} value={p.value}>
                  {p.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      <div className="grid grid-cols-3 gap-3">
        <div className="flex flex-col gap-2">
          <Label htmlFor="board-size">上榜人数</Label>
          <Input
            id="board-size"
            inputMode="numeric"
            className="font-mono tabular-nums"
            value={size}
            onChange={(e) => setSize(e.target.value)}
            disabled={pending}
          />
        </div>
        <div className="flex flex-col gap-2">
          <Label htmlFor="board-min-matches">最少局数</Label>
          <Input
            id="board-min-matches"
            inputMode="numeric"
            className="font-mono tabular-nums"
            value={minMatches}
            onChange={(e) => setMinMatches(e.target.value)}
            disabled={pending}
          />
        </div>
        <div className="flex flex-col gap-2">
          <Label htmlFor="board-sort-order">排序</Label>
          <Input
            id="board-sort-order"
            inputMode="numeric"
            className="font-mono tabular-nums"
            value={sortOrder}
            onChange={(e) => setSortOrder(e.target.value)}
            disabled={pending}
          />
        </div>
      </div>
      <span className="-mt-2 text-xs text-muted-foreground">
        上榜人数为 {MIN_SIZE}-{MAX_SIZE}; 最少局数按同一范围和周期内完成的对局计, 0 为不限; 排序从小到大决定页签顺序
      </span>

      <div className="flex items-center justify-between gap-4 rounded-lg border p-3">
        <div className="flex flex-col gap-0.5">
          <Label htmlFor="board-enabled">启用</Label>
          <span className="text-xs text-muted-foreground">停用后玩家看不到这个榜单</span>
        </div>
        <Switch id="board-enabled" checked={enabled} onCheckedChange={setEnabled} disabled={pending} />
      </div>

      <DialogFooter>
        <Button type="button" variant="outline" onClick={onClose} disabled={pending}>
          取消
        </Button>
        <SubmitButton pending={pending}>{board ? '保存' : '新建'}</SubmitButton>
      </DialogFooter>
    </form>
  );
}
