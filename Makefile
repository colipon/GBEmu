BIN := gbemu
PKG := ./cmd/gbemu

.PHONY: build run tidy clean

build:
	go build -o $(BIN) $(PKG)

run: build
	./$(BIN) $(ROM)

tidy:
	go mod tidy

clean:
	rm -f $(BIN)

# Build a WebAssembly version (playable in browser, no desktop needed)
wasm:
	GOOS=js GOARCH=wasm go build -o web/gbemu.wasm $(PKG)
	cp "$$(go env GOROOT)/misc/wasm/wasm_exec.js" web/

# Check that all packages compile
check:
	go build ./...
