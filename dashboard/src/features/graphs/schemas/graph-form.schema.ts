import { z } from "zod";
import type { GraphInput } from "@/api/types.gen";

export const graphFormSchema = z.object({
  name: z.string().min(1, "Nombre requerido").max(255),
  version: z.string().regex(/^\d+\.\d+\.\d+$/, "Formato: 1.0.0"),
  description: z.string().max(1000).optional(),
  entry_node: z.string().min(1, "Nodo de entrada requerido").max(255),
  definition: z
    .string()
    .min(2, "Definición requerida")
    .refine(
      (v) => {
        try {
          JSON.parse(v);
          return true;
        } catch {
          return false;
        }
      },
      { message: "JSON inválido" }
    ),
  memory_semantic_search: z.boolean().default(false),
  memory_episode_context: z.coerce.number().int().min(0).default(0),
});

export type GraphFormValues = z.infer<typeof graphFormSchema>;

export function formToGraphInput(values: GraphFormValues): GraphInput {
  return {
    name: values.name,
    version: values.version,
    description: values.description,
    entry_node: values.entry_node,
    definition: JSON.parse(values.definition) as Record<string, unknown>,
    memory_config: {
      semantic_search: values.memory_semantic_search,
      episode_context: values.memory_episode_context,
    },
  };
}
