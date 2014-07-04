GO ?= go
DEVENV_DEPS = github.com/golang/lint \
	code.google.com/p/go.tools/cmd/gotype \
	code.google.com/p/go.tools/cmd/vet \
	code.google.com/p/go.tools/cmd/godoc \
	code.google.com/p/go.tools/cmd/goimports \
	code.google.com/p/go.tools/cmd/oracle \
	code.google.com/p/go.tools/cmd/cover \
	code.google.com/p/rog-go/exp/cmd/godef \
	github.com/axw/gocov/gocov \
	gopkg.in/matm/v1/gocov-html \
	github.com/AlekSi/gocov-xml \
	github.com/nsf/gocode \
	github.com/golang/lint/golint \
	github.com/kisielk/errcheck \
	github.com/jstemmer/gotags \
	code.google.com/p/goprotobuf/proto \
	code.google.com/p/goprotobuf/protoc-gen-go \
	github.com/tools/godep

devenv: 
	@$(foreach dir,$(DEVENV_DEPS),echo "installing $(dir)" && go get $(dir);)

update-all:
	$(GO) get -u ./...

setup: 	
	@godep restore

test: 
	@go test -v ./...

dist:

