BIN=edgeclaw
DIST=dist
PKG=.

.PHONY: build run clean all
build:           ## native build
	go build -o $(BIN) $(PKG)
run: build
	./$(BIN)
clean:
	rm -rf $(BIN) $(DIST)

## all: cross-compile static binaries for every OS/arch (pure Go, no CGo)
all:
	@mkdir -p $(DIST)
	GOOS=linux   GOARCH=amd64           go build -o $(DIST)/$(BIN)-linux-amd64    $(PKG)
	GOOS=linux   GOARCH=arm64           go build -o $(DIST)/$(BIN)-linux-arm64    $(PKG)
	GOOS=linux   GOARCH=arm GOARM=7     go build -o $(DIST)/$(BIN)-linux-armv7    $(PKG)
	GOOS=linux   GOARCH=arm GOARM=6     go build -o $(DIST)/$(BIN)-linux-armv6    $(PKG)   # Raspberry Pi Zero
	GOOS=linux   GOARCH=riscv64         go build -o $(DIST)/$(BIN)-linux-riscv64  $(PKG)
	GOOS=linux   GOARCH=loong64         go build -o $(DIST)/$(BIN)-linux-loong64  $(PKG)
	GOOS=linux   GOARCH=mips64le        go build -o $(DIST)/$(BIN)-linux-mips64le $(PKG)
	GOOS=darwin  GOARCH=arm64           go build -o $(DIST)/$(BIN)-darwin-arm64   $(PKG)
	GOOS=darwin  GOARCH=amd64           go build -o $(DIST)/$(BIN)-darwin-amd64   $(PKG)
	GOOS=windows GOARCH=amd64           go build -o $(DIST)/$(BIN)-windows-amd64.exe $(PKG)
	GOOS=windows GOARCH=arm64           go build -o $(DIST)/$(BIN)-windows-arm64.exe $(PKG)
	@ls -lh $(DIST)
