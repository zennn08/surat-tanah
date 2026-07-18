# SITANAH — build & dev tasks
# Catatan: deliverable HARUS bisa dibangun tanpa C compiler (CGO_ENABLED=0).
# Mesin dev ber-RAM kecil: pakai flag pembatas paralelisme saat compile sqlite.

BINARY   := surat-tanah.exe
# Versi ditanam ke binary (tampil di footer web). Override: make release VERSION=v1.0.1
VERSION  ?= $(shell git describe --tags --always 2>/dev/null || echo dev)
LOWMEM   := GOMAXPROCS=2 GOGC=30
LOWFLAGS := -p 1

.PHONY: dev release frontend generate tidy vet test clean

## dev: build binary konsol lokal (ada log terminal)
dev:
	$(LOWMEM) go build $(LOWFLAGS) -ldflags="-X main.version=$(VERSION)" -o $(BINARY) .

## release: deliverable Windows — tanpa jendela terminal, binary kecil, tanpa CGO
release: frontend
	$(LOWMEM) CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LOWFLAGS) -trimpath \
		-ldflags="-H windowsgui -s -w -X main.version=$(VERSION)" -o $(BINARY) .

## frontend: build Svelte -> frontend/dist (di-embed oleh Go)
frontend:
	cd frontend && yarn install && yarn build

## generate: jalankan sqlc dari schema.sql + queries.sql
generate:
	sqlc generate

tidy:
	go mod tidy

vet:
	go vet ./...

test:
	go test ./...

## clean: hapus artefak build & DB percobaan
clean:
	rm -f $(BINARY) surat-tanah.db surat-tanah.db-shm surat-tanah.db-wal
