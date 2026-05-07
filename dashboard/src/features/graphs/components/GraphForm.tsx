import { useForm, type DefaultValues } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Loader2, ChevronsUpDown } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/components/ui/switch";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  graphFormSchema,
  type GraphFormValues,
} from "../schemas/graph-form.schema";
import {
  DEFINITION_TEMPLATES,
  templateToDefinitionString,
  type TemplateId,
} from "../lib/definition-templates";

interface GraphFormProps {
  defaultValues?: Partial<GraphFormValues>;
  onSubmit: (values: GraphFormValues) => Promise<void>;
  isPending: boolean;
  submitLabel: string;
}

export function GraphForm({
  defaultValues,
  onSubmit,
  isPending,
  submitLabel,
}: GraphFormProps) {
  const form = useForm<GraphFormValues>({
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    resolver: zodResolver(graphFormSchema) as any,
    defaultValues: {
      name: "",
      version: "",
      description: "",
      entry_node: "",
      definition: JSON.stringify({ nodes: {}, edges: [] }, null, 2),
      memory_semantic_search: false,
      memory_episode_context: 0,
      ...(defaultValues as DefaultValues<GraphFormValues>),
    },
  });

  function handleTemplateSelect(id: string) {
    form.setValue(
      "definition",
      templateToDefinitionString(id as TemplateId),
      { shouldValidate: false }
    );
  }

  return (
    <Form {...form}>
      <form
        onSubmit={form.handleSubmit(onSubmit)}
        className="space-y-4 max-w-2xl"
      >
        <FormField
          control={form.control}
          name="name"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Name</FormLabel>
              <FormControl>
                <Input placeholder="my-graph" {...field} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />

        <FormField
          control={form.control}
          name="version"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Version</FormLabel>
              <FormControl>
                <Input placeholder="1.0.0" {...field} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />

        <FormField
          control={form.control}
          name="description"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Description (optional)</FormLabel>
              <FormControl>
                <Textarea
                  placeholder="Describe what this graph does..."
                  rows={2}
                  {...field}
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />

        <FormField
          control={form.control}
          name="entry_node"
          render={({ field }) => (
            <FormItem>
              <TooltipProvider>
                <div className="flex items-center gap-2">
                  <FormLabel>Entry node</FormLabel>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <span className="cursor-help text-muted-foreground text-xs">[?]</span>
                    </TooltipTrigger>
                    <TooltipContent>
                      Key of the node where execution begins
                    </TooltipContent>
                  </Tooltip>
                </div>
              </TooltipProvider>
              <FormControl>
                <Input placeholder="llm" {...field} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />

        <div className="space-y-2">
          <label className="text-sm font-medium" aria-label="Template">
            Template
          </label>
          <Select onValueChange={handleTemplateSelect}>
            <SelectTrigger aria-label="Template">
              <SelectValue placeholder="Select a template..." />
            </SelectTrigger>
            <SelectContent>
              {DEFINITION_TEMPLATES.map((t) => (
                <SelectItem key={t.id} value={t.id}>
                  {t.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <FormField
          control={form.control}
          name="definition"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Definition</FormLabel>
              <FormControl>
                <Textarea
                  className="font-mono"
                  rows={12}
                  aria-label="Definition"
                  {...field}
                  onBlur={(e) => {
                    field.onBlur();
                    form.trigger("definition");
                    if (e.target.value) {
                      field.onChange(e.target.value);
                    }
                  }}
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />

        <Collapsible>
          <CollapsibleTrigger asChild>
            <Button variant="ghost" size="sm" type="button" className="gap-2">
              <ChevronsUpDown className="h-4 w-4" />
              Memory
            </Button>
          </CollapsibleTrigger>
          <CollapsibleContent className="space-y-4 pt-2">
            <FormField
              control={form.control}
              name="memory_semantic_search"
              render={({ field }) => (
                <FormItem className="flex items-center gap-3">
                  <FormControl>
                    <Switch
                      checked={field.value}
                      onCheckedChange={field.onChange}
                    />
                  </FormControl>
                  <FormLabel className="!mt-0">Semantic search</FormLabel>
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="memory_episode_context"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Episodic context (episodes)</FormLabel>
                  <FormControl>
                    <Input type="number" min={0} {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </CollapsibleContent>
        </Collapsible>

        <Button type="submit" disabled={isPending}>
          {isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
          {submitLabel}
        </Button>
      </form>
    </Form>
  );
}
