import { useState } from "react";
import { useNavigate, Link } from "react-router-dom";
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
import { GraphForm } from "../components/GraphForm";
import { useCreateGraph } from "../hooks/useCreateGraph";
import { formToGraphInput } from "../schemas/graph-form.schema";
import type { GraphFormValues } from "../schemas/graph-form.schema";

export function GraphCreatePage() {
  const navigate = useNavigate();
  const { mutateAsync, isPending } = useCreateGraph();
  const [fieldErrors, setFieldErrors] = useState<string[]>([]);

  async function handleSubmit(values: GraphFormValues) {
    setFieldErrors([]);
    try {
      const graph = await mutateAsync(formToGraphInput(values));
      toast.success("Graph created");
      navigate(`/graphs/${graph.id}`);
    } catch (err) {
      const error = err as Error & {
        status?: number;
        body?: { message?: string; details?: Record<string, string[]> };
      };
      if (error.status === 409) {
        toast.error(
          error.body?.message ??
            `Version ${values.version} of "${values.name}" already exists`
        );
      } else if (error.status === 422 && error.body?.details) {
        const errors = Object.values(error.body.details).flat();
        setFieldErrors(errors);
      } else {
        toast.error(error.message ?? "Error creating graph");
      }
    }
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
            <BreadcrumbPage>New graph</BreadcrumbPage>
          </BreadcrumbItem>
        </BreadcrumbList>
      </Breadcrumb>

      <h1 className="text-2xl font-bold">Create graph</h1>

      {fieldErrors.length > 0 && (
        <Alert variant="destructive">
          <AlertTitle>Validation errors</AlertTitle>
          <AlertDescription>
            <ul className="list-disc pl-4 space-y-1">
              {fieldErrors.map((err, i) => (
                <li key={i}>{err}</li>
              ))}
            </ul>
          </AlertDescription>
        </Alert>
      )}

      <GraphForm
        onSubmit={handleSubmit}
        isPending={isPending}
        submitLabel="Create"
      />
    </div>
  );
}
