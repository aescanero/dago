import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

type NodePattern =
  | "llm_call"
  | "react"
  | "reflection"
  | "tool_use"
  | "router"
  | "guardrail"
  | "subgraph";

interface Props {
  pattern: string;
}

const patternClasses: Record<NodePattern, string> = {
  llm_call:
    "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-100 border-transparent",
  react:
    "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-100 border-transparent",
  reflection:
    "bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-100 border-transparent",
  tool_use:
    "bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-100 border-transparent",
  router:
    "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-100 border-transparent",
  guardrail:
    "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-100 border-transparent",
  subgraph:
    "bg-gray-200 text-gray-800 dark:bg-gray-700 dark:text-gray-100 border-transparent",
};

export function NodePatternBadge({ pattern }: Props) {
  const extra = patternClasses[pattern as NodePattern] ?? "";
  return (
    <Badge variant="outline" className={cn("font-mono text-xs", extra)}>
      {pattern}
    </Badge>
  );
}
