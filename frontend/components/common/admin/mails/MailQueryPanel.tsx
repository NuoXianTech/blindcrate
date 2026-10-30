'use client';

import {useCallback, useEffect, useState} from 'react';
import {Search} from 'lucide-react';
import {toast} from 'sonner';
import {Badge} from '@/components/ui/badge';
import {Button} from '@/components/ui/button';
import {Input} from '@/components/ui/input';
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select';
import {Table, TableBody, TableCell, TableHead, TableHeader, TableRow} from '@/components/ui/table';
import {bundleText} from '@/components/common/admin/editors';
import {Pagination, TableEmpty, TableSkeleton} from '@/components/common/layout';
import services from '@/lib/services';
import type {AdminItem, Currency} from '@/lib/services/admin';
import {toErrorMessage, type CodeLabel} from '@/lib/services/core';
import type {ListAdminMailsData} from '@/lib/services/mailbox';
import {formatDateTime} from '@/lib/utils';
import {ALL, PAGE_SIZE, mailStatus} from './constants';

/**
 * 邮件查询: 按用户 ID、来源和标题筛选, 包括已删除的邮件
 */
export function MailQueryPanel({items, currencies, sources}: {
  items: AdminItem[];
  currencies: Currency[];
  sources: CodeLabel[];
}) {
  const [userInput, setUserInput] = useState('');
  const [keywordInput, setKeywordInput] = useState('');
  const [filters, setFilters] = useState<{ userID?: number; keyword: string }>({keyword: ''});
  const [source, setSource] = useState(ALL);
  const [current, setCurrent] = useState(1);
  const [data, setData] = useState<ListAdminMailsData | null>(null);
  // 列表加载的时间, 用于判断是否已过期
  const [loadedAt, setLoadedAt] = useState(0);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      setData(
          await services.mailboxAdmin.listMails({
            current,
            size: PAGE_SIZE,
            user_id: filters.userID,
            source: source === ALL ? undefined : source,
            keyword: filters.keyword || undefined,
          }),
      );
      setLoadedAt(Date.now());
    } catch (err) {
      toast.error(toErrorMessage(err));
    } finally {
      setLoading(false);
    }
  }, [current, filters, source]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 筛选条件变化时重新拉取
    void load();
  }, [load]);

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    const text = userInput.trim();
    const userID = Number(text);
    if (text && !(Number.isSafeInteger(userID) && userID > 0)) {
      toast.error('用户 ID 需为正整数');
      return;
    }
    setCurrent(1);
    setFilters({userID: text ? userID : undefined, keyword: keywordInput.trim()});
  };

  const sourceLabel = (code: string) => sources.find((s) => s.code === code)?.label ?? code;

  return (
    <div className="flex flex-col gap-4">
      <form onSubmit={handleSearch} className="flex flex-wrap gap-2">
        <Input className="w-32 font-mono" inputMode="numeric" placeholder="用户 ID" value={userInput} onChange={(e) => setUserInput(e.target.value)} />
        <Input className="w-56" placeholder="按标题搜索" value={keywordInput} onChange={(e) => setKeywordInput(e.target.value)} />
        <Button type="submit" variant="outline" size="icon" aria-label="搜索">
          <Search />
        </Button>
        <Select
          value={source}
          onValueChange={(value) => {
            setCurrent(1);
            setSource(value);
          }}
        >
          <SelectTrigger className="w-36">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL}>全部来源</SelectItem>
            {sources.map((s) => (
              <SelectItem key={s.code} value={s.code}>
                {s.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </form>

      <div className="rounded-xl border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>收件人</TableHead>
              <TableHead>标题</TableHead>
              <TableHead>附件</TableHead>
              <TableHead>来源</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>发送时间</TableHead>
              <TableHead>过期时间</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading && !data ? (
              <TableSkeleton cols={7} />
            ) : data?.mails.length ? (
              data.mails.map((mail) => {
                const attachments = bundleText(mail.attachments, items, currencies);
                const status = mailStatus(mail, loadedAt);
                let statusTitle: string | undefined;
                if (mail.removed_at) statusTitle = `${formatDateTime(mail.removed_at)} 移出信箱`;
                else if (mail.claimed_at) statusTitle = `${formatDateTime(mail.claimed_at)} 领取`;
                return (
                  <TableRow key={mail.id}>
                    <TableCell>
                      <div className="flex flex-col">
                        <span>{mail.username || '-'}</span>
                        <span className="font-mono text-xs text-muted-foreground">#{mail.user_id}</span>
                      </div>
                    </TableCell>
                    <TableCell className="max-w-56">
                      <div className="flex min-w-0 flex-col">
                        <span className="truncate font-medium" title={mail.content || mail.title}>
                          {mail.title}
                        </span>
                        <span className="truncate text-xs text-muted-foreground">{mail.sender}</span>
                      </div>
                    </TableCell>
                    <TableCell className="max-w-48 truncate" title={attachments.join(', ') || undefined}>
                      {attachments.length ? attachments.join(', ') : <span className="text-muted-foreground">无</span>}
                    </TableCell>
                    <TableCell>
                      <div className="flex flex-col">
                        <span>{sourceLabel(mail.source)}</span>
                        <span className="max-w-32 truncate font-mono text-xs text-muted-foreground" title={mail.source_id}>
                          {mail.source_id}
                        </span>
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant={status === '未读' || status === '已读' ? 'secondary' : 'outline'} title={statusTitle}>
                        {status}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground tabular-nums">{formatDateTime(mail.created_at)}</TableCell>
                    <TableCell className="text-xs text-muted-foreground tabular-nums">{formatDateTime(mail.expires_at)}</TableCell>
                  </TableRow>
                );
              })
            ) : (
              <TableEmpty colSpan={7}>没有找到邮件</TableEmpty>
            )}
          </TableBody>
        </Table>
      </div>

      <Pagination current={current} size={PAGE_SIZE} total={data?.total ?? 0} onChange={setCurrent} />
    </div>
  );
}
