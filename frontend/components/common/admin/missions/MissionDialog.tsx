'use client';

import {useId, useState} from 'react';
import {toast} from 'sonner';
import {Button} from '@/components/ui/button';
import {Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle} from '@/components/ui/dialog';
import {Input} from '@/components/ui/input';
import {Label} from '@/components/ui/label';
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select';
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
import {FieldGroup, InputField, SwitchField} from '@/components/common/admin/venues/Fields';
import {SubmitButton} from '@/components/common/form';
import {check, useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';
import type {AdminItem, ConditionType, Currency, StatMetric} from '@/lib/services/admin';
import type {CodeLabel} from '@/lib/services/core';
import type {AdminMission, MissionCycle, MissionRequest} from '@/lib/services/mission';
import {amountInput, parseAmount} from '@/lib/utils';
import {
  CODE_PATTERN,
  DEFAULT_CYCLE,
  MAX_BUNDLE_LINES,
  MAX_REQUIREMENTS,
  MAX_SORT_ORDER,
  MAX_TARGET,
  metricCurrency,
} from './constants';

/** 编辑任务用到的物品、货币、周期、指标、条件类型和统计范围 */
export interface MissionFormOptions {
  items: AdminItem[];
  currencies: Currency[];
  cycles: CodeLabel[];
  metrics: StatMetric[];
  types: ConditionType[];
  scopes: ScopeOption[];
  loaded: boolean;
}

/** 对话框的目标: new 新建; edit 编辑; copy 以已有任务为模板新建 */
export type MissionTarget = { mode: 'new' } | { mode: 'edit' | 'copy'; mission: AdminMission };

/**
 * 任务对话框的Props接口
 */
interface MissionDialogProps {
  /** null 时关闭 */
  target: MissionTarget | null;
  options: MissionFormOptions;
  onClose: () => void;
  /** 保存成功后刷新列表 */
  onSaved: () => void;
}

/**
 * 新建、编辑或复制任务
 */
export function MissionDialog({target, options, onClose, onSaved}: MissionDialogProps) {
  const title = !target ?
    '' :
    target.mode === 'edit' ? `编辑 ${target.mission.name}` : target.mode === 'copy' ? `复制 ${target.mission.name}` : '新建任务';
  return (
    <Dialog open={!!target} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>进度取自对局统计, 统计指标在当前周期达到目标后可以领取奖励; 修改目标后本周期已领取的不受影响</DialogDescription>
        </DialogHeader>
        {target && !options.loaded && (
          <div className="flex flex-col gap-2">
            <Skeleton className="h-9" />
            <Skeleton className="h-9" />
            <Skeleton className="h-9" />
          </div>
        )}
        {/* key 保证切换任务时表单重置 */}
        {target && options.loaded && (
          <MissionForm
            key={target.mode === 'new' ? 'new' : `${target.mode}:${target.mission.id}`}
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
 * 任务表单
 */
function MissionForm({target, options, onClose, onSaved}: Omit<MissionDialogProps, 'target'> & { target: MissionTarget }) {
  const scopeListID = useId();
  const {pending, submit} = useSubmit();
  // 编辑和复制时以已有任务为初始值, 复制时代码留空
  const source = target.mode === 'new' ? null : target.mission;
  const editing = target.mode === 'edit' ? target.mission : null;
  const {items, currencies, cycles, metrics, types, scopes} = options;

  const [code, setCode] = useState(editing?.code ?? '');
  const [name, setName] = useState(source?.name ?? '');
  const [description, setDescription] = useState(source?.description ?? '');
  const [cycle, setCycle] = useState<string>(source?.cycle ?? DEFAULT_CYCLE);
  const [metric, setMetric] = useState(source?.metric ?? '');
  const [scope, setScope] = useState(source?.scope ?? 'all');
  const [goalText, setGoalText] = useState(() =>
    source ? amountInput(source.target, metricCurrency(source.metric, currencies)?.decimals ?? 0) : '');
  const [sortOrder, setSortOrder] = useState(String(source?.sort_order ?? 0));
  const [isEnabled, setIsEnabled] = useState(source?.is_enabled ?? true);
  const [reward, setReward] = useState(() => toBundleDrafts(source?.reward ?? {}, currencies));
  const [requirements, setRequirements] = useState(() => toConditionDrafts(source?.requirements ?? []));

  const currency = metricCurrency(metric, currencies);
  const unknownMetric = metric && !metrics.some((m) => m.key === metric);
  const unknownCycle = !cycles.some((c) => c.code === cycle);

  const handleSubmit = submit(async () => {
    const goal = parseAmount(goalText.trim(), currency?.decimals ?? 0);
    check(goal === null || goal < 1 || goal > MAX_TARGET ? '目标需为正数' : null);
    const sort = parseAmount(sortOrder.trim());
    check(sort === null || Math.abs(sort) > MAX_SORT_ORDER ? `排序需为 -${MAX_SORT_ORDER} 到 ${MAX_SORT_ORDER} 的整数` : null);
    const body: MissionRequest = {
      code: editing?.code ?? code.trim(),
      name: name.trim(),
      description: description.trim(),
      cycle: cycle as MissionCycle,
      scope: scope.trim(),
      metric,
      target: goal!,
      reward: fromBundleDrafts(reward, currencies, '奖励'),
      requirements: fromConditionDrafts(requirements, types, '任务条件'),
      sort_order: sort!,
      is_enabled: isEnabled,
    };
    check(CODE_PATTERN.test(body.code) ? null : '代码需为 1-32 个小写字母、数字或下划线');
    check(body.name.length === 0 || [...body.name].length > 64 ? '名称需为 1-64 个字符' : null);
    check(!body.metric ? '请选择统计指标' : null);
    check(!body.scope ? '请填写统计范围' : null);
    check(reward.length === 0 ? '奖励不能为空' : null);
    if (editing) {
      await services.missionAdmin.updateMission(editing.id, body);
      toast.success(`已保存 ${body.name}`);
    } else {
      await services.missionAdmin.createMission(body);
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
            id="mission-code"
            label="代码"
            placeholder="如 daily_matches"
            maxLength={32}
            value={code}
            onChange={setCode}
            help="领取记录按任务计, 创建后不能修改"
            disabled={pending || !!editing}
          />
          <InputField id="mission-name" label="名称" placeholder="如 每日对局" maxLength={64} value={name} onChange={setName} disabled={pending} />
        </div>
        <div className="flex flex-col gap-2">
          <Label htmlFor="mission-description">描述</Label>
          <Textarea
            id="mission-description"
            maxLength={255}
            placeholder="选填, 最多 255 个字符"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            disabled={pending}
          />
        </div>
        <div className="grid grid-cols-2 gap-3">
          <div className="flex flex-col gap-2">
            <Label htmlFor="mission-cycle">周期</Label>
            <Select value={cycle} onValueChange={setCycle} disabled={pending || !!editing}>
              <SelectTrigger id="mission-cycle" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {unknownCycle && <SelectItem value={cycle}>{cycle}</SelectItem>}
                {cycles.map((c) => (
                  <SelectItem key={c.code} value={c.code}>
                    {c.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <span className="text-xs text-muted-foreground">每日、每周按服务器时区重置, 每周从周一开始; 成就只能领取一次; 创建后不能修改</span>
          </div>
          <InputField
            id="mission-sort"
            label="排序"
            inputMode="numeric"
            value={sortOrder}
            onChange={setSortOrder}
            help="同一周期内从小到大排列"
            disabled={pending}
          />
        </div>
        <SwitchField
          id="mission-enabled"
          label="启用"
          help="停用后玩家看不到, 也不能领取"
          checked={isEnabled}
          onChange={setIsEnabled}
          disabled={pending}
        />
      </FieldGroup>

      <FieldGroup title="目标" help="统计范围内的指标在当前周期达到目标即完成; 净收益按货币输入, 可以有小数">
        <div className="flex flex-col gap-2">
          <Label htmlFor="mission-metric">统计指标</Label>
          <Select value={metric} onValueChange={setMetric} disabled={pending}>
            <SelectTrigger id="mission-metric" className="w-full">
              <SelectValue placeholder="选择统计指标" />
            </SelectTrigger>
            <SelectContent>
              {unknownMetric && <SelectItem value={metric}>{`未知指标 ${metric}`}</SelectItem>}
              {metrics.map((m) => (
                <SelectItem key={m.key} value={m.key}>
                  {m.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="grid grid-cols-2 gap-3">
          <div className="flex flex-col gap-2">
            <Label htmlFor="mission-scope">统计范围</Label>
            <Input
              id="mission-scope"
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
          <InputField
            id="mission-target"
            label="目标"
            inputMode="decimal"
            placeholder="如 3"
            suffix={currency?.name}
            value={goalText}
            onChange={setGoalText}
            disabled={pending}
          />
        </div>
      </FieldGroup>

      <FieldGroup title="奖励" help={`领取时发放的货币和物品, 不能为空; 已下架的物品按回收价折成货币; 最多 ${MAX_BUNDLE_LINES} 项`}>
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

      <FieldGroup title="任务条件" help={`领取时需要全部满足, 不满足时玩家能看到任务但不能领取; 最多 ${MAX_REQUIREMENTS} 个`}>
        <ConditionEditor
          drafts={requirements}
          onChange={setRequirements}
          types={types}
          items={items}
          currencies={currencies}
          scopes={scopes}
          max={MAX_REQUIREMENTS}
          empty="没有任务条件, 所有玩家都能领取"
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
