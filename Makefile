GO?=go

.PHONY: bin
bin:
	rm -f bin/devbao
	mkdir -p bin/
	$(GO) build -o bin/devbao github.com/cipherboy/devbao/cmd/devbao
	ls -lh bin/devbao

.PHONY: fmt
fmt:
	$(GO) run mvdan.cc/gofumpt@latest -w -l $$(find . -name "*.go")
