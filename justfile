# build project
build: clean
     go build -o ./bin/teltonikasrv -ldflags="-s -w" cmd/*

# build & run server
run: build
    ./bin/teltonikasrv start --host 127.0.0.1

# clean build directory
clean:
     @[ -d "./bin" ] && rm -r ./bin && echo "bin directory cleaned" || true

# build and compress binary
upx: build
    upx --best --lzma bin/teltonikasrv

#build docker image
image tag:
    docker buildx build --build-arg GITHUB_TOKEN="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" --tag {{tag}} .