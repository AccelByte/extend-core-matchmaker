# Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
# This is licensed software from AccelByte Inc, for limitations
# and restrictions contact your company contract manager.

SHELL := /bin/bash

PROJECT_NAME := $(shell basename "$$(pwd)")
GOLANG_IMAGE := golang:1.24.1-alpine

BUILD_CACHE_VOLUME := $(shell echo '$(PROJECT_NAME)' | sed 's/[^a-zA-Z0-9_-]//g')-build-cache

IMAGE_NAME := $(shell basename "$$(pwd)")-app
BUILDER := extend-builder

COMPOSE_TEST=docker-compose --no-ansi -p $(PROJECT_NAME)-test -f docker-compose-test.yaml

build: proto
	docker run -t --rm \
			-v $(BUILD_CACHE_VOLUME):/tmp/build-cache \
			$(GOLANG_IMAGE) \
			chown $$(id -u):$$(id -g) /tmp/build-cache		# For MacOS docker host: Workaround for /tmp/build-cache folder
	docker run -t --rm \
			-u $$(id -u):$$(id -g) \
			-e GOCACHE=/tmp/build-cache/go/cache \
			-e GOMODCACHE=/tmp/build-cache/go/modcache \
			-v $(BUILD_CACHE_VOLUME):/tmp/build-cache \
			-v $$(pwd):/data/ \
			-w /data/ \
			$(GOLANG_IMAGE) \
			go build -modcacherw

proto:
	rm -rfv pkg/pb/*.pb.go
	mkdir -p pkg/pb
	docker run -t --rm -u $$(id -u):$$(id -g) -v $$(pwd):/data/ -w /data/ rvolosatovs/protoc:4.0.0 \
			--proto_path=pkg/proto --go_out=pkg/pb \
			--go_opt=paths=source_relative --go-grpc_out=pkg/pb \
			--go-grpc_opt=paths=source_relative pkg/proto/*.proto

image:
	docker buildx build -t ${IMAGE_NAME} --load .

imagex:
	docker buildx inspect $(BUILDER) || docker buildx create --name $(BUILDER) --use
	docker buildx build -t ${IMAGE_NAME} --platform linux/amd64 .
	docker buildx build -t ${IMAGE_NAME} --load .
	docker buildx rm --keep-state $(BUILDER)

imagex_push:
	@test -n "$(IMAGE_TAG)" || (echo "IMAGE_TAG is not set (e.g. 'v0.1.0', 'latest')"; exit 1)
	@test -n "$(REPO_URL)" || (echo "REPO_URL is not set"; exit 1)
	docker buildx inspect $(BUILDER) || docker buildx create --name $(BUILDER) --use
	docker buildx build -t ${REPO_URL}:${IMAGE_TAG} --platform linux/amd64 --push .
	docker buildx rm --keep-state $(BUILDER)

ngrok:
	@which ngrok || (echo "ngrok is not installed" ; exit 1)
	@test -n "$(NGROK_AUTHTOKEN)" || (echo "NGROK_AUTHTOKEN is not set" ; exit 1)
	ngrok tcp 6565	# gRPC server port

lint:
	$(RUN) $(BUILDER_IMAGE) go mod vendor
	make lint-golang-ci
	make lint-outdated-dependencies

lint-golang-ci:
	docker run --rm -v $(PWD):$(PWD) -w $(PWD) golangci/golangci-lint:v1.60.3-alpine golangci-lint run -v --fix

test:
	$(COMPOSE_TEST) run --remove-orphans --rm -v $(CURDIR):/app -e CGO_ENABLED=1 -e PWD=$(CURDIR) test sh coverage.sh

# get golang modules with current and latest versions
# filter only direct dependencies(throw dependencies of dependencies)
# show error code 1 if AB dependencies are outdated
lint-outdated-dependencies:
	@echo "linting golang dependencies ..."
	go list -u -m -json -mod=mod all | \
	docker run -i psampaz/go-mod-outdated:v0.9.0 -update -direct | \
	grep "github.com/AccelByte" && exit 1 || echo "OK"