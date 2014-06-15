DEPS = github.com/onsi/ginkgo/ginkgo \
	github.com/onsi/gomega \
	code.google.com/p/gomock/gomock \
	code.google.com/p/gomock/mockgen \
	github.com/qur/withmock \
	github.com/samuel/go-zookeeper/zk \
	github.com/jessevdk/go-flags \
	github.com/rcrowley/go-metrics \
	github.com/yvasiyarov/gorelic \
	github.com/wendal/mustache \
	github.com/op/go-logging \
	github.com/hishboy/gocommons/lang \
	github.com/armon/gomdb \
	github.com/hashicorp/go-version \
	github.com/sdming/gosnow \
	bitbucket.org/tebeka/base62 \
	github.com/codegangsta/negroni \
	github.com/julienschmidt/httprouter \
	github.com/imdario/mergo \
	github.com/reverb/go-mesos/mesos \
	github.com/reverb/go-utils \
	github.com/op/go-logging \
	github.com/codegangsta/gin \
	github.com/pquerna/ffjson \
	github.com/astaxie/beego/validation \
	github.com/golang/lint \
	code.google.com/p/go.tools/cmd/gotype \
	code.google.com/p/go.tools/cmd/vet \
	code.google.com/p/go.tools/cmd/godoc \
	code.google.com/p/go.tools/cmd/goimports \
	code.google.com/p/go.tools/cmd/oracle \
	code.google.com/p/go.tools/cmd/gotype \
	code.google.com/p/rog-go/exp/cmd/godef \
	github.com/axw/gocov/gocov \
	gopkg.in/matm/v1/gocov-html \
	github.com/AlekSi/gocov-xml \
	github.com/nsf/gocode \
	github.com/golang/lint/golint \
	github.com/kisielk/errcheck \
	github.com/jstemmer/gotags 

setup: 
	@$(foreach dir,$(DEPS),echo "installing $(dir)" && go get $(dir);)

update-deps: 
	@$(foreach dir,$(DEPS),echo "installing $(dir)" && go get -u $(dir);)

agora-gin:
	@cd agora;
	@gin -b /tmp/gin-agora-bin

agora-test:
	@cd agora
	@ginkgo -r 

state-test:
	@cd state
	@ginkgo -r

store-test:
	@cd store
	@ginkgo -r

test-all: state-test store-test agora-test
