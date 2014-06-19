VERSION=0.0.1

all: go-build docker-build 

go-build:
	go build

docker-build:
	docker build -t $(shell basename $(CURDIR))-$(VERSION) .

test:
	go test
