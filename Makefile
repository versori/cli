.PHONY: build
build:
	go build -o bin/ ./...

.PHONY: patch-api-spec
patch-api-spec:
	sed -i.bak 's|versori.dev/vergo/ulid|github.com/versori/cli/pkg/ulid|g' versori-api.yaml
	rm -f versori-api.yaml.bak

.PHONY: generate
generate: patch-api-spec
	go generate ./...
	go run scripts/add_copyright/main.go
	go run scripts/docs/main.go

.PHONY: lint
lint:
	go run scripts/add_copyright/main.go
	golangci-lint run

.PHONY: lint-fix
lint-fix:
	go run scripts/add_copyright/main.go
	golangci-lint run --fix

.PHONY: test
test:
	go test ./...

.PHONY: e2e
e2e:
	go test -v -tags=e2e ./test/e2e/...

.PHONY: cli
cli: # build the Versori CLI tool
	mkdir -p bin
	go build -o bin/versori -ldflags="-X 'github.com/versori/cli/pkg/cmd.version=$$(git describe --tags --abbrev=0 2>/dev/null || git rev-parse --short HEAD)'" .
	cp bin/versori /usr/local/bin/versori

.PHONY: build-all
build-all: # build binaries for all supported OSes (darwin, linux, windows)
	mkdir -p bin
	GOOS=darwin GOARCH=amd64 go build -o bin/versori-darwin-amd64 -ldflags="-X 'github.com/versori/cli/pkg/cmd.version=$$(git describe --tags --abbrev=0 2>/dev/null || git rev-parse --short HEAD)'" .
	GOOS=linux GOARCH=amd64 go build -o bin/versori-linux-amd64 -ldflags="-X 'github.com/versori/cli/pkg/cmd.version=$$(git describe --tags --abbrev=0 2>/dev/null || git rev-parse --short HEAD)'" .
	GOOS=windows GOARCH=amd64 go build -o bin/versori-windows-amd64.exe -ldflags="-X 'github.com/versori/cli/pkg/cmd.version=$$(git describe --tags --abbrev=0 2>/dev/null || git rev-parse --short HEAD)'" .

.PHONY: versori-docs
versori-docs:
	go run scripts/docs/main.go --prefix="" --out=../user-docs/latest/cli/commands --disable-md-ext

.PHONY: combine-skills
combine-skills:
	go run scripts/combine_skills/main.go

.PHONY: install-git-hooks
install-git-hooks:
	cp hooks/* .git/hooks/
