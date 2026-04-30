# ADR-017: Paquetes como unidad de distribución

**Estado:** Aceptado
**Fecha:** 2026-04-20
**Autores:** [Equipo de arquitectura]

## Contexto

Dago necesita una unidad de distribución que agrupe todo lo necesario
para ejecutar un workflow: la definición del grafo, la configuración
de los nodos agénticos, las herramientas MCP requeridas, los
componentes de UI, y los metadatos de seguridad. Sin esta
formalización, cada workflow es una colección de artefactos sueltos
difíciles de versionar, compartir y reutilizar.

## Decisión

Se define el **paquete** como la unidad atómica de distribución del
sistema. Un paquete es un artefacto versionado que se publica en el
servicio catalog (ADR-013) y contiene todo lo necesario para instalar,
configurar y ejecutar un workflow.

### Estructura de un paquete

```json
{
  "$schema": "https://dago.dev/schemas/package.json",
  "package": {
    "id": "pkg_customer_support",
    "name": "Customer Support Workflow",
    "version": "2.1.0",
    "author": "team-support",
    "license": "MIT",
    "description": "Automated customer support with routing, escalation and resolution",
    "tags": ["department:support", "tier:production"],
    "dependencies": {
      "packages": [],
      "mcp_servers": ["zendesk-mcp", "jira-mcp"],
      "models": ["claude-sonnet-4-20250514"]
    }
  },

  "workflow": {
    "graph": {
      "entry_node": "classifier",
      "nodes": {
        "classifier": { "pattern": "router", "config": {} },
        "technical": { "pattern": "react", "config": {} },
        "billing": { "pattern": "llm_call", "config": {} },
        "review": { "pattern": "guardrail", "config": {} }
      },
      "edges": [
        {"type": "conditional", "from": "classifier", "conditions": []},
        {"type": "sequential", "from": "technical", "to": "review"},
        {"type": "sequential", "from": "billing", "to": "review"}
      ]
    },
    "memory": {
      "semantic_search": true,
      "episode_context": 3
    }
  },

  "skills": {
    "system_prompts": {
      "classifier": "You are a customer support classifier...",
      "technical": "You are a technical support agent..."
    },
    "few_shot_examples": [],
    "business_rules": [
      "Always escalate billing disputes over $500",
      "Never share internal ticket IDs with customers"
    ]
  },

  "tools": {
    "mcp_servers": [
      {
        "id": "zendesk-mcp",
        "required_tools": ["search_tickets", "update_ticket", "add_comment"]
      },
      {
        "id": "jira-mcp",
        "required_tools": ["create_issue", "assign_issue"]
      }
    ]
  },

  "ui": {
    "components": [
      {
        "id": "ticket_timeline",
        "type": "microfrontend",
        "module": "@dago-pkg/customer-support/TicketTimeline",
        "surface": "execution_detail",
        "props_schema": {
          "type": "object",
          "properties": {
            "executionId": { "type": "string", "format": "uuid" }
          }
        }
      }
    ],
    "a2ui_catalog": [
      {
        "id": "satisfaction_survey",
        "component_type": "form",
        "schema": {}
      }
    ],
    "ag_ui": {
      "streaming": true,
      "human_in_the_loop": true,
      "state_sync": true,
      "tool_visualization": true
    }
  }
}
```

### Secciones del paquete

**`package`** — Metadatos: id, nombre, versión (semver), autor,
licencia, descripción, tags ABAC para control de acceso, y
dependencias (otros paquetes, MCP servers, modelos LLM).

**`workflow`** — Definición del grafo (ADR-016): nodos con sus
patrones, aristas con sus flujos, configuración de memoria.
Es la spec de ejecución — el orchestrator la consume para ejecutar.

**`skills`** — Conocimiento del dominio: system prompts por nodo,
ejemplos few-shot, reglas de negocio. Es lo que diferencia un
agente genérico de uno especializado en un dominio concreto.

**`tools`** — Herramientas requeridas: MCP servers con las tools
específicas que cada nodo necesita. El orchestrator verifica
disponibilidad contra el mcp-registry antes de ejecutar.

**`ui`** — Componentes de interfaz: microfrontales que el dashboard
carga dinámicamente, componentes A2UI que los agentes pueden
solicitar durante la ejecución, y configuración AG-UI para la
comunicación en tiempo real.

### Versionado

Los paquetes siguen **Semantic Versioning** (semver):

- **Major** — Cambios incompatibles en el workflow o la API del paquete.
- **Minor** — Nuevos nodos, tools o componentes UI sin romper.
- **Patch** — Correcciones en prompts, reglas de negocio, configuración.

El catálogo mantiene todas las versiones. Las ejecuciones en curso
usan la versión con la que se iniciaron, no la última publicada.

### Validación

Al publicar un paquete, el catalog valida:

1. **Schema** — El JSON cumple el JSON Schema del paquete.
2. **Workflow** — El grafo es estructuralmente válido (ADR-016).
3. **Dependencias** — Los MCP servers referenciados existen en
   el mcp-registry. Los modelos LLM están disponibles.
4. **Seguridad** — Los tags ABAC son válidos. El autor tiene
   permisos para publicar con esos tags.
5. **UI** — Los componentes referenciados existen en el catálogo
   de microfrontales o son componentes A2UI válidos.

### Ciclo de vida

```
Draft → Published → Active → Deprecated → Archived
```

- **Draft** — En desarrollo. No ejecutable.
- **Published** — Disponible para ejecución. Inmutable.
- **Active** — Tiene ejecuciones en curso.
- **Deprecated** — Reemplazado por versión nueva. Las ejecuciones
  existentes continúan pero no se inician nuevas.
- **Archived** — Retirado. Solo consulta histórica.

## Notas para Claude Code

- El schema del paquete vive en `specs/schemas/package.json`.
- La validación de paquetes se implementa en `services/catalog/internal/`.
- El orchestrator obtiene la definición del paquete del catalog via
  HTTP antes de ejecutar un grafo.
- Los tags del paquete se evalúan con ABAC (ADR-012) para control
  de acceso.
- Los componentes de UI del paquete los carga el dashboard
  dinámicamente (ADR-019).
