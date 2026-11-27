'use client';

import {useState} from 'react';
import {Ticket} from 'lucide-react';
import {useForm} from 'react-hook-form';
import {Button} from '@/components/ui/button';
import {Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle} from '@/components/ui/dialog';
import {Form, FormControl, FormField, FormItem, FormLabel, FormMessage} from '@/components/ui/form';
import {Input} from '@/components/ui/input';
import {SubmitButton} from '@/components/common/form';
import {RewardResultDialog} from '@/components/common/reward';
import {useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';
import type {RedeemResult} from '@/lib/services/redeem';

interface RedeemValues {
  code: string;
}

/** 输入框中的兑换码最多的字符数, 含分隔用的空格和连字符, 与后端一致 */
const MAX_CODE_INPUT = 64;

/**
 * 兑换码入口: 点击后输入兑换码, 兑换成功后展示得到的奖励
 */
/**
 * 兑换码入口: 兑换成功后奖励以邮件发放, 在信箱中领取
 */
export function RedeemButton() {
  const [open, setOpen] = useState(false);
  const [result, setResult] = useState<RedeemResult | null>(null);

  return (
    <>
      <Button variant="outline" onClick={() => setOpen(true)}>
        <Ticket />
        兑换码
      </Button>

      <Dialog open={open} onOpenChange={(next) => !next && setOpen(false)}>
        <DialogContent className="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>使用兑换码</DialogTitle>
            <DialogDescription>输入兑换码, 奖励会发送到信箱; 不区分大小写</DialogDescription>
          </DialogHeader>
          {/* 每次打开都是新的表单 */}
          {open && (
            <RedeemForm
              onClose={() => setOpen(false)}
              onRedeemed={setResult}
            />
          )}
        </DialogContent>
      </Dialog>

      <RewardResultDialog
        lines={result?.reward ?? null}
        title="兑换成功"
        description={result && `${result.name} 的奖励已发送到信箱, 请在邮件中领取`}
        onClose={() => setResult(null)}
      />
    </>
  );
}

/**
 * 兑换码表单
 */
function RedeemForm({onClose, onRedeemed}: { onClose: () => void; onRedeemed: (result: RedeemResult) => void }) {
  const {pending, submit} = useSubmit();
  const form = useForm<RedeemValues>({defaultValues: {code: ''}});

  // 兑换码是否有效由后端判断, 失败的原因以提示框展示
  const handleSubmit = form.handleSubmit(({code}) => submit(async () => {
    const redeemed = await services.redeem.redeem(code.trim());
    onClose();
    onRedeemed(redeemed);
  })());

  return (
    <Form {...form}>
      <form onSubmit={handleSubmit} className="flex flex-col gap-4" noValidate>
        <FormField
          control={form.control}
          name="code"
          rules={{validate: (value) => (value.trim() ? true : '请填写兑换码')}}
          render={({field}) => (
            <FormItem>
              <FormLabel>兑换码</FormLabel>
              <FormControl>
                <Input
                  autoFocus
                  autoComplete="off"
                  spellCheck={false}
                  maxLength={MAX_CODE_INPUT}
                  placeholder="请输入兑换码"
                  className="font-mono"
                  disabled={pending}
                  {...field}
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose} disabled={pending}>
            取消
          </Button>
          <SubmitButton pending={pending}>兑换</SubmitButton>
        </DialogFooter>
      </form>
    </Form>
  );
}
