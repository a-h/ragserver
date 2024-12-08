# ragserver

## Tasks

### db-run

```bash
rqlited -auth=auth.json -extensions-path="${SQLITE_VEC_PATH}" ~/ragserver
```

### db-migration-create

```bash
migrate create -ext sql -dir db/migrations -seq create_documents_table
```

### serve

```bash
go run ./cmd/ragserver/ serve
```

### ollama-serve

```bash
ollama serve
```

### gomod2nix-update

```bash
gomod2nix
```

### build

```bash
nix build
```

### run

```bash
nix run
```

### develop

```bash
nix develop
```

### docker-build

```bash
nix build .#docker-image
```

### docker-load

Once you've built the image, you can load it into a local Docker daemon with `docker load`.

```bash
docker load < result
```

### docker-run

```bash
docker run -p 8080:8080 app:latest
```
