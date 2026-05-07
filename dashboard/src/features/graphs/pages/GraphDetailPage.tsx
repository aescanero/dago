import { useParams, Link } from "react-router-dom";
import { GitGraph } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { GraphStatusBadge } from "../components/GraphStatusBadge";
import { GraphDefinitionViewer } from "../components/GraphDefinitionViewer";
import { GraphArchiveDialog } from "../components/GraphArchiveDialog";
import { useGraph } from "../hooks/useGraph";

export function GraphDetailPage() {
  const { id } = useParams<{ id: string }>();
  const { data: graph, isLoading, error } = useGraph(id ?? "");

  if (isLoading) {
    return (
      <div className="p-6 space-y-4">
        <Skeleton className="h-6 w-48" />
        <Skeleton className="h-10 w-full max-w-md" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  if (error || !graph) {
    return (
      <div className="p-6">
        <p className="text-destructive">
          Graph not found or could not be loaded.
        </p>
      </div>
    );
  }

  return (
    <div className="p-6 space-y-4">
      <Breadcrumb>
        <BreadcrumbList>
          <BreadcrumbItem>
            <BreadcrumbLink asChild>
              <Link to="/graphs">Graphs</Link>
            </BreadcrumbLink>
          </BreadcrumbItem>
          <BreadcrumbSeparator />
          <BreadcrumbItem>
            <BreadcrumbPage>{graph.name}</BreadcrumbPage>
          </BreadcrumbItem>
        </BreadcrumbList>
      </Breadcrumb>

      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <GitGraph className="h-5 w-5 text-muted-foreground" />
          <h1 className="text-2xl font-bold">{graph.name}</h1>
          <span className="text-muted-foreground">{graph.version}</span>
          <GraphStatusBadge status={graph.status} />
        </div>
        <div className="flex items-center gap-2">
          {graph.status === "draft" && (
            <Button variant="outline" size="sm" asChild>
              <Link to={`/graphs/${graph.id}/edit`}>Edit</Link>
            </Button>
          )}
          {graph.status !== "archived" && (
            <GraphArchiveDialog graphId={graph.id} />
          )}
        </div>
      </div>

      <Tabs defaultValue="metadata">
        <TabsList>
          <TabsTrigger value="metadata">Metadata</TabsTrigger>
          <TabsTrigger value="definition">Definition</TabsTrigger>
          <TabsTrigger value="executions">Executions</TabsTrigger>
        </TabsList>

        <TabsContent value="metadata">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Graph details</CardTitle>
            </CardHeader>
            <CardContent className="grid gap-3 text-sm">
              {graph.description && (
                <div>
                  <span className="font-medium">Description: </span>
                  {graph.description}
                </div>
              )}
              <div>
                <span className="font-medium">Entry node: </span>
                <code className="font-mono bg-muted px-1 rounded">
                  {graph.entry_node}
                </code>
              </div>
              <div>
                <span className="font-medium">Created: </span>
                {new Date(graph.created_at).toLocaleString()}
              </div>
              <div>
                <span className="font-medium">Updated: </span>
                {new Date(graph.updated_at).toLocaleString()}
              </div>
              {graph.memory_config && (
                <div className="space-y-1">
                  <span className="font-medium">Memory config:</span>
                  <div className="ml-4 space-y-1 text-muted-foreground">
                    <div>
                      Semantic search:{" "}
                      {graph.memory_config.semantic_search ? "enabled" : "disabled"}
                    </div>
                    <div>
                      Episodic context: {graph.memory_config.episode_context ?? 0}
                    </div>
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="definition">
          <Card>
            <CardContent className="pt-4">
              <GraphDefinitionViewer definition={graph.definition} />
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="executions">
          <Card>
            <CardContent className="pt-6 flex flex-col items-center gap-4 py-12">
              <p className="text-muted-foreground text-sm">
                Executions will appear here once they are started.
              </p>
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span>
                      <Button disabled>Start execution</Button>
                    </span>
                  </TooltipTrigger>
                  <TooltipContent>
                    Available in the next version
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}
