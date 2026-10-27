import { CircleDollarSign, Coins, Gem } from "lucide-react";
import { cn } from "cn";

/** 各货币的图标和颜色, 与后端 internal/apps/wallet/constants.go 的货币代码对应, 新货币没有配置时用通用图标 */
const CURRENCY_STYLES: Record<string, { icon: typeof Coins; className: string }> = {
  chip: { icon: Coins, className: "text-primary" },
  gold: { icon: Gem, className: "text-quality-gold" },
};

const FALLBACK_STYLE = { icon: CircleDollarSign, className: "text-muted-foreground" };

export function CurrencyIcon({ currency, className }: { currency: string; className?: string }) {
  const style = CURRENCY_STYLES[currency] ?? FALLBACK_STYLE;
  const Icon = style.icon;
  return <Icon className={cn("shrink-0", style.className, className)} aria-hidden />;
}
