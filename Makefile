# Copyright 2020 The MayaData Authors.
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#     https://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# repo's import path
PACKAGE=github.com/mayadata-io/dmaas-operator

git_branch := $(or ${TRAVIS_BRANCH}, $(shell git rev-parse --abbrev-ref HEAD))
git_tag := $(shell git describe --exact-match --abbrev=0 2>/dev/null || echo "")

VERSION ?= $(or $(or ${TRAVIS_TAG}, $(git_tag)),$(git_branch))

ifeq ($(VERSION), master)
VERSION = ci
else ifneq (${TRAVIS_TAG},$(VERSION))
VERSION := $(VERSION)-ci
endif

# to push image with tag latest
TAG_LATEST ?= false

GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

# Where to push the docker image.
REGISTRY ?= mayadataio

BIN=dmaas-operator

IMAGE ?= $(REGISTRY)/$(BIN)-$(GOARCH)

# Specify the url for docker image
DBUILD_SITE_URL ?= "https://mayadata.io/"

# date of build used during docker image build
DBUILD_DATE=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

# repository url used in docker image
DBUILD_REPO_URL ?= "https://github.com/mayadata-io/dmaas-operator"

build-dirs:
	@mkdir -p _output/bin/$(GOOS)/$(GOARCH)

build: build-dirs
	@echo "building binary _output/bin/$(GOOS)/$(GOARCH)/dmaas-operator"
	@GOOS=$(GOOS) \
	GOARCH=$(GOARCH) \
	VERSION=$(VERSION) \
	PACKAGE=$(PACKAGE) \
	OUTPUT_DIR=$$(pwd)/_output/bin/$(GOOS)/$(GOARCH) \
	./hack/build.sh

image: build
	@echo "building: $(IMAGE):$(VERSION)"
	@docker build \
	--build-arg ARCH=$(GOARCH) \
	--build-arg OS=$(GOOS) \
	--build-arg DBUILD_DATE=$(DBUILD_DATE) \
	--build-arg DBUILD_REPO_URL=$(DBUILD_REPO_URL) \
	--build-arg DBUILD_SITE_URL=$(DBUILD_SITE_URL) \
	-t $(IMAGE):$(VERSION) -f Dockerfile .

push-image: image
	@echo "push: $(IMAGE):$(VERSION)"
	@docker push $(IMAGE):$(VERSION)
ifeq ($(TAG_LATEST), true)
	docker tag $(IMAGE):$(VERSION) $(IMAGE):latest
	docker push $(IMAGE):latest
endif

test:
	@go test -race -timeout=20m -cover -coverprofile=coverage.out -covermode=atomic ./cmd/... ./pkg/...

clean:
	@rm -rf .go _output

update:
	@hack/verify-update.sh
	@hack/check-license.sh
