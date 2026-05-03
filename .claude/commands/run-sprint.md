---
description: Ejecuta un sprint completo siguiendo el SDLC (ADR-005, ADR-020)
---

Ejecuta el ciclo SDLC completo para el sprint indicado en $ARGUMENTS.

## Paso 1: Preparar rama
Crear rama desde main según nomenclatura de ADR-005.

## Paso 2: Ejecutar TODOs
Actúa como @developer. Lee el documento de sprint en `docs/sprints/`.
Ejecuta los TODOs en el orden definido (ADR-020):
specs → contratos → tests (Red) → datos → implementación (Green) → refactor → docs.

Si un contrato no existe en `specs/contracts/`, crearlo o PREGUNTAR al usuario.
Cada TODO se commitea por separado con formato de ADR-005.

## Paso 3: Verificar calidad
Ejecutar todos los checks definidos en ADR-004 (goimports, golangci-lint, go test, go build).
Si algo falla, corregir antes de continuar.

## Paso 4: Review
Actúa como @reviewer. Revisa contra el documento de sprint (ADR-020).
Completar la sección "Resultado del sprint".
Si hay problemas bloqueantes: PARAR y pedir al usuario que decida.

## Paso 5: Push y PR
Push de la rama. Crear PR según ADR-005 (descripción, refs a sprint y ADRs, matriz de trazabilidad).
Usar `gh pr create` si disponible, si no dar instrucciones al usuario.

## Paso 6: Esperar CI
Informar al usuario que CI debe pasar (ADR-005). NO hacer merge — lo aprueba el usuario desde GitHub.

## Paso 7: Post-merge
Solo cuando el usuario confirme que el PR se mergeó:
Limpiar rama, actualizar `docs/log.md` y `docs/index.md`.
Si hay observaciones menores, propagarlas al siguiente sprint (ADR-020).
