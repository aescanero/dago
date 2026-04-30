# ADR-008: Valkey para eventos, caché y sesiones

**Estado:** Aceptado (revisado: Redis → Valkey)
**Fecha:** 2026-04-20
**Autores:** [Equipo de arquitectura]

## Contexto

El sistema necesita mensajería/eventos ligera, caché y almacenamiento
de sesiones. Se busca una solución open source sin restricciones de
licencia que cubra las tres necesidades.

## Decisión

Se adopta **Valkey** (fork de Redis mantenido por la Linux Foundation,
licencia BSD-3) como solución unificada para eventos (Pub/Sub y Streams),
caché y almacenamiento de sesiones. Valkey es 100% compatible con la
API de Redis — el cliente Go es el mismo (`github.com/redis/go-redis/v9`,
que soporta Valkey nativamente).

### Motivación del cambio de Redis a Valkey

Redis 8+ adoptó licencia dual RSALv2+SSPLv1 que restringe ciertos usos
comerciales. Valkey es el fork comunitario bajo la Linux Foundation con
licencia BSD-3, respaldado por AWS, Google, Oracle, Ericsson y otros.
API 100% compatible — cambio transparente.

### Tres responsabilidades, un sistema

```
Valkey
├── Pub/Sub & Streams   → Eventos entre servicios (ADR-011, ADR-014)
├── Key-Value + TTL     → Caché de datos frecuentes
└── Hash + TTL          → Sesiones de usuario
```

### Reglas concretas

Las reglas son idénticas a las definidas originalmente para Redis:

1. **Eventos de negocio:** Streams + consumer groups (Event-Carried State).
2. **Señales efímeras:** Pub/Sub (Event Notification).
3. **Caché:** TTL obligatorio. Naming: `cache:{entidad}:{id}`. Cache-aside.
4. **Sesiones:** Hash con sliding expiration. Token opaco (crypto/rand).
5. **DBs lógicas separadas:** DB 0 caché, DB 1 sesiones, DB 2 eventos.
6. **Cliente Go:** `github.com/redis/go-redis/v9` (compatible con Valkey).

## Notas para Claude Code

- Usar `github.com/redis/go-redis/v9` como cliente. Funciona con Valkey.
- En docker-compose, usar imagen `valkey/valkey:8` en vez de `redis`.
- Toda clave con TTL. Naming `{tipo}:{entidad}:{id}`.
- Adaptadores en `adapters/eventbus/` y `adapters/storage/`.
