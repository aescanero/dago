# ADR-008: Valkey for events, cache and sessions

**Status:** Accepted (revised: Redis → Valkey)
**Date:** 2026-04-20
**Authors:** [Architecture team]

## Context

The system needs lightweight messaging/events, cache and session storage.
An open source solution without license restrictions is sought that covers
all three needs.

## Decision

**Valkey** (Redis fork maintained by the Linux Foundation,
BSD-3 license) is adopted as the unified solution for events (Pub/Sub and Streams),
cache and session storage. Valkey is 100% compatible with the
Redis API — the Go client is the same (`github.com/redis/go-redis/v9`,
which supports Valkey natively).

### Motivation for switching from Redis to Valkey

Redis 8+ adopted a dual RSALv2+SSPLv1 license that restricts certain commercial
uses. Valkey is the community fork under the Linux Foundation with
BSD-3 license, backed by AWS, Google, Oracle, Ericsson and others.
100% compatible API — transparent change.

### Three responsibilities, one system

```
Valkey
├── Pub/Sub & Streams   → Events between services (ADR-011, ADR-014)
├── Key-Value + TTL     → Cache of frequently accessed data
└── Hash + TTL          → User sessions
```

### Concrete rules

The rules are identical to those originally defined for Redis:

1. **Business events:** Streams + consumer groups (Event-Carried State).
2. **Ephemeral signals:** Pub/Sub (Event Notification).
3. **Cache:** TTL mandatory. Naming: `cache:{entity}:{id}`. Cache-aside.
4. **Sessions:** Hash with sliding expiration. Opaque token (crypto/rand).
5. **Separate logical DBs:** DB 0 cache, DB 1 sessions, DB 2 events.
6. **Go client:** `github.com/redis/go-redis/v9` (compatible with Valkey).

## Notes for Claude Code

- Use `github.com/redis/go-redis/v9` as the client. It works with Valkey.
- In docker-compose, use the `valkey/valkey:8` image instead of `redis`.
- Every key must have a TTL. Naming: `{type}:{entity}:{id}`.
- Adapters in `adapters/eventbus/` and `adapters/storage/`.
