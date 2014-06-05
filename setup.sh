#!/bin/sh

if [ ! -f `which go` ]; then
  echo "You need to install go for this script to work"
  exit 1
fi

while read package; do
    echo "installing ${package}"
    go get "${package}"
done << EOF
github.com/onsi/ginkgo/ginkgo
github.com/onsi/gomega
code.google.com/p/gomock/gomock
code.google.com/p/gomock/mockgen
github.com/qur/withmock
github.com/samuel/go-zookeeper/zk
github.com/jessevdk/go-flags
github.com/rcrowley/go-metrics
github.com/yvasiyarov/gorelic
github.com/wendal/mustache
github.com/op/go-logging
github.com/hishboy/gocommons/lang
github.com/armon/gomdb
github.com/hashicorp/go-version
github.com/sdming/gosnow
bitbucket.org/tebeka/base62
github.com/codegangsta/negroni
github.com/julienschmidt/httprouter
gopkg.in/validator.v1
github.com/imdario/mergo
github.com/reverb/go-mesos/mesos
github.com/reverb/go-utils
github.com/op/go-logging
go get github.com/codegangsta/gin
EOF