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

### import

interactive: true

```bash
go run ./cmd/ragserver/ import --collection "entities" --expand "contacts,dependsOn,contributesTo,tags"
```

### query-context

interactive: true

```bash
go run ./cmd/ragserver query -q "What is the plan to destroy the Death Star?"
```

### query-nocontext

interactive: true

```bash
go run ./cmd/ragserver query --no-context -q "What is the plan to destroy the Death Star?"
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

### docker-build-aarch64

```bash
nix build .#packages.aarch64-linux.docker-image
```

### docker-build-x86_64

```bash
nix build .#packages.x86_64-linux.docker-image
```

### crane-push-app

env: CONTAINER_REGISTRY=ghcr.io/ragserver

```bash
nix build .#packages.x86_64-linux.docker-image
cp ./result /tmp/ragserver.tar.gz
gunzip -f /tmp/ragserver.tar.gz
crane push /tmp/ragserver.tar ${CONTAINER_REGISTRY}/ragserver:v0.0.1
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

### docker-build-rqlite-aarch64

```bash
nix build .#packages.aarch64-linux.rqlite-docker-image
```

### docker-build-rqlite-x86_64

```bash
nix build .#packages.x86_64-linux.rqlite-docker-image
```

### crane-push-rqlite

env: CONTAINER_REGISTRY=ghcr.io/ragserver

```bash
nix build .#packages.x86_64-linux.rqlite-docker-image
cp ./result /tmp/rqlite.tar.gz
gunzip -f /tmp/rqlite.tar.gz
crane push /tmp/rqlite.tar ${CONTAINER_REGISTRY}/rqlite:v0.0.1
```

### docker-load-rqlite

Once you've built the image, you can load it into a local Docker daemon with `docker load`.

```bash
docker load < result
```

### docker-run-rqlite

```bash
docker run -v "$PWD/auth.json:/mnt/rqlite/auth.json" -v "$PWD/.rqlite:/mnt/data" -p 4001:4001 -p 4002:4002 -p 4003:4003 rqlite:latest
```

### k8s-create-namespace

```bash
kubectl create namespace ragserver
```

### k8s-create-secret

Need to create auth.json as a k8s secret.

```bash
kubectl -n ragserver create secret generic rqlite-auth --from-file=auth.json
```

### k8s-local-create-volume

Local k8s requires a volume to store data. In a cloud provider, this will likely already exists.

```bash
envsubst < k8s/local/volume.yaml | kubectl --namespace ragserver apply -f -
```

### k8s-apply

env: CONTAINER_REGISTRY=ghcr.io/ragserver
dir: k8s
interactive: true

```bash
for f in *.yaml; do envsubst < $f | kubectl apply --namespace ragserver -f -; done
```

### k8s-local-expose-ports

Once the application is deployed, we need to expose the ports in the k8s pods to the local machine. After this, the application will be available at `localhost:9020`.

```bash
kubectl port-forward service/ragserver 9020:9020 -n ragserver
```

### k8s-get-logs-ragserver

```bash
kubectl logs -n ragserver -l service=ragserver
```

### k8s-get-logs-rqlite

```bash
kubectl logs -n ragserver -l service=rqlite
```
