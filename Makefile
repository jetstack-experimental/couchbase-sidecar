IMAGE_NAME:=jetstackexperimental/couchbase-sidecar
APP_NAME:=couchbase-sidecar
build: version
	CGO_ENABLED=0 GOOS=linux go build \
		-a -tags netgo \
		-o couchbase-sidecar \
		-ldflags "-X couchbase_sidecar.AppGitState=${GIT_STATE} -X couchbase_sidecar.AppGitCommit=${GIT_COMMIT} -X couchbase_sidecar.AppVersion=${APP_VERSION}"

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
