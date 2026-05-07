import { Badge } from "@/components/ui/badge";
import type { GraphResponse } from "@/api/types.gen";

interface Props {
  status: GraphResponse["status"];
}

export function GraphStatusBadge({ status }: Props) {
  const variant =
    status === "active"
      ? "default"
      : status === "archived"
        ? "outline"
        : "secondary";

  return <Badge variant={variant}>{status}</Badge>;
}
