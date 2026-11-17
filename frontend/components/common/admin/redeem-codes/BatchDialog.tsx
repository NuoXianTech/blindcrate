'use client';

import {useState} from 'react';
import {cn} from 'cn';
import {nanoid} from 'nanoid';
import {useForm, useWatch, type Control} from 'react-hook-form';
import {toast} from 'sonner';
import {Button} from '@/components/ui/button';
import {Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle} from '@/components/ui/dialog';
import {Form, FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage} from '@/components/ui/form';
import {Input} from '@/components/ui/input';
import {Skeleton} from '@/components/ui/skeleton';
import {Switch} from '@/components/ui/switch';
import {
  BundleEditor,
  ConditionEditor,
  fromBundleDrafts,
  fromConditionDrafts,
  toBundleDrafts,
  toConditionDrafts,
  type BundleDraft,
  type ConditionDraft,
} from '@/components/common/admin/editors';
import {FieldGroup} from '@/components/common/admin/venues/Fields';
import {SubmitButton} from '@/components/common/form';
import {useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';
import type {ConditionType} from '@/lib/services/condition';
import type {AdminItem} from '@/lib/services/item';
import type {RedeemBatch, RedeemBatchRequest} from '@/lib/services/redeem';
import type {Currency} from '@/lib/services/wallet';
import {fromLocalInput, parseAmount, toLocalInput} from '@/lib/utils';
import {
  MAX_BATCH_CODES,
  MAX_CODE_INPUT,
  MAX_CODE_USES,
  MAX_NAME,
  MAX_REQUIREMENTS,
  MAX_REWARD_LINES,
  MAX_USER_LIMIT,
  checkCode,
} from './constants';

/** 编辑批次用到的物品、货币和条件类型 */
export interface BatchOptions {
  items: AdminItem[];
  currencies: Currency[];
  types: ConditionType[];
  loaded: boolean;
}

/** 对话框的目标: new 生成; edit 修改批次的设置; copy 以已有批次为模板再生成一批 */
export type BatchTarget = { mode: 'new' } | { mode: 'edit' | 'copy'; batch: RedeemBatch };

/** 生成方式: random 随机生成多个, custom 指定一个兑换码 */
type CodeMode = 'random' | 'custom';

const CODE_MODES: { value: CodeMode; label: string; description: string }[] = [
  {value: 'random', label: '随机生成', description: '生成多个 12 位的兑换码, 适合每人发一个'},
  {value: 'custom', label: '指定兑换码', description: '只生成一个自己定的兑换码, 适合公开发给很多人'},
];

interface BatchValues {
  name: string;
  /** 生成方式、个数和指定的兑换码只在生成时有 */
  mode: CodeMode;
  count: string;
  code: string;
  maxUses: string;
  userLimit: string;
  /** datetime-local 输入框的值, 留空为不过期 */
  expiresAt: string;
  isEnabled: boolean;
  reward: BundleDraft[];
  requirements: ConditionDraft[];
}

/**
 * 批次对话框的Props接口
 */
interface BatchDialogProps {
  /** null 时关闭 */
  target: BatchTarget | null;
  options: BatchOptions;
  onClose: () => void;
  /** 保存成功后刷新列表; 生成了新的批次时带上它, 用于展示兑换码 */
  onSaved: (created?: RedeemBatch) => void;
}

/**
 * 生成兑换码, 或修改批次的设置; 兑换码生成后不能修改
 */
export function BatchDialog({target, options, onClose, onSaved}: BatchDialogProps) {
  const title = !target ? '' : target.mode === 'edit' ? `编辑 ${target.batch.name}` : target.mode === 'copy' ? `以 ${target.batch.name} 为模板生成` : '生成兑换码';
  return (
    <Dialog open={!!target} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>
            {target?.mode === 'edit' ?
              '兑换码生成后不能修改; 其他设置修改后只影响之后的兑换, 已兑换的奖励不受影响' :
              '一次生成的兑换码为一个批次, 共用下面的设置; 生成后兑换码不能修改, 其他设置可以随时修改'}
          </DialogDescription>
        </DialogHeader>
        {target && !options.loaded && (
          <div className="flex flex-col gap-2">
            <Skeleton className="h-9" />
            <Skeleton className="h-9" />
            <Skeleton className="h-9" />
          </div>
        )}
        {/* key 保证切换批次时表单重置 */}
        {target && options.loaded && (
          <BatchForm
            key={target.mode === 'new' ? 'new' : `${target.mode}:${target.batch.id}`}
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
 * 批次表单
 * 幂等键与提交的内容对应: 超时后重试同一份内容只生成一次, 修改内容后视为新的一次生成
 */
function BatchForm({target, options, onClose, onSaved}: Omit<BatchDialogProps, 'target'> & { target: BatchTarget }) {
  const {pending, submit} = useSubmit();
  // 编辑和复制时以已有批次为初始值
  const source = target.mode === 'new' ? null : target.batch;
  const editing = target.mode === 'edit' ? target.batch : null;
  const {items, currencies, types} = options;
  const [attempt, setAttempt] = useState<{ content: string; key: string } | null>(null);

  const form = useForm<BatchValues>({
    defaultValues: {
      name: source?.name ?? '',
      mode: 'random',
      count: '1',
      code: '',
      maxUses: String(source?.max_uses ?? 1),
      userLimit: String(source?.user_limit ?? 1),
      expiresAt: source?.expires_at ? toLocalInput(source.expires_at) : '',
      isEnabled: source?.is_enabled ?? true,
      reward: toBundleDrafts(source?.reward ?? {}, currencies),
      requirements: toConditionDrafts(source?.requirements ?? [], types, currencies),
    },
  });
  const mode = useWatch({control: form.control, name: 'mode'});

  // 字段校验由表单完成, 错误显示在字段下方; 奖励和使用条件每一行的校验、请求失败以提示框展示
  const handleSubmit = form.handleSubmit((values) => submit(async () => {
    // 没有修改的到期时间原样提交; 留空为不过期
    const unchanged = editing?.expires_at && values.expiresAt === toLocalInput(editing.expires_at);
    const settings: RedeemBatchRequest = {
      name: values.name.trim(),
      reward: fromBundleDrafts(values.reward, currencies, '奖励'),
      requirements: fromConditionDrafts(values.requirements, types, '使用条件', currencies),
      max_uses: parseAmount(values.maxUses.trim()) ?? 0,
      user_limit: parseAmount(values.userLimit.trim()) ?? 0,
      expires_at: unchanged ? editing.expires_at : values.expiresAt ? fromLocalInput(values.expiresAt) : null,
      is_enabled: values.isEnabled,
    };
    if (editing) {
      await services.redeemAdmin.updateBatch(editing.id, settings);
      toast.success(`已保存 ${settings.name}`);
      onSaved();
      onClose();
      return;
    }
    const custom = values.mode === 'custom';
    const body = {...settings, count: custom ? 1 : parseAmount(values.count.trim()) ?? 0, code: custom ? values.code.trim() : ''};
    const content = JSON.stringify(body);
    const key = attempt?.content === content ? attempt.key : nanoid();
    setAttempt({content, key});
    const created = await services.redeemAdmin.createBatch({...body, idempotency_key: key});
    toast.success(`已生成 ${created.code_count} 个兑换码`);
    onSaved(created);
    onClose();
  })());

  return (
    <Form {...form}>
      <form onSubmit={handleSubmit} className="flex flex-col gap-6" noValidate>
        <FieldGroup title="基本信息">
          <FormField
            control={form.control}
            name="name"
            rules={{
              validate: (value) => {
                const length = [...value.trim()].length;
                return length >= 1 && length <= MAX_NAME ? true : `批次名称需为 1-${MAX_NAME} 个字符`;
              },
            }}
            render={({field}) => (
              <FormItem>
                <FormLabel>批次名称</FormLabel>
                <FormControl>
                  <Input maxLength={MAX_NAME} placeholder="如 开服礼包" disabled={pending} {...field} />
                </FormControl>
                <FormDescription className="text-xs">兑换成功后展示给玩家, 也是钱包和背包流水的备注</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name="isEnabled"
            render={({field}) => (
              <FormItem className="flex items-center justify-between gap-4 rounded-lg border p-3">
                <div className="flex flex-col gap-0.5">
                  <FormLabel>启用</FormLabel>
                  <FormDescription className="text-xs">停用后这个批次的兑换码都不能兑换, 可以随时再启用</FormDescription>
                </div>
                <FormControl>
                  <Switch checked={field.value} onCheckedChange={field.onChange} disabled={pending} />
                </FormControl>
              </FormItem>
            )}
          />
        </FieldGroup>

        {!editing && (
          <FieldGroup title="兑换码" help="兑换时不区分大小写, 其中的空格和连字符会被忽略">
            <FormField
              control={form.control}
              name="mode"
              render={({field}) => (
                <div role="group" aria-label="生成方式" className="grid gap-2 sm:grid-cols-2">
                  {CODE_MODES.map((m) => (
                    <button
                      key={m.value}
                      type="button"
                      className={cn(
                          'flex flex-col gap-0.5 rounded-lg border p-2.5 text-left text-sm transition-colors',
                          field.value === m.value ? 'border-primary bg-primary/5' : 'hover:bg-accent',
                          pending && 'cursor-not-allowed opacity-50',
                      )}
                      aria-pressed={field.value === m.value}
                      onClick={() => field.onChange(m.value)}
                      disabled={pending}
                    >
                      <span className="font-medium">{m.label}</span>
                      <span className="text-xs text-muted-foreground">{m.description}</span>
                    </button>
                  ))}
                </div>
              )}
            />
            {mode === 'random' ? (
              <NumberField
                control={form.control}
                name="count"
                label="生成数量"
                suffix="个"
                help={`一次最多生成 ${MAX_BATCH_CODES} 个`}
                min={1}
                max={MAX_BATCH_CODES}
                disabled={pending}
              />
            ) : (
              <FormField
                control={form.control}
                name="code"
                rules={{validate: (value) => checkCode(value) ?? true}}
                render={({field}) => (
                  <FormItem>
                    <FormLabel>兑换码</FormLabel>
                    <FormControl>
                      <Input
                        autoComplete="off"
                        spellCheck={false}
                        maxLength={MAX_CODE_INPUT}
                        placeholder="如 WELCOME2026"
                        className="font-mono"
                        disabled={pending}
                        {...field}
                      />
                    </FormControl>
                    <FormDescription className="text-xs">4-32 个字母、数字或汉字, 字母保存为大写; 不能与已有的兑换码重复, 包括已删除的批次中的</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            )}
          </FieldGroup>
        )}

        <FieldGroup title="次数和有效期">
          <div className="grid gap-3 sm:grid-cols-2">
            <NumberField
              control={form.control}
              name="maxUses"
              label="每个兑换码可使用次数"
              suffix="次"
              help="可以被多少名玩家各兑换一次; 1 为单次使用, 0 为不限"
              max={MAX_CODE_USES}
              disabled={pending}
            />
            <NumberField
              control={form.control}
              name="userLimit"
              label="每人可兑换个数"
              suffix="个"
              help="同一名玩家最多兑换这个批次中的几个兑换码; 0 为不限"
              max={MAX_USER_LIMIT}
              disabled={pending}
            />
            <FormField
              control={form.control}
              name="expiresAt"
              rules={{
                validate: (value) => {
                  if (!value) return true;
                  const time = fromLocalInput(value);
                  if (!time) return '到期时间格式不正确';
                  // 修改时可以改到过去, 批次立即到期
                  return editing || new Date(time).getTime() > Date.now() ? true : '到期时间需晚于当前时间';
                },
              }}
              render={({field}) => (
                <FormItem>
                  <FormLabel>到期时间</FormLabel>
                  <FormControl>
                    <Input type="datetime-local" disabled={pending} {...field} />
                  </FormControl>
                  <FormDescription className="text-xs">按本地时间, 精确到分钟, 到这个时间就不能再兑换; 留空为不过期</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
        </FieldGroup>

        <FieldGroup title="奖励" help={`兑换一次得到的货币和物品, 不能为空; 已下架的物品按回收价折成货币发放; 最多 ${MAX_REWARD_LINES} 项`}>
          <FormField
            control={form.control}
            name="reward"
            rules={{validate: (value) => (value.length ? true : '奖励不能为空')}}
            render={({field}) => (
              <FormItem>
                <BundleEditor
                  drafts={field.value}
                  onChange={field.onChange}
                  items={items}
                  currencies={currencies}
                  max={MAX_REWARD_LINES}
                  empty="还没有奖励"
                  disabled={pending}
                />
                <FormMessage />
              </FormItem>
            )}
          />
        </FieldGroup>

        <FieldGroup title="使用条件" help={`需要全部满足才能兑换, 兑换时检查; 最多 ${MAX_REQUIREMENTS} 个`}>
          <FormField
            control={form.control}
            name="requirements"
            render={({field}) => (
              <FormItem>
                <ConditionEditor
                  drafts={field.value}
                  onChange={field.onChange}
                  types={types}
                  items={items}
                  currencies={currencies}
                  max={MAX_REQUIREMENTS}
                  empty="没有使用条件, 所有玩家都能兑换"
                  disabled={pending}
                />
              </FormItem>
            )}
          />
        </FieldGroup>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose} disabled={pending}>
            取消
          </Button>
          <SubmitButton pending={pending}>{editing ? '保存' : '生成'}</SubmitButton>
        </DialogFooter>
      </form>
    </Form>
  );
}

/**
 * 整数输入框, 带单位和说明; 不是范围内的整数时在下方提示
 */
function NumberField({control, name, label, suffix, help, min = 0, max, disabled}: {
  control: Control<BatchValues>;
  name: 'count' | 'maxUses' | 'userLimit';
  label: string;
  suffix: string;
  help: string;
  min?: number;
  max: number;
  disabled: boolean;
}) {
  return (
    <FormField
      control={control}
      name={name}
      rules={{
        validate: (value) => {
          const number = parseAmount(value.trim());
          return number === null || number < min || number > max ? `${label}需为 ${min}-${max} 的整数` : true;
        },
      }}
      render={({field}) => (
        <FormItem>
          <FormLabel>{label}</FormLabel>
          <div className="relative">
            <FormControl>
              <Input inputMode="numeric" className="pr-12 font-mono tabular-nums" disabled={disabled} {...field} />
            </FormControl>
            <span className="pointer-events-none absolute inset-y-0 right-3 flex items-center text-sm text-muted-foreground">
              {suffix}
            </span>
          </div>
          <FormDescription className="text-xs">{help}</FormDescription>
          <FormMessage />
        </FormItem>
      )}
    />
  );
}
