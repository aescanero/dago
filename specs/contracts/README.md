# Contratos de comportamiento

Los contratos de comportamiento definen **qué hace cada operación**
bajo cada condición posible. Son la pieza que conecta las specs
(estructura) con los tests (verificación) y la implementación (código).

## Cuándo se crean

Los contratos se crean orgánicamente durante los sprints, justo antes
de los tests. El orden dentro del sprint es:

```
specs (estructura) → contratos (comportamiento) → tests (verificación) → implementación (código)
```

Si un agente necesita implementar una operación y no existe su contrato,
debe crearlo primero. Si el comportamiento no está claro, debe preguntar
al usuario.

## Formato

Cada contrato es un fichero Markdown con el nombre de la operación:

```
specs/contracts/
├── execute-graph.md
├── create-package.md
├── authenticate-user.md
├── invoke-mcp-tool.md
└── ...
```

### Estructura del contrato

```markdown
# Contrato: [NombreOperación]

## Trigger
[Cómo se invoca: endpoint HTTP, evento, proceso background]

## Precondiciones
[Qué debe ser cierto ANTES de ejecutar. Incluye validación de negocio,
no solo de formato — eso lo cubre el schema.]

## Postcondiciones (éxito)
[Qué es cierto DESPUÉS de ejecutar con éxito. Qué cambió en el sistema:
estado en DB, eventos publicados, notificaciones enviadas.]

## Casos de error
[Cada condición de error con:
- Qué la causa
- Qué devuelve (código, error response)
- Qué NO cambia en el sistema (rollback, sin side effects)]

## Invariantes
[Qué debe ser cierto SIEMPRE, antes y después de cualquier operación.]

## Ejemplo
[Escenario concreto con datos de ejemplo.]

## Refs
[ADRs, specs y sprints relacionados.]
```

## Relación con otros artefactos

```
ADR (por qué)
  → Spec (qué forma tiene)
    → Contrato (qué hace)
      → Test (lo verifica)
        → Código (lo implementa)
```

Los tests se derivan directamente del contrato:
- Cada precondición genera un test negativo ("should fail when...").
- Cada postcondición genera un test positivo ("should create... when...").
- Cada caso de error genera un test de error ("should return 403 when...").
- Cada invariante genera un test de consistencia.
