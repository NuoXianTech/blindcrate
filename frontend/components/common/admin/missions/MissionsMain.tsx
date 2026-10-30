'use client';

import {useCallback, useEffect, useMemo, useState} from 'react';
import {Plus, Search} from 'lucide-react';
import {toast} from 'sonner';
import {Button} from '@/components/ui/button';
import {Input} from '@/components/ui/input';
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select';
import {Table, TableBody, TableHead, TableHeader, TableRow} from '@/components/ui/table';
import {scopeOptions} from '@/components/common/admin/editors';
import {useAuth} from '@/components/common/auth';
import {Pagination, TableEmpty, TableSkeleton} from '@/components/common/layout';
import {useAdminCatalog} from '@/hooks/use-admin-catalog';
import services from '@/lib/services';
import type {AdminVenue, ConditionType} from '@/lib/services/admin';
import {toErrorMessage} from '@/lib/services/core';
import type {AdminMission, ListMissionsData, MissionCycle, MissionOptions} from '@/lib/services/mission';
import {DeleteMissionDialog} from './DeleteMissionDialog';
import {MissionDialog, type MissionFormOptions, type MissionTarget} from './MissionDialog';
import {MissionRow} from './MissionRow';
import {ALL, PAGE_SIZE, STATUS_FILTERS, VENUE_LIMIT, toRequest, type StatusFilter} from './constants';

/**
 * 任务管理: 搜索、按周期和启用状态筛选, 新建、编辑、复制、启停、删除
 */
export function MissionsMain() {
  const {user: me} = useAuth();
  const isAdmin = !!me?.is_admin;
  const [keywordInput, setKeywordInput] = useState('');
  const [keyword, setKeyword] = useState('');
  const [cycle, setCycle] = useState(ALL);
  const [status, setStatus] = useState<StatusFilter>('all');
  const [current, setCurrent] = useState(1);
  const [data, setData] = useState<ListMissionsData | null>(null);
  const [loading, setLoading] = useState(true);

  const catalog = useAdminCatalog();
  const [choices, setChoices] = useState<MissionOptions | null>(null);
  const [types, setTypes] = useState<ConditionType[] | null>(null);
  const [venues, setVenues] = useState<AdminVenue[] | null>(null);

  const [editing, setEditing] = useState<MissionTarget | null>(null);
  const [deleting, setDeleting] = useState<AdminMission | null>(null);

  const load = useCallback(async () => {
    // 确认是管理员后再拉取, 否则 AdminGuard 会跳走
    if (!isAdmin) return;
    setLoading(true);
    try {
      setData(
          await services.missionAdmin.listMissions({
            current,
            size: PAGE_SIZE,
            keyword: keyword || undefined,
            cycle: cycle === ALL ? undefined : cycle as MissionCycle,
            is_enabled: status === 'all' ? undefined : status === 'enabled',
          }),
      );
    } catch (err) {
      toast.error(toErrorMessage(err));
    } finally {
      setLoading(false);
    }
  }, [current, keyword, cycle, status, isAdmin]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 筛选条件变化时重新拉取
    void load();
  }, [load]);

  // 周期、指标、条件类型和地图在列表和编辑时都要用到, 挂载时拉取一次
  useEffect(() => {
    if (!isAdmin) return;
    Promise.all([
      services.missionAdmin.listOptions(),
      services.admin.listConditionTypes(),
      services.admin.listVenues({current: 1, size: VENUE_LIMIT}),
    ])
        .then(([optionData, typeData, venueData]) => {
          setChoices(optionData);
          setTypes(typeData.types);
          setVenues(venueData.venues);
        })
        .catch((err) => toast.error(toErrorMessage(err)));
  }, [isAdmin]);

  const options = useMemo<MissionFormOptions>(() => ({
    items: catalog.items,
    currencies: catalog.currencies,
    cycles: choices?.cycles ?? [],
    metrics: choices?.metrics ?? [],
    types: types ?? [],
    scopes: scopeOptions(venues ?? []),
    loaded: catalog.loaded && choices !== null && types !== null && venues !== null,
  }), [catalog.items, catalog.currencies, catalog.loaded, choices, types, venues]);

  const handleSearch = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setCurrent(1);
    setKeyword(keywordInput.trim());
  };

  // 启停也走修改接口, 需提交全部字段
  const toggleEnabled = async (mission: AdminMission) => {
    try {
      await services.missionAdmin.updateMission(mission.id, {...toRequest(mission), is_enabled: !mission.is_enabled});
      toast.success(mission.is_enabled ? `已停用 ${mission.name}` : `已启用 ${mission.name}`);
      await load();
    } catch (err) {
      toast.error(toErrorMessage(err));
    }
  };

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between gap-2">
        <h1 className="text-2xl font-bold">任务管理</h1>
        <Button onClick={() => setEditing({mode: 'new'})}>
          <Plus />
          新建任务
        </Button>
      </div>

      <div className="flex flex-col gap-2 sm:flex-row">
        <form onSubmit={handleSearch} className="flex flex-1 gap-2">
          <Input
            placeholder="代码或名称"
            value={keywordInput}
            onChange={(e) => setKeywordInput(e.target.value)}
            className="sm:max-w-xs"
          />
          <Button type="submit" variant="secondary">
            <Search />
            搜索
          </Button>
        </form>
        <Select
          value={cycle}
          onValueChange={(value) => {
            setCurrent(1);
            setCycle(value);
          }}
        >
          <SelectTrigger className="sm:w-32" aria-label="周期">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL}>全部周期</SelectItem>
            {options.cycles.map((c) => (
              <SelectItem key={c.code} value={c.code}>
                {c.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select
          value={status}
          onValueChange={(value) => {
            setCurrent(1);
            setStatus(value as StatusFilter);
          }}
        >
          <SelectTrigger className="sm:w-32" aria-label="启用状态">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {STATUS_FILTERS.map((f) => (
              <SelectItem key={f.value} value={f.value}>
                {f.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="rounded-xl border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>任务</TableHead>
              <TableHead>周期</TableHead>
              <TableHead>目标</TableHead>
              <TableHead>奖励</TableHead>
              <TableHead className="text-right">条件</TableHead>
              <TableHead className="text-right">排序</TableHead>
              <TableHead>状态</TableHead>
              <TableHead className="text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading && !data ? (
              <TableSkeleton cols={8} />
            ) : data?.missions.length ? (
              data.missions.map((m) => (
                <MissionRow
                  key={m.id}
                  mission={m}
                  cycleLabel={options.cycles.find((c) => c.code === m.cycle)?.label ?? m.cycle}
                  metricLabel={options.metrics.find((x) => x.key === m.metric)?.label ?? m.metric}
                  scopeLabel={options.scopes.find((s) => s.value === m.scope)?.label ?? m.scope}
                  items={catalog.items}
                  currencies={catalog.currencies}
                  onEdit={() => setEditing({mode: 'edit', mission: m})}
                  onCopy={() => setEditing({mode: 'copy', mission: m})}
                  onToggle={() => toggleEnabled(m)}
                  onDelete={() => setDeleting(m)}
                />
              ))
            ) : (
              <TableEmpty colSpan={8}>没有找到任务</TableEmpty>
            )}
          </TableBody>
        </Table>
      </div>

      <Pagination current={current} size={PAGE_SIZE} total={data?.total ?? 0} onChange={setCurrent} />

      <MissionDialog target={editing} options={options} onClose={() => setEditing(null)} onSaved={load} />
      <DeleteMissionDialog mission={deleting} onClose={() => setDeleting(null)} onSaved={load} />
    </div>
  );
}
