import { useState } from "react";
import { useGraphs } from "@/hooks/useGraphs";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

function statusVariant(status: string): "default" | "success" | "warning" | "destructive" {
  switch (status) {
    case "active": return "success";
    case "inactive": return "warning";
    case "error": return "destructive";
    default: return "default";
  }
}

export function GraphsPage() {
  const [page, setPage] = useState(1);
  const { data, isLoading, error } = useGraphs({ page, perPage: 20 });

  if (isLoading) {
    return (
      <div className="p-6 space-y-4">
        <h1 className="text-2xl font-bold">Graphs</h1>
        {Array.from({ length: 5 }).map((_, i) => (
          <Skeleton key={i} className="h-12 w-full" />
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-6">
        <h1 className="text-2xl font-bold">Graphs</h1>
        <p className="text-destructive mt-4">Error loading graphs. Please try again.</p>
      </div>
    );
  }

  const items = data?.items ?? [];
  const pagination = data?.pagination;

  return (
    <div className="p-6 space-y-4">
      <h1 className="text-2xl font-bold">Graphs</h1>

      {items.length === 0 ? (
        <p className="text-muted-foreground">No graphs found.</p>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Version</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Created</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.map((graph) => (
              <TableRow key={graph.id}>
                <TableCell className="font-medium">{graph.name}</TableCell>
                <TableCell>{graph.version}</TableCell>
                <TableCell>
                  <Badge variant={statusVariant(graph.status)}>{graph.status}</Badge>
                </TableCell>
                <TableCell>{new Date(graph.created_at).toLocaleDateString()}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      {pagination && pagination.total_pages > 1 && (
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            disabled={page <= 1}
            onClick={() => setPage((p) => p - 1)}
          >
            Previous
          </Button>
          <span className="text-sm text-muted-foreground">
            Page {pagination.page} of {pagination.total_pages}
          </span>
          <Button
            variant="outline"
            size="sm"
            disabled={page >= pagination.total_pages}
            onClick={() => setPage((p) => p + 1)}
          >
            Next
          </Button>
        </div>
      )}
    </div>
  );
}
