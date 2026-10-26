const dateTimeFormat = new Intl.DateTimeFormat("zh-CN", {
  year: "numeric",
  month: "2-digit",
  day: "2-digit",
  hour: "2-digit",
  minute: "2-digit",
  second: "2-digit",
  hour12: false,
});

/** 格式化后端返回的时间, 空值显示 "-" */
export function formatDateTime(value: string | null | undefined): string {
  return value ? dateTimeFormat.format(new Date(value)) : "-";
}
