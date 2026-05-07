import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

type EdgeType =
  | "sequential"
  | "conditional"
  | "parallel"
  | "loop"
  | "interrupt";

interface Props {
  type: string;
}

const edgeClasses: Record<EdgeType, string> = {
  sequential:
    "bg-slate-100 text-slate-800 dark:bg-slate-800 dark:text-slate-100 border-transparent",
  conditional:
    "bg-amber-100 text-amber-800 dark:bg-amber-900 dark:text-amber-100 border-transparent",
  parallel:
    "bg-cyan-100 text-cyan-800 dark:bg-cyan-900 dark:text-cyan-100 border-transparent",
  loop:
    "bg-violet-100 text-violet-800 dark:bg-violet-900 dark:text-violet-100 border-transparent",
  interrupt:
    "bg-rose-100 text-rose-800 dark:bg-rose-900 dark:text-rose-100 border-transparent",
};

export function EdgeTypeBadge({ type }: Props) {
  const extra = edgeClasses[type as EdgeType] ?? "";
  return (
    <Badge variant="outline" className={cn("font-mono text-xs", extra)}>
      {type}
    </Badge>
  );
}
