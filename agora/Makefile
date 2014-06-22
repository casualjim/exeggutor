GOPATH := $(CURDIR)/../_vendor:$(GOPATH)
GO ?= go

all: run-forever

build-frontend:
	@cd static; grunt build 

build-binary:
	$(shell GOOS=darwin $(GO) build -o $(shell basename $(CURDIR)))

build-linux:
	$(shell GOOS=linux CGO_ENABLED=1 CC_FOR_TARGET=gcc-4.8 CXX_FOR_TARGET=g++-4.8 $(GO) build -o $(shell basename $(CURDIR)).linux)

docker: build-linux build-frontend
	-rm -rf dist
	mkdir -p dist
	mv agora.linux dist
	vm static/build dist/static
	$(DOCKER) build -t $(shell basename $(CURDIR))-$(VERSION) .

dist: build-binary build-frontend
	mkdir -p dist
	mv agora dist
	mv static/build dist/static
	tar -czf agora.tar.gz agora
	rm -rf agora

upload: 
	scp agora.tar.gz mesos.helloreverb.com:~

clean:
	-rm -rf agora
	-rm -rf static/build

distclean: clean
	-rm -rf dist	
	-rm -rf agora.tar.gz	

run-forever: 
	gin

run:
	@$(GO) run main.go

test:
	@ginkgo -r 

