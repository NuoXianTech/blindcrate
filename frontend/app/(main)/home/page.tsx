import type { Metadata } from "next";

// AuthGuard 确认登录前不渲染页面, 校验不到, 不参与即时导航校验
export const instant = false;

export const metadata: Metadata = { title: "游戏大厅" };

/** 游戏大厅, 内容后续补充 */
export default function GameHomePage() {
  return null;
}
