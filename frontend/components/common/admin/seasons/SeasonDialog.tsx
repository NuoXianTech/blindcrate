'use client';

import {useState} from 'react';
import {toast} from 'sonner';
import {Button} from '@/components/ui/button';
import {Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle} from '@/components/ui/dialog';
import {Input} from '@/components/ui/input';
import {Label} from '@/components/ui/label';
import {SubmitButton} from '@/components/common/form';
import {check, useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';
import type {SeasonRequest} from '@/lib/services/admin';
import {SEASON_STATUS, type Season} from '@/lib/services/season';
import {parseAmount} from '@/lib/utils';
import {CODE_PATTERN, DEFAULT_DAYS, DEFAULT_RATING_CARRY, fromLocalInput, toLocalInput} from './constants';

/**
 * 赛季对话框的Props接口
 */
interface SeasonDialogProps {
  /** null: 关闭; "new": 新建; 赛季: 编辑 */
  season: Season | 'new' | null;
  onClose: () => void;
  /** 保存成功后刷新列表 */
  onSaved: () => void;
}

/**
 * 新建或编辑赛季: 代码创建后不能修改, 开始后不能修改开始时间和段位分继承比例, 结束后只能修改名称
 */
export function SeasonDialog({season, ...props}: SeasonDialogProps) {
  const editing = season && season !== 'new' ? season : null;
  return (
    <Dialog open={!!season} onOpenChange={(open) => !open && props.onClose()}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{editing ? `编辑 ${editing.name}` : '新建赛季'}</DialogTitle>
          <DialogDescription>赛季之间的时间不能重叠; 进行中的赛季会额外统计赛季数据, 赛季排行榜按它排名</DialogDescription>
        </DialogHeader>
        {/* key 保证切换赛季时表单重置 */}
        {season && <SeasonForm key={editing?.id ?? 'new'} season={editing} {...props} />}
      </DialogContent>
    </Dialog>
  );
}

/** 新建赛季时默认的开始时间: 下一个整点 */
function defaultStart(): Date {
  const d = new Date();
  d.setHours(d.getHours() + 1, 0, 0, 0);
  return d;
}

/**
 * 赛季表单, season 为 null 时新建
 */
function SeasonForm({season, onClose, onSaved}: Omit<SeasonDialogProps, 'season'> & { season: Season | null }) {
  const {pending, submit} = useSubmit();
  const [code, setCode] = useState(season?.code ?? '');
  const [name, setName] = useState(season?.name ?? '');
  const [startsAt, setStartsAt] = useState(() => toLocalInput(season?.starts_at ?? defaultStart()));
  const [endsAt, setEndsAt] = useState(() => {
    if (season) return toLocalInput(season.ends_at);
    const end = defaultStart();
    end.setDate(end.getDate() + DEFAULT_DAYS);
    return toLocalInput(end);
  });
  const [carry, setCarry] = useState(String(season?.rating_carry ?? DEFAULT_RATING_CARRY));

  const status = season?.status ?? SEASON_STATUS.upcoming;
  const startLocked = status !== SEASON_STATUS.upcoming;
  const endLocked = status === SEASON_STATUS.ended || status === SEASON_STATUS.settled;

  const handleSubmit = submit(async () => {
    const starts = fromLocalInput(startsAt);
    const ends = fromLocalInput(endsAt);
    check(!starts || !ends ? '请填写开始和结束时间' : null);
    const ratingCarry = parseAmount(carry);
    check(ratingCarry === null || ratingCarry < 0 || ratingCarry > 100 ? '段位分继承比例需为 0-100' : null);
    const body: SeasonRequest = {
      name: name.trim(),
      // 没有修改的时间原样提交, 后端按分钟比较
      starts_at: season && startsAt === toLocalInput(season.starts_at) ? season.starts_at : starts!,
      ends_at: season && endsAt === toLocalInput(season.ends_at) ? season.ends_at : ends!,
      rating_carry: ratingCarry!,
    };
    check(body.name.length === 0 || [...body.name].length > 64 ? '名称需为 1-64 个字符' : null);
    check(new Date(body.ends_at) <= new Date(body.starts_at) ? '结束时间需晚于开始时间' : null);
    if (season) {
      await services.admin.updateSeason(season.id, body);
      toast.success(`已保存 ${body.name}`);
    } else {
      const created = {...body, code: code.trim()};
      check(CODE_PATTERN.test(created.code) ? null : '代码需为 1-24 个小写字母、数字或下划线');
      await services.admin.createSeason(created);
      toast.success(`已新建 ${body.name}`);
    }
    onSaved();
    onClose();
  });

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4" noValidate>
      <div className="grid grid-cols-2 gap-3">
        <div className="flex flex-col gap-2">
          <Label htmlFor="season-code">代码</Label>
          <Input
            id="season-code"
            maxLength={24}
            placeholder="如 s1"
            className="font-mono"
            value={code}
            onChange={(e) => setCode(e.target.value)}
            disabled={pending || !!season}
          />
        </div>
        <div className="flex flex-col gap-2">
          <Label htmlFor="season-name">名称</Label>
          <Input
            id="season-name"
            maxLength={64}
            placeholder="如 第一赛季"
            value={name}
            onChange={(e) => setName(e.target.value)}
            disabled={pending}
          />
        </div>
      </div>
      <span className="-mt-2 text-xs text-muted-foreground">赛季数据按代码统计, 代码创建后不能修改</span>

      <div className="grid grid-cols-2 gap-3">
        <div className="flex flex-col gap-2">
          <Label htmlFor="season-starts-at">开始时间</Label>
          <Input
            id="season-starts-at"
            type="datetime-local"
            value={startsAt}
            onChange={(e) => setStartsAt(e.target.value)}
            disabled={pending || startLocked}
          />
        </div>
        <div className="flex flex-col gap-2">
          <Label htmlFor="season-ends-at">结束时间</Label>
          <Input
            id="season-ends-at"
            type="datetime-local"
            value={endsAt}
            onChange={(e) => setEndsAt(e.target.value)}
            disabled={pending || endLocked}
          />
        </div>
      </div>
      <span className="-mt-2 text-xs text-muted-foreground">
        {endLocked ?
          '赛季已结束, 只能修改名称' :
          startLocked ?
            '赛季已开始, 不能修改开始时间和段位分继承比例; 结束时间需晚于现在, 要立即结束请使用提前结束' :
            '按本地时间, 精确到分钟; 包含开始时间, 不包含结束时间, 结束时间需晚于现在'}
      </span>

      <div className="flex flex-col gap-2">
        <Label htmlFor="season-rating-carry">段位分继承比例 (%)</Label>
        <Input
          id="season-rating-carry"
          inputMode="numeric"
          className="w-32 font-mono tabular-nums"
          value={carry}
          onChange={(e) => setCarry(e.target.value)}
          disabled={pending || startLocked}
        />
        <span className="text-xs text-muted-foreground">
          初始分 = 基础分 + (上赛季的段位分 - 基础分) × 比例; 0 为全部重置, 100 为原样继承, 上赛季没打过排位时为基础分; 赛季开始后不能修改
        </span>
      </div>

      <DialogFooter>
        <Button type="button" variant="outline" onClick={onClose} disabled={pending}>
          取消
        </Button>
        <SubmitButton pending={pending}>{season ? '保存' : '新建'}</SubmitButton>
      </DialogFooter>
    </form>
  );
}
