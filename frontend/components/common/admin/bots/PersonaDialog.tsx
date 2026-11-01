'use client';

import {useState} from 'react';
import {RotateCcw} from 'lucide-react';
import {toast} from 'sonner';
import {Button} from '@/components/ui/button';
import {Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle} from '@/components/ui/dialog';
import {Label} from '@/components/ui/label';
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select';
import {Skeleton} from '@/components/ui/skeleton';
import {Textarea} from '@/components/ui/textarea';
import {fromParamsDraft, ParamField, toParamsDraft, type ParamsDraft} from '@/components/common/admin/editors';
import {FieldGroup, InputField} from '@/components/common/admin/venues/Fields';
import {SubmitButton} from '@/components/common/form';
import {check, useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';
import type {BotPersona, BotPersonaRequest, BotStrategy} from '@/lib/services/bot';
import {CODE_PATTERN, MAX_DESCRIPTION_LENGTH, MAX_PERSONA_NAME_LENGTH} from './constants';

/** 对话框的目标: new 新建; edit 编辑; copy 以已有风格为模板新建 */
export type PersonaTarget = { mode: 'new' } | { mode: 'edit' | 'copy'; persona: BotPersona };

/**
 * 风格对话框的Props接口
 */
interface PersonaDialogProps {
  /** null 时关闭 */
  target: PersonaTarget | null;
  /** 决策策略, 加载完成前为 null */
  strategies: BotStrategy[] | null;
  onClose: () => void;
  /** 保存成功后刷新列表 */
  onSaved: () => void;
}

/**
 * 新建、编辑或复制人机风格
 */
export function PersonaDialog({target, strategies, onClose, onSaved}: PersonaDialogProps) {
  const title = !target ?
    '' :
    target.mode === 'edit' ? `编辑 ${target.persona.name}` : target.mode === 'copy' ? `复制 ${target.persona.name}` : '新建风格';
  return (
    <Dialog open={!!target} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>开局时为每个人机快照风格, 修改不影响进行中的对局</DialogDescription>
        </DialogHeader>
        {target && !strategies && (
          <div className="flex flex-col gap-2">
            <Skeleton className="h-9" />
            <Skeleton className="h-9" />
            <Skeleton className="h-9" />
          </div>
        )}
        {/* key 保证切换风格时表单重置 */}
        {target && strategies && (
          <PersonaForm
            key={target.mode === 'new' ? 'new' : `${target.mode}:${target.persona.id}`}
            target={target}
            strategies={strategies}
            onClose={onClose}
            onSaved={onSaved}
          />
        )}
      </DialogContent>
    </Dialog>
  );
}

/** 策略的参数转为表单, 缺少的参数(如策略之后新增的参数)取默认值, 与后端一致 */
function paramsDraftOf(strategy: BotStrategy | undefined, params?: BotPersona['params']): ParamsDraft {
  return toParamsDraft(strategy?.fields ?? [], {...strategy?.defaults, ...params});
}

/**
 * 风格表单
 */
function PersonaForm({target, strategies, onClose, onSaved}: Omit<PersonaDialogProps, 'target' | 'strategies'> & {
  target: PersonaTarget;
  strategies: BotStrategy[];
}) {
  const {pending, submit} = useSubmit();
  // 编辑和复制时以已有风格为初始值, 复制时代码留空
  const source = target.mode === 'new' ? null : target.persona;
  const editing = target.mode === 'edit' ? target.persona : null;

  const [code, setCode] = useState(editing?.code ?? '');
  const [name, setName] = useState(source?.name ?? '');
  const [description, setDescription] = useState(source?.description ?? '');
  const [strategyCode, setStrategyCode] = useState(source?.strategy ?? strategies[0]?.code ?? '');
  const strategy = strategies.find((s) => s.code === strategyCode);
  const [params, setParams] = useState(() => paramsDraftOf(strategy, source?.params));

  // 换策略时参数结构不同, 换成新策略的默认参数
  const changeStrategy = (next: string) => {
    setStrategyCode(next);
    setParams(paramsDraftOf(strategies.find((s) => s.code === next)));
  };

  const handleSubmit = submit(async () => {
    const body: BotPersonaRequest = {
      code: editing?.code ?? code.trim(),
      name: name.trim(),
      description: description.trim(),
      strategy: strategyCode,
      params: {},
    };
    check(CODE_PATTERN.test(body.code) ? null : '代码需为 1-32 个小写字母、数字或下划线');
    check(body.name.length === 0 || [...body.name].length > MAX_PERSONA_NAME_LENGTH ?
      `名称需为 1-${MAX_PERSONA_NAME_LENGTH} 个字符` :
      null);
    check([...body.description].length > MAX_DESCRIPTION_LENGTH ? `说明最多 ${MAX_DESCRIPTION_LENGTH} 个字符` : null);
    check(!strategy ? '请选择决策策略' : null);
    body.params = fromParamsDraft(strategy?.fields ?? [], params, '策略参数');
    if (editing) {
      await services.botAdmin.updatePersona(editing.id, body);
      toast.success(`已保存 ${body.name}`);
    } else {
      await services.botAdmin.createPersona(body);
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
            id="bot-persona-code"
            label="代码"
            placeholder="如 cautious"
            maxLength={32}
            value={code}
            onChange={setCode}
            help="地图按代码引用, 创建后不能修改"
            disabled={pending || !!editing}
          />
          <InputField
            id="bot-persona-name"
            label="名称"
            placeholder="如 谨慎"
            maxLength={MAX_PERSONA_NAME_LENGTH}
            value={name}
            onChange={setName}
            help="只在后台展示, 玩家看不到"
            disabled={pending}
          />
        </div>
        <div className="flex flex-col gap-2">
          <Label htmlFor="bot-persona-description">说明</Label>
          <Textarea
            id="bot-persona-description"
            maxLength={MAX_DESCRIPTION_LENGTH}
            placeholder={`选填, 最多 ${MAX_DESCRIPTION_LENGTH} 个字符`}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            disabled={pending}
          />
        </div>
      </FieldGroup>

      <FieldGroup title="决策策略" help={strategy?.description ?? '策略由玩法模块注册, 决定人机每轮怎么出价'}>
        <Select value={strategyCode} onValueChange={changeStrategy} disabled={pending}>
          <SelectTrigger className="w-full" aria-label="决策策略">
            <SelectValue placeholder="选择决策策略" />
          </SelectTrigger>
          <SelectContent>
            {strategyCode && !strategy && <SelectItem value={strategyCode}>未知策略 {strategyCode}</SelectItem>}
            {strategies.map((s) => (
              <SelectItem key={s.code} value={s.code}>
                {s.label}
                <span className="font-mono text-xs text-muted-foreground">{s.code}</span>
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        {strategy && strategy.fields.length > 0 && (
          <div className="grid gap-3 sm:grid-cols-2">
            {strategy.fields.map((field) => (
              <ParamField
                key={field.key}
                id={`bot-persona-${field.key}`}
                field={field}
                value={params[field.key]}
                onChange={(value) => setParams((prev) => ({...prev, [field.key]: value}))}
                items={[]}
                currencies={[]}
                disabled={pending}
              />
            ))}
          </div>
        )}
        {strategy && (
          <div>
            <Button type="button" variant="outline" size="sm" onClick={() => setParams(paramsDraftOf(strategy))} disabled={pending}>
              <RotateCcw />
              恢复默认参数
            </Button>
          </div>
        )}
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
