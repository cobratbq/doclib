SUFFIXES :=
MAKEFLAGS += --no-builtin-rules --no-builtin-variables

.PHONY: clean build release run

build:
	go build -v -buildmode=pie -tags gles,tracelog ./cmd/doclib
	go build -v -buildmode=pie -tags gles,tracelog ./cmd/doccli

release:
	go build -v -buildmode=pie -tags gles ./cmd/doclib
	go build -v -buildmode=pie -tags gles ./cmd/doccli

run:
	go run $(BUILDARGS) ./cmd/doclib/main.go

clean:
	rm -f doclib doccli
