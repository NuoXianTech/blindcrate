"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { SiteLogo, useSite } from "@/components/common/layout/Site";
import { SubmitButton } from "@/components/common/form/SubmitButton";
import { check, useSubmit } from "@/components/common/form/use-submit";
import { AuthService } from "@/lib/services/auth";
import { ROUTES, loginHref, safeRedirect } from "@/lib/access";
import { checkPassword, checkPasswordConfirm, checkUsername } from "@/lib/validation";
import { useAuth } from "./AuthProvider";

type Mode = "login" | "register";

const COPY = {
  login: { description: "登录账号, 继续开箱", submit: "登录", switchText: "还没有账号?", switchLink: "注册一个", switchTo: ROUTES.register },
  register: { description: "注册账号, 开启你的第一个盲盒", submit: "注册", switchText: "已有账号?", switchLink: "直接登录", switchTo: ROUTES.login },
} as const;

export function AuthForm({ mode }: { mode: Mode }) {
  const router = useRouter();
  const redirect = useSearchParams().get("redirect");
  const copy = COPY[mode];
  const { user, setUser } = useAuth();
  const { site, auth } = useSite();
  // 关闭注册时注册页只显示提示, 登录页不显示注册入口
  const registerClosed = !auth.register_enabled;
  const { pending, submit } = useSubmit();

  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");

  // 已登录(含刚登录成功)时离开登录页
  useEffect(() => {
    if (user) router.replace(safeRedirect(redirect));
  }, [user, redirect, router]);

  const handleSubmit = submit(async () => {
    const name = username.trim();
    check(!name || !password ? "请填写用户名和密码" : null);
    if (mode === "register") {
      check(checkUsername(name));
      check(checkPassword(password));
      check(checkPasswordConfirm(password, confirm));
    }
    const body = { username: name, password };
    const info = mode === "login" ? await AuthService.login(body) : await AuthService.register(body);
    toast.success(mode === "login" ? `欢迎回来, ${info.nickname}` : `注册成功, 欢迎 ${info.nickname}`);
    setUser(info);
  });

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
          {mode === "register" && registerClosed ? "暂未开放注册, 已有账号可以直接登录" : copy.description}
        </p>
      </div>

      {mode === "register" && registerClosed ? (
        <Button asChild size="lg" className="w-full">
          <Link href={loginHref(redirect)}>去登录</Link>
        </Button>
      ) : (
        <form onSubmit={handleSubmit} className="flex flex-col gap-4" noValidate>
          <div className="flex flex-col gap-2">
            <Label htmlFor="username">{mode === "login" && auth.email_login_enabled ? "用户名或邮箱" : "用户名"}</Label>
            <Input
              id="username"
              autoComplete="username"
              autoFocus
              placeholder={
                mode === "register"
                  ? "3-32 位字母、数字或下划线"
                  : auth.email_login_enabled
                    ? "用户名 / 已验证的邮箱"
                    : "请输入用户名"
              }
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              disabled={pending}
            />
          </div>
          <div className="flex flex-col gap-2">
            <Label htmlFor="password">密码</Label>
            <Input
              id="password"
              type="password"
              autoComplete={mode === "login" ? "current-password" : "new-password"}
              placeholder={mode === "login" ? "请输入密码" : "8-72 位"}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              disabled={pending}
            />
          </div>
          {mode === "register" && (
            <div className="flex flex-col gap-2">
              <Label htmlFor="confirm">确认密码</Label>
              <Input
                id="confirm"
                type="password"
                autoComplete="new-password"
                placeholder="再次输入密码"
                value={confirm}
                onChange={(e) => setConfirm(e.target.value)}
                disabled={pending}
              />
            </div>
          )}
          <SubmitButton pending={pending} size="lg" className="mt-2 w-full">
            {pending ? `正在${copy.submit}...` : copy.submit}
          </SubmitButton>
        </form>
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
