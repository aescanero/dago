import { useState } from "react";
import { ChevronDown, ChevronRight } from "lucide-react";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { NodePatternBadge } from "./NodePatternBadge";
import { EdgeTypeBadge } from "./EdgeTypeBadge";

interface NodeDef {
  pattern?: string;
  config?: Record<string, unknown>;
}

interface EdgeDef {
  type?: string;
  from?: string;
  to?: string;
  conditions?: unknown[];
}

interface GraphDefinition {
  nodes?: Record<string, NodeDef>;
  edges?: EdgeDef[];
}

interface Props {
  definition: unknown;
}

function isGraphDefinition(v: unknown): v is GraphDefinition {
  return (
    typeof v === "object" &&
    v !== null &&
    ("nodes" in v || "edges" in v)
  );
}

function NodeRow({ nodeKey, node }: { nodeKey: string; node: NodeDef }) {
  const [open, setOpen] = useState(false);
  const summary = [
    node.config?.model ? `model: ${node.config.model}` : null,
    node.config?.max_iterations
      ? `max_iter: ${node.config.max_iterations}`
      : null,
  ]
    .filter(Boolean)
    .join(", ");

  return (
    <Collapsible open={open} onOpenChange={setOpen}>
      <CollapsibleTrigger className="flex items-center gap-2 w-full text-left py-1 hover:bg-muted/50 rounded px-2">
        {open ? (
          <ChevronDown className="h-3 w-3 shrink-0" />
        ) : (
          <ChevronRight className="h-3 w-3 shrink-0" />
        )}
        {node.pattern && <NodePatternBadge pattern={node.pattern} />}
        <span className="font-mono text-sm">{nodeKey}</span>
        {summary && (
          <span className="text-xs text-muted-foreground ml-auto">
            {summary}
          </span>
        )}
      </CollapsibleTrigger>
      <CollapsibleContent>
        <pre className="ml-6 text-xs bg-muted rounded p-2 overflow-x-auto">
          {JSON.stringify(node.config ?? {}, null, 2)}
        </pre>
      </CollapsibleContent>
    </Collapsible>
  );
}

function EdgeRow({ edge, idx }: { edge: EdgeDef; idx: number }) {
  const label =
    edge.from && edge.to
      ? `${edge.from} → ${edge.to}`
      : `edge ${idx + 1}`;
  const condCount = edge.conditions?.length ?? 0;

  return (
    <div className="flex items-center gap-2 py-1 px-2 text-sm">
      <span className="font-mono">{label}</span>
      {edge.type && <EdgeTypeBadge type={edge.type} />}
      {condCount > 0 && (
        <span className="text-xs text-muted-foreground">
          {condCount} condition{condCount > 1 ? "s" : ""}
        </span>
      )}
    </div>
  );
}

export function GraphDefinitionViewer({ definition }: Props) {
  const [nodesOpen, setNodesOpen] = useState(true);
  const [edgesOpen, setEdgesOpen] = useState(true);

  if (!isGraphDefinition(definition)) {
    return (
      <Alert variant="destructive">
        <AlertDescription>
          <p className="mb-2">Invalid definition format.</p>
          <pre className="text-xs overflow-x-auto">
            {typeof definition === "string"
              ? definition
              : JSON.stringify(definition, null, 2)}
          </pre>
        </AlertDescription>
      </Alert>
    );
  }

  const nodes = definition.nodes ?? {};
  const edges = definition.edges ?? [];
  const nodeEntries = Object.entries(nodes);

  return (
    <ScrollArea className="max-h-[500px]">
      <div className="space-y-2 p-2">
        <Collapsible open={nodesOpen} onOpenChange={setNodesOpen}>
          <CollapsibleTrigger className="flex items-center gap-2 w-full text-left font-medium text-sm py-1 px-2 bg-muted/50 rounded">
            {nodesOpen ? (
              <ChevronDown className="h-4 w-4" />
            ) : (
              <ChevronRight className="h-4 w-4" />
            )}
            Nodes ({nodeEntries.length})
          </CollapsibleTrigger>
          <CollapsibleContent>
            <div className="ml-2 mt-1 space-y-0.5">
              {nodeEntries.map(([key, node]) => (
                <NodeRow key={key} nodeKey={key} node={node} />
              ))}
              {nodeEntries.length === 0 && (
                <p className="text-xs text-muted-foreground px-2 py-1">
                  No nodes defined.
                </p>
              )}
            </div>
          </CollapsibleContent>
        </Collapsible>

        <Collapsible open={edgesOpen} onOpenChange={setEdgesOpen}>
          <CollapsibleTrigger className="flex items-center gap-2 w-full text-left font-medium text-sm py-1 px-2 bg-muted/50 rounded">
            {edgesOpen ? (
              <ChevronDown className="h-4 w-4" />
            ) : (
              <ChevronRight className="h-4 w-4" />
            )}
            Edges ({edges.length})
          </CollapsibleTrigger>
          <CollapsibleContent>
            <div className="ml-2 mt-1">
              {edges.map((edge, i) => (
                <EdgeRow key={i} edge={edge} idx={i} />
              ))}
              {edges.length === 0 && (
                <p className="text-xs text-muted-foreground px-2 py-1">
                  No edges defined.
                </p>
              )}
            </div>
          </CollapsibleContent>
        </Collapsible>
      </div>
    </ScrollArea>
  );
}
