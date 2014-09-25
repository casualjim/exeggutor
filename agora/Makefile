GOPATH := $(GOPATH)
GO ?= go
VERSION ?= 0.0.1
DOCKER ?= docker

all: run-forever

build-frontend:
	-@cd static; grunt compile

build-binary:
	$(shell GOOS=darwin $(GO) build -o $(shell basename $(CURDIR)))

build-linux:
	$(shell GOOS=linux CGO_ENABLED=1 $(GO) build -o $(shell basename $(CURDIR)).linux)

docker: build-linux build-frontend
	-rm -rf dist
	mkdir -p dist
	mv agora.linux dist
	mv static/bin dist/static
	cp /usr/local/lib/libmesos.la /usr/local/lib/libmesos-0.19.0-rvb3.so dist
	cp Dockerfile dist
	$(DOCKER) build -t dev-docker.helloreverb.com/library/$(shell basename $(CURDIR)):$(VERSION) dist

dist: build-binary build-frontend
	-rm -rf dist
	mkdir -p dist
	mv agora dist
	mv static/bin dist/static
	tar -czf agora.tar.gz agora
	rm -rf agora

upload: 
	scp agora.tar.gz mesos.dev.helloreverb.com:~

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

