#!/usr/bin/make -f
# make debug 时使用
KUBECONFIG ?= hack/kubeconfig

# make debug 时使用
DEPLOY ?= hack/debug-deploy.yml

NAMESPACE ?= openstack

#IMAGE=install-ecnf.es.easystack.cn/production/escloud-linux-source-node-problem-detector:v0.0.1
IMAGE=docker.io/altonzhu/node-problem-detector:v0.0.1

GOPROXY=http://goproxy.easystack.io/,https://goproxy.cn/,direct
GONOSUMDB=review.easystack.cn


PKG=review.easystack.cn/easystack/node-problem-detector

GIT_COMMIT_VERSION = $(shell git show -s --format='format:%H')
GIT_COMMIT_TIME = $(shell git show -s --format='format:%aI')
GIT_BRANCH = $(shell git rev-parse --abbrev-ref HEAD)

GO_LDFLAGS=-ldflags '-s -w -X $(PKG)/version.CommitVersion=$(GIT_COMMIT_VERSION) -X $(PKG)/version.CommitTime=$(GIT_COMMIT_TIME) -X $(PKG)/version.Branch=$(GIT_BRANCH)'

all: build
build: build-amd64
test:  test-style  test-unit coverage
.PHONY: build-arm64 build-amd64

build-arm64:
	@go env -w GOPROXY=${GOPROXY}
	@go env -w GONOSUMDB=${GONOSUMDB}
	@go env -w CGO_ENABLED=0
	@mkdir -p bin
	@GOOS=linux GOARCH=arm64 go build $(GO_LDFLAGS) -o ./bin/node-problem-detector ./cmd/nodeproblemdetector
	@GOOS=linux GOARCH=arm64 go build $(GO_LDFLAGS) -o ./bin/net-checker ./cmd/netchecker

build-amd64:
	@go env -w GOPROXY=${GOPROXY}
	@go env -w GONOSUMDB=${GONOSUMDB}
	@go env -w CGO_ENABLED=0
	@mkdir -p bin
	@GOOS=linux GOARCH=amd64 go build $(GO_LDFLAGS) -o ./bin/node-problem-detector ./cmd/nodeproblemdetector
	@GOOS=linux GOARCH=amd64 go build $(GO_LDFLAGS) -o ./bin/net-checker ./cmd/netchecker

image:
	docker build --platform linux/amd64 -f Dockerfile -t ${IMAGE} .
	docker push ${IMAGE}
	# docker buildx build --platform linux/amd64,linux/arm64 -f Dockerfile -t ${IMAGE} .
	# docker push ${IMAGE}

gen: gen-api copyright

gen-api:
	scripts/generate.sh

debug: ktctl
	kubectl --kubeconfig=${KUBECONFIG} apply -f ${DEPLOY}
	ktctl -c ${KUBECONFIG} -n ${NAMESPACE} exchange debug --expose 8080

ktctl:
	# Get ktctl from one of the possibly locations, ordered by priority
	@KTCTL=$(shell PATH=$$PATH:$$GOPATH/bin:$$GOBIN:$$GOPATH/bin:$$HOME/go/bin which ktctl)
ifeq ($(KTCTL), "")
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get github.com/alibaba/kt-connect/cmd/ktctl@v0.2.5 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
	@KTCTL=$(shell PATH=$$PATH:$$GOPATH/bin:$$GOBIN:$$GOPATH/bin:$$HOME/go/bin which ktctl)
endif

# 给文件添加 copyright
.PHONY: copyright
copyright:
	bash ./scripts/copyright.sh

.PHONY: test-style
test-style:
	@scripts/teststyle.sh
	@echo "test-style PASS"
.PHONY: test-unit
test-unit:
	@go test ./...
	@echo "test-unit PASS"
#	@scripts/testunit.sh

.PHONY: coverage
coverage:
	@go vet ./...
	@echo "coverage PASS"
#	@scripts/coverage.sh

.PHONY: fmt
fmt:
	gofmt -s -w .