import {NextResponse, type NextRequest} from 'next/server';
import {ROUTES, loginHref, requiresAuth} from '@/lib/access';

/** 与后端 app.session_cookie_name 一致 */
const SESSION_COOKIE = process.env.SESSION_COOKIE_NAME ?? 'blindcrate_session_id';
const BACKEND_ORIGIN = process.env.BACKEND_ORIGIN ?? 'http://localhost:8000';

/**
 * 在页面渲染前按登录状态分流, 避免先渲染再跳转的闪烁
 * - 需要登录的页面没有 session cookie: 直接去登录页
 * - 首页且会话有效: 直接去游戏首页
 * cookie 存在不代表会话有效(退出登录、被下线后 cookie 仍在), 失效的情况由客户端收到 401 后处理
 */
export async function proxy(request: NextRequest) {
  const {pathname, search} = request.nextUrl;
  const session = request.cookies.get(SESSION_COOKIE);

  if (requiresAuth(pathname)) {
    if (!session) {
      return NextResponse.redirect(new URL(loginHref(pathname + search), request.url));
    }
    return NextResponse.next();
  }

  // 首页: 只有确认会话有效才跳转, 否则失效 cookie 会把人困在登录页和游戏首页之间
  if (pathname === ROUTES.landing && session && (await hasValidSession(request))) {
    return NextResponse.redirect(new URL(ROUTES.gameHome, request.url));
  }
  return NextResponse.next();
}

async function hasValidSession(request: NextRequest) {
  try {
    const res = await fetch(`${BACKEND_ORIGIN}/api/v1/auth/user-info`, {
      headers: {cookie: request.headers.get('cookie') ?? ''},
      cache: 'no-store',
      signal: AbortSignal.timeout(3000),
    });
    return res.ok;
  } catch {
    // 后端不可用时留在首页
    return false;
  }
}

// 除接口、静态资源和带扩展名的文件外都经过这里, 具体哪些页面要登录由 requiresAuth 判断, 不在这里重复列出
export const config = {
  matcher: ['/((?!api/|_next/|.*\\..*).*)'],
};
