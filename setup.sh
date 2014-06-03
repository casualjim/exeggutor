#!/bin/sh

if [ ! -f `which hub` ]; then
  echo "You need to install the hub command for this script to work"
  echo "Try executing brew install hub"
  exit 1
fi

go get github.com/onsi/ginkgo/ginkgo
go get github.com/onsi/gomega
go get code.google.com/p/gomock/gomock
go get code.google.com/p/gomock/mockgen
go get github.com/qur/withmock
go get github.com/samuel/go-zookeeper/zk
go get github.com/jessevdk/go-flags
go get github.com/rcrowley/go-metrics
go get github.com/yvasiyarov/gorelic
go get github.com/wendal/mustache
go get github.com/op/go-logging
go get github.com/ludmiloff/revel
go get github.com/revel/cmd/revel
go get github.com/hishboy/gocommons/lang
go get github.com/armon/gomdb
go get github.com/hashicorp/go-version
go get github.com/sdming/gosnow
go get bitbucket.org/tebeka/base62

# Until this pull request is merged we need to pull this in.
cd $GOPATH/src/github.com/revel/revel
hub am -3 https://github.com/revel/revel/pull/593