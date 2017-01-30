REGISTRY := jetstackexperimental
IMAGE_NAME := couchbase-sidecar
IMAGE_TAGS := canary
BUILD_TAG := build

BUILD_DATE := $(shell date +%FT%T%z)

APP_VERSION := dev

build: version
	CGO_ENABLED=0 GOOS=linux go build \
		-a -tags netgo \
		-o couchbase-sidecar \
		-ldflags "-X main.AppGitState=${GIT_STATE} -X main.AppGitCommit=${GIT_COMMIT} -X main.AppVersion=${APP_VERSION} -X main.AppBuildDate=${BUILD_DATE}"

image:
	docker build -t $(REGISTRY)/$(IMAGE_NAME):$(BUILD_TAG) .

push: image
	set -e; \
	for tag in $(IMAGE_TAGS); do \
		docker tag $(REGISTRY)/$(IMAGE_NAME):$(BUILD_TAG) $(REGISTRY)/$(IMAGE_NAME):$${tag} ; \
		docker push $(REGISTRY)/$(IMAGE_NAME):$${tag}; \
	done

push_minikube: image
	docker save $(REGISTRY)/$(IMAGE_NAME):$(BUILD_TAG) | minikube ssh -- docker load

#codegen:
#	mockgen -package=mocks -source=pkg/interfaces/interfaces.go > pkg/mocks/mocks.go

version:
	$(eval GIT_STATE := $(shell if test -z "`git status --porcelain 2> /dev/null`"; then echo "clean"; else echo "dirty"; fi))
	$(eval GIT_COMMIT := $(shell git rev-parse HEAD))
