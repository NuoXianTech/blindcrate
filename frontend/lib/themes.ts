/**
 * 可选主题列表, 都是暗色系
 * 新增主题: 在这里加一项, 再到 app/globals.css 里复制一份 [data-theme="xxx"] 改变量
 */
export const THEMES = [{ value: "amber", label: "琥珀金" }] as const;

export type ThemeName = (typeof THEMES)[number]["value"];

export const DEFAULT_THEME: ThemeName = "amber";
