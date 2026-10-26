"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { LoaderCircle, Package } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { AuthService } from "@/lib/services/auth";
import { ApiError } from "@/lib/services/core";

type Mode = "login" | "register";

const USERNAME_PATTERN = /^[A-Za-z0-9_]{3,32}$/;

const COPY = {
  login: { description: "登录账号, 继续你的开箱之旅", submit: "登录", switchText: "还没有账号?", switchLink: "立即注册" },
  register: { description: "创建账号, 开启你的第一个盲盒", submit: "注册", switchText: "已有账号?", switchLink: "去登录" },
} as const;

/** 只允许站内相对路径, 防止开放重定向 */
function safeRedirect(target: string | null): string {
  if (!target || !target.startsWith("/") || target.startsWith("//") || target.startsWith("/\\")) {
    return "/";
  }
  return target;
}

/** 校验注册表单, 规则与后端一致; 登录只要求非空 */
function validate(mode: Mode, username: string, password: string, confirm: string): string | null {
  if (!username || !password) return "请填写完整";
  if (mode === "login") return null;
  if (!USERNAME_PATTERN.test(username)) return "用户名需为 3-32 位字母、数字或下划线";
  if (password.length < 8 || password.length > 72) return "密码长度需为 8-72 位";
  if (password !== confirm) return "两次输入的密码不一致";
  return null;
}

export function AuthForm({ mode }: { mode: Mode }) {
  const router = useRouter();
  const searchParams = useSearchParams();
  const redirect = searchParams.get("redirect");
  const copy = COPY[mode];

  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const name = username.trim();
    const invalid = validate(mode, name, password, confirm);
    if (invalid) {
      setError(invalid);
      return;
    }

    setError(null);
    setSubmitting(true);
    try {
      const body = { username: name, password };
      const user = mode === "login" ? await AuthService.login(body) : await AuthService.register(body);
      toast.success(mode === "login" ? `欢迎回来, ${user.nickname}` : `注册成功, 欢迎 ${user.nickname}`);
      router.replace(safeRedirect(redirect));
      router.refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "请求失败, 请稍后再试");
      setSubmitting(false);
    }
  };

  // 切换登录/注册时带上 redirect
  const switchHref = `/${mode === "login" ? "register" : "login"}${redirect ? `?redirect=${encodeURIComponent(redirect)}` : ""}`;

  return (
    <div className="flex w-full max-w-sm flex-col gap-6">
      <div className="flex flex-col items-center gap-2 text-center">
        <Link
          href="/"
          aria-label="返回首页"
          className="glow m-4 flex size-12 items-center justify-center rounded-xl border bg-card text-primary"
        >
          <Package className="size-6" />
        </Link>
        <h1 className="text-xl font-bold">
          欢迎来到 Blind<span className="text-primary">Crate</span>
        </h1>
        <p className="text-sm text-muted-foreground">{copy.description}</p>
      </div>
      <form onSubmit={handleSubmit} className="flex flex-col gap-4" noValidate>
        <div className="flex flex-col gap-2">
          <Label htmlFor="username">{mode === "login" ? "用户名或邮箱" : "用户名"}</Label>
          <Input
            id="username"
            autoComplete="username"
            autoFocus
            placeholder={mode === "login" ? "用户名 / 已验证的邮箱" : "3-32 位字母、数字或下划线"}
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            disabled={submitting}
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
            disabled={submitting}
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
              disabled={submitting}
            />
          </div>
        )}
        {error && (
          <p role="alert" className="text-center text-sm text-destructive">
            {error}
          </p>
        )}
        <Button type="submit" size="lg" className="glow mt-2 w-full" disabled={submitting}>
          {submitting && <LoaderCircle className="animate-spin" />}
          {submitting ? `正在${copy.submit}...` : copy.submit}
        </Button>
      </form>

      <p className="text-center text-xs text-muted-foreground">
        {copy.switchText}
        <Link href={switchHref} className="ml-1 text-primary underline-offset-4 hover:underline">
          {copy.switchLink}
        </Link>
      </p>
    </div>
  );
}
