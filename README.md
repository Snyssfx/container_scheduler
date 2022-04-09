# container_scheduler
container_scheduler is a service for launching Docker containers with slow calculations and proxying user requests.

## Problem
Schedule Docker containers with heavy calculations, slow initialization and save as many resources as possible.  
As an example, there is a bio-informatic container with image `quay.io/milaboratory/qual-2021-devops-server`.

## Architecture
- `ContainersMap` holds a mapping of seeds to `CachedDeduplicator`'s;
- `CachedDeduplicator` holds a cache for a `RequestDeduplicator`;
- `RequestDeduplicator` deduplicates user requests and pass an input for a calculation to a `Qual` one by one;
- `Qual` is a container that starts and initializes `quay` docker container, pass calculations to it and stops it after the last request and the given time.

## Testing
- `make test`
- `make container_scheduler`
- `make lint`
- `go run ./cmd/main.go -port 9002`

```bash
go run ./cmd/main.go -port 9002 2>&1
curl 0.0.0.0:9002/calculate/1234/3 -v
curl 0.0.0.0:9002/calculate/1234/3 -v # check that cache works
```

## TODO
- add hard limits and eviction strategy for a cache.
- if we need metrics, we can add Requests, Errors, Durations in `/internal/api/calculate.go`.
- add limit of maximum containers count in `containersmap`
