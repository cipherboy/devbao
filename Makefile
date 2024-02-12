GO?=go

.PHONY: bin
bin:
	rm -f bin/devbao
	mkdir -p bin/
	$(GO) build -o bin/devbao github.com/cipherboy/devbao/cmd/devbao
	ls -lh bin/devbao
