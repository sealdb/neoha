# pkg/

This directory is reserved for **public Go packages** that other modules may import.

NeoHA is primarily an application (`cmd/neoha`, `cmd/neohactl`). Most code lives under `internal/` and is not intended for external use.

Add packages here only when you deliberately expose a stable API for other projects, for example:

- a reusable client library
- shared types consumed by operators or sidecars

Until then, leave this directory empty (except this README).
