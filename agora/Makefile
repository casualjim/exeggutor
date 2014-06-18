GOPATH := $(CURDIR)/../_vendor:$(GOPATH)
GO ?= go

all: run-forever

build-binary: 
	@go build

build-frontend:
	@cd static; grunt build 

dist: build-binary build-frontend
	mkdir -p dist
	mv agora dist
	mv static/build dist/static
	mv dist agora
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
	@go run main.go

test:
	@ginkgo -r 

