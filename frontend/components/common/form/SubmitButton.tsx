import { LoaderCircle } from "lucide-react";
import { Button } from "@/components/ui/button";

/** 表单提交按钮, 提交中禁用并转圈 */
export function SubmitButton({
  pending,
  disabled,
  children,
  ...props
}: React.ComponentProps<typeof Button> & { pending: boolean }) {
  return (
    <Button type="submit" {...props} disabled={pending || disabled}>
      {pending && <LoaderCircle className="animate-spin" />}
      {children}
    </Button>
  );
}
