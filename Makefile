GOPATH := $(CURDIR)/_vendor:$(GOPATH)
GO ?= go

setup: 
	@godep restore

test: 
	@ginkgo -r
