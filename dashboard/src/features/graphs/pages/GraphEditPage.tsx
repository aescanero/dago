import { useParams, Link, useNavigate } from "react-router-dom";
import { toast } from "sonner";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { GraphForm } from "../components/GraphForm";
import { useGraph } from "../hooks/useGraph";
import { useUpdateGraph } from "../hooks/useUpdateGraph";
import { formToGraphInput } from "../schemas/graph-form.schema";
import type { GraphFormValues } from "../schemas/graph-form.schema";

export function GraphEditPage() {
  const { id = "" } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { data: graph, isLoading } = useGraph(id);
  const { mutateAsync, isPending } = useUpdateGraph(id);

  async function handleSubmit(values: GraphFormValues) {
    try {
      await mutateAsync(formToGraphInput(values));
      toast.success("Graph updated");
      navigate(`/graphs/${id}`);
    } catch (err) {
      const error = err as Error & { status?: number };
      toast.error(error.message ?? "Error updating graph");
    }
  }

  if (isLoading) {
    return (
      <div className="p-6 space-y-4">
        <Skeleton className="h-6 w-48" />
        <Skeleton className="h-64 w-full max-w-2xl" />
      </div>
    );
  }

  if (!graph) {
    return (
      <div className="p-6">
        <p className="text-destructive">Graph not found.</p>
      </div>
    );
  }

  return (
    <div className="p-6 space-y-4 max-w-3xl">
      <Breadcrumb>
        <BreadcrumbList>
          <BreadcrumbItem>
            <BreadcrumbLink asChild>
              <Link to="/graphs">Graphs</Link>
            </BreadcrumbLink>
          </BreadcrumbItem>
          <BreadcrumbSeparator />
          <BreadcrumbItem>
            <BreadcrumbLink asChild>
              <Link to={`/graphs/${id}`}>{graph.name}</Link>
            </BreadcrumbLink>
          </BreadcrumbItem>
          <BreadcrumbSeparator />
          <BreadcrumbItem>
            <BreadcrumbPage>Edit</BreadcrumbPage>
          </BreadcrumbItem>
        </BreadcrumbList>
      </Breadcrumb>

      <h1 className="text-2xl font-bold">Edit graph</h1>

      {graph.status !== "draft" ? (
        <div className="space-y-4">
          <Alert variant="destructive">
            <AlertTitle>Cannot edit this graph</AlertTitle>
            <AlertDescription>
              Only draft graphs can be edited. This graph has status &quot;{graph.status}&quot;.
            </AlertDescription>
          </Alert>
          <Button variant="outline" asChild>
            <Link to={`/graphs/${id}`}>Back</Link>
          </Button>
        </div>
      ) : (
        <GraphForm
          defaultValues={{
            name: graph.name,
            version: graph.version,
            description: graph.description ?? "",
            entry_node: graph.entry_node,
            definition: JSON.stringify(graph.definition, null, 2),
            memory_semantic_search:
              graph.memory_config?.semantic_search ?? false,
            memory_episode_context:
              graph.memory_config?.episode_context ?? 0,
          }}
          onSubmit={handleSubmit}
          isPending={isPending}
          submitLabel="Save"
        />
      )}
    </div>
  );
}
