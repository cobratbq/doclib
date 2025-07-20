SUFFIXES :=
MAKEFLAGS += --no-builtin-rules --no-builtin-variables

BUILDARGS := -v -tags gles,tracelog

.PHONY: clean build run

build:
	go build $(BUILDARGS) ./cmd/doclib
	go build $(BUILDARGS) ./cmd/doccli

run:
	go run $(BUILDARGS) ./cmd/doclib/main.go

clean:
	rm -f doclib doccli
