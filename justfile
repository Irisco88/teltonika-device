# build project
build: clean
     go build -o ./bin/teltonikasrv -ldflags="-s -w" cmd/*

# build & run server
run: build
    ./bin/teltonikasrv start --host 127.0.0.1

# clean build directory
clean:
     @[ -d "./bin" ] && rm -r ./bin && echo "bin directory cleaned" || true

# generate proto
proto:
    @echo "run proto linter..."
    @cd proto && buf lint && cd -
    @echo "format proto..."
    @cd proto && buf format -w && cd -
    @echo "generate proto..."
    @cd proto && buf generate && cd -
