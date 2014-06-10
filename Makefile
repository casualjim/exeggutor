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
	gopkg.in/validator.v1 \
	github.com/imdario/mergo \
	github.com/reverb/go-mesos/mesos \
	github.com/reverb/go-utils \
	github.com/op/go-logging \
	github.com/codegangsta/gin \
	github.com/pquerna/ffjson \
	github.com/astaxie/beego/validation \
	github.com/reverb/go-utils \
	github.com/reverb/go-mesos \
	github.com/golang/lint \ 
	code.google.com/p/go.tools/cmd/gotype

setup: 
	@$(foreach dir,$(DEPS),echo "installing $(dir)" && go get $(dir);)

agora-gin:
	@cd agora
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