# show help by default
default:
    @just --list --justfile {{ justfile() }}

# update go deps
update *flags:
    go get {{ flags }} ./cmd
    go mod tidy
    go mod vendor

# run linter
lint:
    golangci-lint run ./...

# automatically fix liter issues
lintfix:
    golangci-lint run --fix ./...

# generate mocks
mocks:
    @mockery --all --inpackage --testonly --exclude vendor

# run unit tests
test packages="./...":
    @go test -cover -coverprofile=cover.out -coverpkg={{ packages }} -covermode=set {{ packages }}
    @go tool cover -func=cover.out
    -@rm -f cover.out

# run app
run:
    @echo "[WARNING] you may want to 'just dryrun' first"
    @go run ./cmd

# run app in dryrun mode
dryrun:
    @go run ./cmd -dryrun


# build app
build:
    CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' -tags timetzdata,goolm -v -o suae ./cmd
