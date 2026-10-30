/**
 * 页面访问规则: 哪些页面要登录、要管理员、已登录时不该再看, 以及不满足时去哪
 * proxy.ts 和客户端共用这一份; 新增页面放在已有前缀下即可, 新的前缀加进 AUTH_PREFIXES, 导航入口见 lib/nav.ts
 */

export const ROUTES = {
  landing: '/',
  gameHome: '/home',
  wallet: '/wallet',
  inventory: '/inventory',
  shop: '/shop',
  mail: '/mail',
  leaderboard: '/leaderboard',
  stats: '/stats',
  match: '/match',
  matches: '/matches',
  login: '/login',
  register: '/register',
  admin: '/admin',
  adminUsers: '/admin/users',
  adminItems: '/admin/items',
  adminQualities: '/admin/qualities',
  adminCurrencies: '/admin/currencies',
  adminLootPools: '/admin/loot-pools',
  adminVenues: '/admin/venues',
  adminLeaderboards: '/admin/leaderboards',
  adminSeasons: '/admin/seasons',
  adminTiers: '/admin/tiers',
  adminShopItems: '/admin/shop-items',
  adminShopOrders: '/admin/shop-orders',
  adminMails: '/admin/mails',
  adminMatches: '/admin/matches',
  adminSuspects: '/admin/suspects',
  adminSettings: '/admin/settings',
  adminTransactions: '/admin/transactions',
  adminPacks: '/admin/packs',
  adminAuditLogs: '/admin/audit-logs',
} as const;

/** 需要登录的路径前缀 */
const AUTH_PREFIXES = ['/home', '/wallet', '/inventory', '/shop', '/mail', '/leaderboard', '/stats', '/match', '/matches', '/admin'];
/** 需要管理员的路径前缀 */
const ADMIN_PREFIXES = ['/admin'];

function matches(pathname: string, prefixes: string[]) {
  return prefixes.some((prefix) => pathname === prefix || pathname.startsWith(`${prefix}/`));
}

/** 是否需要登录才能访问 */
export function requiresAuth(pathname: string) {
  return matches(pathname, AUTH_PREFIXES);
}

/** 是否需要管理员才能访问 */
export function requiresAdmin(pathname: string) {
  return matches(pathname, ADMIN_PREFIXES);
}

/** 对局页地址 */
export function matchHref(id: number) {
  return `${ROUTES.match}/${id}`;
}

/** 只允许站内相对路径, 防止开放重定向 */
export function safeRedirect(target: string | null | undefined, fallback: string = ROUTES.gameHome): string {
  if (!target || !target.startsWith('/') || target.startsWith('//') || target.startsWith('/\\')) {
    return fallback;
  }
  return target;
}

/** 登录页地址, 登录后回到 redirect */
export function loginHref(redirect?: string | null, page: string = ROUTES.login) {
  return redirect ? `${page}?redirect=${encodeURIComponent(redirect)}` : page;
}
