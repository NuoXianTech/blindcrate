/**
 * 页面访问规则: 哪些页面要登录、要管理员、已登录时不该再看, 以及不满足时去哪
 * proxy.ts 和客户端共用这一份, 新增页面只需放进对应前缀下
 */

export const ROUTES = {
  landing: "/",
  gameHome: "/home",
  wallet: "/wallet",
  inventory: "/inventory",
  login: "/login",
  register: "/register",
  adminUsers: "/admin/users",
  adminItems: "/admin/items",
  adminSettings: "/admin/settings",
  adminTransactions: "/admin/transactions",
  adminAuditLogs: "/admin/audit-logs",
} as const;

/** 需要登录的路径前缀 */
const AUTH_PREFIXES = ["/home", "/wallet", "/inventory", "/admin"];
/** 需要管理员的路径前缀 */
const ADMIN_PREFIXES = ["/admin"];

function matches(pathname: string, prefixes: string[]) {
  return prefixes.some((prefix) => pathname === prefix || pathname.startsWith(`${prefix}/`));
}

export function requiresAuth(pathname: string) {
  return matches(pathname, AUTH_PREFIXES);
}

export function requiresAdmin(pathname: string) {
  return matches(pathname, ADMIN_PREFIXES);
}

/** 只允许站内相对路径, 防止开放重定向 */
export function safeRedirect(target: string | null | undefined, fallback: string = ROUTES.gameHome): string {
  if (!target || !target.startsWith("/") || target.startsWith("//") || target.startsWith("/\\")) {
    return fallback;
  }
  return target;
}

/** 登录页地址, 登录后回到 redirect */
export function loginHref(redirect?: string | null, page: string = ROUTES.login) {
  return redirect ? `${page}?redirect=${encodeURIComponent(redirect)}` : page;
}
