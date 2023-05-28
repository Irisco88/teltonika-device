
build: clean
     go build -o ./bin/teltonikasrv -ldflags="-s -w" cmd/*
run: build
    ./bin/teltonikasrv start --host 127.0.0.1
clean:
     @[ -d "./bin" ] && rm -r ./bin && echo "bin directory cleaned" || true