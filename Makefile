IMAGE_NAME:=jetstackexperimental/couchbase-sidecar
APP_NAME:=couchbase-sidecar
BUILD_DATE := $(shell date +%FT%T%z)
build: version
	CGO_ENABLED=0 GOOS=linux go build \
		-a -tags netgo \
		-o couchbase-sidecar \
		-ldflags "-X main.AppGitState=${GIT_STATE} -X main.AppGitCommit=${GIT_COMMIT} -X main.AppVersion=${APP_VERSION} -X main.AppBuildDate=${BUILD_DATE}"

image: build
	docker build -t $(IMAGE_NAME):latest .
	docker build -t $(IMAGE_NAME):$(APP_VERSION) .

push: image
	docker push $(IMAGE_NAME):latest
	docker push $(IMAGE_NAME):$(APP_VERSION)

#codegen:
#	mockgen -package=mocks -source=pkg/interfaces/interfaces.go > pkg/mocks/mocks.go

version:
	$(eval GIT_STATE := $(shell if test -z "`git status --porcelain 2> /dev/null`"; then echo "clean"; else echo "dirty"; fi))
	$(eval GIT_COMMIT := $(shell git rev-parse HEAD))
	$(eval APP_VERSION := $(shell cat VERSION))
