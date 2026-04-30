---
description: Planificar un sprint a partir de un requisito
---

Actúa como el agente **planner**. Analiza el requisito proporcionado en $ARGUMENTS
contra los ADRs en `docs/adr/` y las specs en `specs/`.

Genera un documento de sprint completo en `docs/sprints/SPRINT-NNN-descripcion.md`
siguiendo el formato definido en ADR-020:

1. Analiza qué servicios, specs y entidades se ven afectados.
2. Define objetivo y alcance (incluido y excluido).
3. Para cada operación a implementar, verifica si existe un contrato de
   comportamiento en `specs/contracts/`. Si no existe:
   - Intenta generarlo a partir de los ADRs y specs disponibles.
   - Si el comportamiento no está claro, PREGUNTA al usuario antes de continuar.
   - No generar tests sin un contrato que los fundamente.
4. Genera TODOs en orden: specs → contratos → tests → datos (Ent) → implementación → docs.
5. Cada TODO indica: agente, skill, ubicación, qué verifica/implementa, dependencias.
6. Construye la matriz de trazabilidad (spec → contrato → test → implementación).
7. Estima si cabe en un sprint reducido (1-3 días) o hay que dividir.

El sprint debe ser autocontenido y verificable.

Al finalizar la planificación:
1. Añade una entrada al log en docs/log.md: ## [YYYY-MM-DD] sprint | SPRINT-NNN: Título
2. Actualiza docs/index.md si se crean nuevos artefactos.
