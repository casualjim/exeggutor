VERSION=0.0.1
GO ?= go
DOCKER ?= docker

all: docker

build:
	$(GO) build -o $(shell basename $(CURDIR)-$(VERSION))

build-linux:
	$(shell GOOS=linux $(GO) build -o $(shell basename $(CURDIR)).linux)

docker: build-linux
	$(DOCKER) build -t $(shell basename $(CURDIR))-$(VERSION) .

test:
	$(GO) test

clean:
	-rm -f $(shell basename $(CURDIR))*
