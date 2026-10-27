'use client';

import {useState} from 'react';
import {Send} from 'lucide-react';
import {toast} from 'sonner';
import {cn} from 'cn';
import {Input} from '@/components/ui/input';
import {Label} from '@/components/ui/label';
import {useAuth} from '@/components/common/auth';
import {SubmitButton} from '@/components/common/form';
import {check, useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';

/**
 * 用已保存的邮件设置发送测试邮件
 */
export function MailTest({dirty}: { dirty: boolean }) {
  const {user} = useAuth();
  const {pending, submit} = useSubmit();
  const [to, setTo] = useState(user?.email ?? '');

  const handleSubmit = submit(async () => {
    check(to.trim() ? null : '请填写收件人');
    await services.admin.testMail(to.trim());
    toast.success(`测试邮件已发送到 ${to.trim()}`);
  });

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-2 border-t pt-6" noValidate>
      <Label htmlFor="mail-test-to">发送测试邮件</Label>
      <div className="flex gap-2">
        <Input
          id="mail-test-to"
          type="email"
          placeholder="收件人邮箱"
          value={to}
          onChange={(e) => setTo(e.target.value)}
          disabled={pending}
          className="sm:max-w-xs"
        />
        <SubmitButton pending={pending} variant="secondary">
          {!pending && <Send />}
          发送
        </SubmitButton>
      </div>
      <span className={cn('text-xs', dirty ? 'text-quality-gold' : 'text-muted-foreground')}>
        {dirty ? '使用已保存的设置发送, 当前未保存的修改不会生效' : '使用已保存的设置发送, 未启用邮件发送时也可以测试'}
      </span>
    </form>
  );
}
