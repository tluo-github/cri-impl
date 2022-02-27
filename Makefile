GO              ?=  go
GOPATH          := $(firstword $(subst :, ,$(shell $(GO) env GOPATH)))
GOLANGCI_LINT   ?= $(GOPATH)/bin/golangci-lint
BIN_DIR         ?= $(shell pwd)/bin
GIT_BRANCH      ?= `git symbolic-ref --short -q HEAD`
GIT_COMMIT      ?= `git rev-parse --short HEAD`
BUILD_DATE      ?= `date +%FT%T%z`
LDFLAGS		    ?= -ldflags "-w -s -X gitlab.poizon.com/luotao/work-tools/pkg/version.GitBranch=${GIT_BRANCH}  -X  gitlab.poizon.com/luotao/work-tools/pkg/version.GitCommit=${GIT_COMMIT} -X gitlab.poizon.com/luotao/work-tools/pkg/version.BuildDate=${BUILD_DATE}"

ecs-ttl: clean lint macos linux

lint:
	@echo ">> linting code"
	@$(GOLANGCI_LINT) run

macos:
	GOARCH=amd64 GOOS=darwin go build  -o ${BIN_DIR}/cri-impl-macos  main.go
	GOARCH=amd64 GOOS=darwin go build  -o ${BIN_DIR}/crictl-macos  ctl/main.go

linux:
	GOARCH=amd64 GOOS=linux go build  -o ${BIN_DIR}/cri-impl-macos  main.go
	GOARCH=amd64 GOOS=linux go build  -o ${BIN_DIR}/crictl-macos ctl/main.go



clean:
	rm -rf bin/*