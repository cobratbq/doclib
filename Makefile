SUFFIXES :=
MAKEFLAGS += --no-builtin-rules --no-builtin-variables

BUILDARGS := -tags gles,tracelog

.PHONY: clean build run

build:
	go build -v $(BUILDARGS) ./cmd/...

run:
	go run -v $(BUILDARGS) ./cmd/doclib/main.go

clean:
	rm -f doclib
