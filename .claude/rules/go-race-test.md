# Go Test Rule

After any code change to the skill-go project, verify with:

```bash
gofmt -w . && go build ./... && go test ./... -count=1
```

Do not mark implementation as complete until all three steps pass:
1. `gofmt` — no formatting issues
2. `go build` — compiles cleanly
3. `go test` — all tests pass

On Windows, race detection (`-race`) may fail with STATUS_DLL_NOT_FOUND. In that case, run without `-race` and note it. On other platforms, prefer `go test -race ./...`.
