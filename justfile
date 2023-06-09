composeFile := "docker-compose.yaml"
composeEnvFile := "compose.env"

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

upx: build
    upx --best --lzma bin/teltonikasrv

# run docker compose up
dcompose-up:
    @echo "run docker compose up"
    docker compose -f {{composeFile}} --env-file {{composeEnvFile}} up -d
    @echo "env variables are:"
    @cat compose.env

# stop docker compose containers
dcompose-stop:
    docker compose -f {{composeFile}} --env-file {{composeEnvFile}} stop

# down and clean all compose file containers
dcompose-clean:
    docker compose -f {{composeFile}} --env-file {{composeEnvFile}} down --volumes --remove-orphans --rmi local
