export const DEFINITION_TEMPLATES = [
  {
    id: "empty",
    label: "Grafo vacío",
    definition: { nodes: {}, edges: [] },
  },
  {
    id: "llm_call",
    label: "LLM Call simple",
    definition: {
      nodes: {
        llm: {
          pattern: "llm_call",
          config: {
            model: "claude-sonnet-4-6",
            system_prompt: "You are a helpful assistant.",
            temperature: 0.7,
            max_tokens: 2048,
          },
        },
      },
      edges: [],
    },
  },
  {
    id: "react_agent",
    label: "React agent",
    definition: {
      nodes: {
        agent: {
          pattern: "react",
          config: {
            model: "claude-sonnet-4-6",
            system_prompt: "You are an agent that reasons step by step.",
            tools: [],
            max_iterations: 10,
            timeout_seconds: 120,
          },
        },
      },
      edges: [],
    },
  },
  {
    id: "router_handlers",
    label: "Router + handlers",
    definition: {
      nodes: {
        router: {
          pattern: "router",
          config: {
            mode: "llm",
            llm_fallback: {
              model: "claude-sonnet-4-6",
              routes: ["handler_a", "handler_b"],
            },
          },
        },
        handler_a: {
          pattern: "llm_call",
          config: { model: "claude-sonnet-4-6" },
        },
        handler_b: {
          pattern: "llm_call",
          config: { model: "claude-sonnet-4-6" },
        },
      },
      edges: [
        {
          type: "conditional",
          from: "router",
          conditions: [
            {
              expression: "output.route == 'handler_a'",
              target: "handler_a",
            },
            {
              expression: "output.route == 'handler_b'",
              target: "handler_b",
            },
          ],
        },
      ],
    },
  },
] as const;

export type TemplateId = (typeof DEFINITION_TEMPLATES)[number]["id"];

export function templateToDefinitionString(id: TemplateId): string {
  const template = DEFINITION_TEMPLATES.find((t) => t.id === id);
  if (!template) return "{}";
  return JSON.stringify(template.definition, null, 2);
}
