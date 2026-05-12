---
paths:
  - "services/**"
---

# Service lifecycle rules

## Signal handler — mandatory in every service

Every Go service `main()` MUST block until SIGTERM/SIGINT is received.
A `main()` that returns immediately causes an infinite restart loop in any
container runtime (Docker Compose, Kubernetes) with exit code 0, making the
problem invisible in logs.

Required skeleton for ALL services, including stubs:

```go
func main() {
    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
    defer stop()
    // initialisation ...
    log.Println("<service>: started")
    <-ctx.Done()
    log.Println("<service>: stopping")
    // graceful shutdown ...
}
```

- `func main() {}` is NEVER a valid stub. Use `<-ctx.Done()` as the minimum blocker.
- Goroutines inside a service must respect the context and exit when it is cancelled.

## Stub services

A stub service is one registered in docker-compose but not yet implemented.
It MUST have the signal handler skeleton above so it runs cleanly without
restart loops. Mark it clearly in the log line: `log.Println("<service>: started (stub)")`.

## Graceful shutdown checklist

When implementing a real service, always:
1. Pass the signal context to every long-running goroutine.
2. Close publishers, consumers, and DB connections in the shutdown block.
3. Add a shutdown timeout (e.g. `context.WithTimeout(context.Background(), 10*time.Second)`)
   so a stuck goroutine does not prevent the container from stopping.
