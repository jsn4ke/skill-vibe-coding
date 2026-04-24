# Go Race Test Rule

After any successful `go run` or `go build` of the skill-go project, always run race detection tests to catch concurrency issues:

```bash
go test -race ./...
```

If no test files exist yet, at minimum verify the build is race-safe:

```bash
go build -race ./...
go run -race ./server/
```

Do not mark implementation as complete until race detection passes cleanly with no warnings.
