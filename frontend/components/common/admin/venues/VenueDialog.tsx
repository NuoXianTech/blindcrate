'use client';

import {useState} from 'react';
import {FlaskConical, LoaderCircle} from 'lucide-react';
import {toast} from 'sonner';
import {Button} from '@/components/ui/button';
import {Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle} from '@/components/ui/dialog';
import {Skeleton} from '@/components/ui/skeleton';
import {Tabs, TabsContent, TabsList, TabsTrigger} from '@/components/ui/tabs';
import type {ScopeOption} from '@/components/common/admin/editors';
import {SubmitButton} from '@/components/common/form';
import {useSite} from '@/components/common/site';
import {useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';
import type {AdminItem, AdminVenue, ConditionType, Currency, LootPoolListItem, Simulation} from '@/lib/services/admin';
import {AuctionTab} from './AuctionTab';
import {BasicTab} from './BasicTab';
import {BotTab} from './BotTab';
import {EntryTab} from './EntryTab';
import {SimulationStats} from './SimulationStats';
import {WarehouseTab} from './WarehouseTab';
import {toRequest} from './constants';
import {toVenueForm, toVenueRequest, type VenueForm} from './form';

/** 编辑地图用到的选项, 由列表页拉取一次 */
export interface VenueOptions {
  items: AdminItem[];
  currencies: Currency[];
  pools: LootPoolListItem[];
  types: ConditionType[];
  /** 入场条件中统计范围的候选项 */
  scopes: ScopeOption[];
  loaded: boolean;
}

/**
 * 地图对话框的Props接口
 */
interface VenueDialogProps {
  /** null: 关闭; "new": 新建; 地图: 编辑 */
  venue: AdminVenue | 'new' | null;
  options: VenueOptions;
  onClose: () => void;
  /** 保存成功后刷新列表 */
  onSaved: () => void;
}

/**
 * 新建或编辑地图, 保存前可以先模拟仓库的价值分布
 */
export function VenueDialog({venue, options, onClose, onSaved}: VenueDialogProps) {
  const editing = venue && venue !== 'new' ? venue : null;
  return (
    <Dialog open={!!venue} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle>{editing ? `编辑 ${editing.name}` : '新建地图'}</DialogTitle>
          <DialogDescription>
            保存时会模拟 10000 个仓库, 摆放失败率超过 1% 时不能保存; 开局时快照配置, 修改不影响进行中的对局
          </DialogDescription>
        </DialogHeader>
        {venue && !options.loaded && (
          <div className="flex flex-col gap-2">
            <Skeleton className="h-9" />
            <Skeleton className="h-9" />
            <Skeleton className="h-9" />
          </div>
        )}
        {/* key 保证切换地图时表单重置 */}
        {venue && options.loaded && (
          <VenueEditor key={editing?.id ?? 'new'} venue={editing} options={options} onClose={onClose} onSaved={onSaved} />
        )}
      </DialogContent>
    </Dialog>
  );
}

/**
 * 地图表单, venue 为 null 时新建
 */
function VenueEditor({venue, options, onClose, onSaved}: Omit<VenueDialogProps, 'venue'> & { venue: AdminVenue | null }) {
  const {items, currencies, pools, types, scopes} = options;
  const save = useSubmit();
  const simulate = useSubmit();
  const pending = save.pending || simulate.pending;
  const {economy} = useSite();
  const [form, setForm] = useState<VenueForm>(() => toVenueForm(venue ? toRequest(venue) : null, currencies, economy.base_currency));
  const [tab, setTab] = useState('basic');
  const [simulation, setSimulation] = useState<Simulation | null>(venue?.simulation ?? null);
  // 模拟结果之后表单是否有改动
  const [stale, setStale] = useState(false);
  const currency = currencies.find((c) => c.code === form.bidCurrency);

  const onChange = (patch: Partial<VenueForm>) => {
    setForm((prev) => ({...prev, ...patch}));
    setStale(true);
  };
  const tabProps = {form, onChange, disabled: pending};

  const handleSimulate = simulate.submit(async () => {
    const body = toVenueRequest(form, currencies, types);
    setTab('simulation');
    setSimulation(await services.admin.simulateVenue(body));
    setStale(false);
  });

  const handleSubmit = save.submit(async () => {
    const body = toVenueRequest(form, currencies, types);
    if (venue) {
      await services.admin.updateVenue(venue.id, body);
      toast.success(`已保存 ${body.name}`);
    } else {
      await services.admin.createVenue(body);
      toast.success(`已新建 ${body.name}`);
    }
    onSaved();
    onClose();
  });

  return (
    <form onSubmit={handleSubmit} className="flex min-w-0 flex-col gap-4" noValidate>
      <Tabs value={tab} onValueChange={setTab} className="min-w-0">
        <TabsList className="w-full">
          <TabsTrigger value="basic">基本</TabsTrigger>
          <TabsTrigger value="entry">入场</TabsTrigger>
          <TabsTrigger value="warehouse">仓库</TabsTrigger>
          <TabsTrigger value="auction">拍卖</TabsTrigger>
          <TabsTrigger value="bots">人机</TabsTrigger>
          <TabsTrigger value="simulation">模拟</TabsTrigger>
        </TabsList>
        <div className="max-h-[60vh] overflow-y-auto px-1 py-2">
          <TabsContent value="basic">
            <BasicTab {...tabProps} editing={!!venue} currencies={currencies} />
          </TabsContent>
          <TabsContent value="entry">
            <EntryTab {...tabProps} items={items} currencies={currencies} types={types} scopes={scopes} />
          </TabsContent>
          <TabsContent value="warehouse">
            <WarehouseTab {...tabProps} pools={pools} />
          </TabsContent>
          <TabsContent value="auction">
            <AuctionTab {...tabProps} currency={currency} />
          </TabsContent>
          <TabsContent value="bots">
            <BotTab {...tabProps} currency={currency} />
          </TabsContent>
          <TabsContent value="simulation" className="flex flex-col gap-3">
            {simulate.pending && (
              <p className="flex items-center gap-2 text-sm text-muted-foreground">
                <LoaderCircle className="size-4 animate-spin" />
                正在模拟
              </p>
            )}
            {simulation ? (
              <SimulationStats simulation={simulation} currency={currency} stale={stale} />
            ) : (
              !simulate.pending && <p className="py-6 text-center text-sm text-muted-foreground">还没有模拟过, 点击下方的模拟按钮</p>
            )}
          </TabsContent>
        </div>
      </Tabs>

      <DialogFooter className="sm:justify-between">
        <Button type="button" variant="secondary" onClick={handleSimulate} disabled={pending}>
          {simulate.pending ? <LoaderCircle className="animate-spin" /> : <FlaskConical />}
          模拟
        </Button>
        <div className="flex flex-col-reverse gap-2 sm:flex-row">
          <Button type="button" variant="outline" onClick={onClose} disabled={pending}>
            取消
          </Button>
          <SubmitButton pending={save.pending} disabled={simulate.pending}>
            {venue ? '保存' : '新建'}
          </SubmitButton>
        </div>
      </DialogFooter>
    </form>
  );
}
