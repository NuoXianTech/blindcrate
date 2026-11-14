'use client';

import {useEffect, useState} from 'react';
import Link from 'next/link';
import {useRouter, useSearchParams} from 'next/navigation';
import {LoaderCircle} from 'lucide-react';
import {useForm} from 'react-hook-form';
import {toast} from 'sonner';
import {Button} from '@/components/ui/button';
import {Form, FormControl, FormField, FormItem, FormLabel, FormMessage} from '@/components/ui/form';
import {Input} from '@/components/ui/input';
import {useCaptcha} from '@/components/common/captcha';
import {SiteLogo, useLiveSite} from '@/components/common/site';
import {SubmitButton} from '@/components/common/form';
import {CurrencyIcon} from '@/components/common/wallet';
import {useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';
import {ROUTES, loginHref, safeRedirect} from '@/lib/access';
import {formatAmount} from '@/lib/utils';
import {checkEmail, checkPassword, checkPasswordConfirm, checkUsername} from '@/lib/validation';
import {useAuth} from './AuthProvider';

type Mode = 'login' | 'register';

interface AuthValues {
  username: string;
  password: string;
  /** 确认密码, 只有注册时有 */
  confirm: string;
  /** 邮箱和验证码, 只有开启注册时验证邮箱后有 */
  email: string;
  code: string;
}

/** 发送验证码的冷却秒数, 与后端一致 */
const EMAIL_CODE_COOLDOWN = 60;

const COPY = {
  login: {description: '登录账号, 继续开箱', submit: '登录', switchText: '还没有账号?', switchLink: '注册一个', switchTo: ROUTES.register},
  register: {description: '注册账号, 开启你的第一个盲盒', submit: '注册', switchText: '已有账号?', switchLink: '直接登录', switchTo: ROUTES.login},
} as const;

/**
 * 登录、注册表单, 成功后跳回来源页
 */
export function AuthForm({mode}: { mode: Mode }) {
  const router = useRouter();
  const redirect = useSearchParams().get('redirect');
  const copy = COPY[mode];
  const {user, setUser} = useAuth();
  // 页面上的公开设置有缓存, 管理员刚改过注册、邮箱验证、人机验证的开关时可能是旧的; 打开表单时读取最新的设置
  const {site, auth, economy} = useLiveSite();
  // 关闭注册时注册页只显示提示, 登录页不显示注册入口
  const registerClosed = !auth.register_enabled;
  // 开启后注册时要填写邮箱并通过验证码验证
  const emailRequired = mode === 'register' && auth.register_email_required;
  const {pending, submit} = useSubmit();
  const send = useSubmit();
  // 登录和注册各有开关; 发送验证码和注册共用一个组件, 令牌每次请求用掉一个
  const captcha = useCaptcha(mode);
  const [cooldown, setCooldown] = useState(0);

  const form = useForm<AuthValues>({defaultValues: {username: '', password: '', confirm: '', email: '', code: ''}});

  useEffect(() => {
    if (cooldown <= 0) return;
    const timer = setTimeout(() => setCooldown(cooldown - 1), 1000);
    return () => clearTimeout(timer);
  }, [cooldown]);

  // 发送前只校验邮箱一栏, 其他字段可以稍后再填
  const handleSend = send.submit(async () => {
    if (!(await form.trigger('email'))) return;
    await services.auth.sendRegisterCode(form.getValues('email').trim(), captcha.take());
    setCooldown(EMAIL_CODE_COOLDOWN);
    toast.success('验证码已发送, 10 分钟内有效');
  });

  // 已登录(含刚登录成功)时离开登录页
  useEffect(() => {
    if (user) router.replace(safeRedirect(redirect));
  }, [user, redirect, router]);

  // 字段校验由表单完成, 错误显示在字段下方; 请求失败仍以提示框展示
  const handleSubmit = form.handleSubmit(({username, password, email, code}) => submit(async () => {
    const body = {username: username.trim(), password};
    const token = captcha.take();
    if (mode === 'login') {
      const info = await services.auth.login(body, token);
      toast.success(`欢迎回来, ${info.nickname}`);
      setUser(info);
      return;
    }
    const info = await services.auth.register(emailRequired ? {...body, email: email.trim(), code: code.trim()} : body, token);
    toast.success(`注册成功, 欢迎 ${info.nickname}`, {description: await grantedText(economy.base_currency)});
    setUser(info);
  })());

  return (
    <div className="flex w-full max-w-sm flex-col gap-6">
      <div className="flex flex-col items-center gap-2 text-center">
        <Link
          href={ROUTES.landing}
          aria-label="返回首页"
          className="m-4 flex size-12 items-center justify-center overflow-hidden rounded-xl border bg-card text-primary"
        >
          <SiteLogo />
        </Link>
        <h1 className="text-xl font-bold">欢迎来到 {site.name}</h1>
        <p className="text-sm text-muted-foreground">
          {mode === 'register' && registerClosed ? '暂未开放注册, 已有账号可以直接登录' : copy.description}
        </p>
      </div>

      {mode === 'register' && registerClosed ? (
        <Button asChild size="lg" className="w-full">
          <Link href={loginHref(redirect)}>去登录</Link>
        </Button>
      ) : (
        <Form {...form}>
          <form onSubmit={handleSubmit} className="flex flex-col gap-4" noValidate>
            <FormField
              control={form.control}
              name="username"
              rules={{
                validate: (v) => (!v.trim() ? '请填写用户名' : mode === 'register' ? checkUsername(v.trim()) ?? true : true),
              }}
              render={({field}) => (
                <FormItem>
                  <FormLabel>{mode === 'login' && auth.email_login_enabled ? '用户名或邮箱' : '用户名'}</FormLabel>
                  <FormControl>
                    <Input
                      autoComplete="username"
                      autoFocus
                      placeholder={
                        mode === 'register' ?
                          '3-32 位字母、数字或下划线' :
                          auth.email_login_enabled ?
                            '用户名 / 已验证的邮箱' :
                            '请输入用户名'
                      }
                      disabled={pending}
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="password"
              rules={{
                validate: (v) => (!v ? '请填写密码' : mode === 'register' ? checkPassword(v) ?? true : true),
              }}
              render={({field}) => (
                <FormItem>
                  <FormLabel>密码</FormLabel>
                  <FormControl>
                    <Input
                      type="password"
                      autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
                      placeholder={mode === 'login' ? '请输入密码' : '8-72 位'}
                      disabled={pending}
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            {mode === 'register' && (
              <FormField
                control={form.control}
                name="confirm"
                rules={{validate: (v, values) => checkPasswordConfirm(values.password, v) ?? true}}
                render={({field}) => (
                  <FormItem>
                    <FormLabel>确认密码</FormLabel>
                    <FormControl>
                      <Input type="password" autoComplete="new-password" placeholder="再次输入密码" disabled={pending} {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            )}
            {emailRequired && (
              <>
                <FormField
                  control={form.control}
                  name="email"
                  rules={{validate: (v) => (!v.trim() ? '请填写邮箱' : checkEmail(v.trim(), auth.block_email_alias) ?? true)}}
                  render={({field}) => (
                    <FormItem>
                      <FormLabel>邮箱</FormLabel>
                      <div className="flex gap-2">
                        <FormControl>
                          <Input type="email" autoComplete="email" placeholder="name@example.com" disabled={pending} {...field} />
                        </FormControl>
                        <Button type="button" variant="secondary" onClick={() => handleSend()} disabled={pending || send.pending || cooldown > 0 || !captcha.ready}>
                          {send.pending && <LoaderCircle className="animate-spin" />}
                          {cooldown > 0 ? `${cooldown} 秒后重发` : '发送验证码'}
                        </Button>
                      </div>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name="code"
                  rules={{validate: (v) => (v.trim() ? true : '请填写验证码')}}
                  render={({field}) => (
                    <FormItem>
                      <FormLabel>验证码</FormLabel>
                      <FormControl>
                        <Input
                          inputMode="numeric"
                          autoComplete="one-time-code"
                          maxLength={6}
                          placeholder="6 位数字"
                          disabled={pending}
                          {...field}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </>
            )}
            {mode === 'register' && economy.signup_grant > 0 && (
              <p className="flex items-center justify-center gap-1.5 rounded-lg border bg-card px-3 py-2 text-sm">
                注册即送
                <CurrencyIcon currency={economy.base_currency} className="size-4" />
                {/* 注册前拿不到货币的名称和小数位, 只显示图标和整数 */}
                <span className="font-mono font-bold tabular-nums">{formatAmount(economy.signup_grant)}</span>
              </p>
            )}
            {captcha.widget}
            <SubmitButton pending={pending} disabled={!captcha.ready} size="lg" className="mt-2 w-full">
              {pending ? `正在${copy.submit}...` : copy.submit}
            </SubmitButton>
          </form>
        </Form>
      )}

      {!registerClosed && (
        <p className="text-center text-xs text-muted-foreground">
          {copy.switchText}
          {/* 切换登录/注册时带上 redirect */}
          <Link href={loginHref(redirect, copy.switchTo)} className="ml-1 text-primary underline-offset-4 hover:underline">
            {copy.switchLink}
          </Link>
        </p>
      )}
    </div>
  );
}

/**
 * 注册成功后新手资金的到账提示, 以余额为准: 关闭发放或发放失败时余额为 0, 不提示
 */
async function grantedText(currency: string): Promise<string | undefined> {
  try {
    const {balances} = await services.wallet.listBalances();
    const granted = balances.find((b) => b.currency === currency);
    return granted && granted.balance > 0 ?
      `新手资金 ${formatAmount(granted.balance, granted.decimals)} ${granted.name}已到账` :
      undefined;
  } catch {
    // 只影响提示, 不影响注册
    return undefined;
  }
}
