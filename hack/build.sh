#!/bin/bash -e

if [ -z "$GOOS" ]; then
	echo "Empty GOOS"
	exit 1
fi

if [ -z "$GOARCH" ]; then
	echo "Empty GOARCH"
	exit 1
fi

if [ -z "$VERSION" ]; then
	echo "Empty VERSION"
	exit 1
fi

if [ -z "$PACKAGE" ]; then
	echo "Empty PACKAGE"
	exit 1
fi

if [[ -z "${OUTPUT_DIR:-}" ]]; then
  OUTPUT_DIR=.
fi

OUTPUT=${OUTPUT_DIR}/dmaas-operator

GIT_SHA=$(git rev-parse HEAD)
git_dirty=$(git status --porcelain 2> /dev/null)
if [[ -z "${git_dirty}" ]]; then
  GIT_TREE_STATE=clean
else
  GIT_TREE_STATE=dirty
fi

LDFLAGS="-X ${PACKAGE}/pkg/cmd.Version=${VERSION}"
LDFLAGS="${LDFLAGS} -X ${PACKAGE}/pkg/cmd.GitSHA=${GIT_SHA}"
LDFLAGS="${LDFLAGS} -X ${PACKAGE}/pkg/cmd.GitTreeState=${GIT_TREE_STATE}"

export CGO_ENABLED=0

go build \
  -o ${OUTPUT} \
  -installsuffix "static" \
  -ldflags "${LDFLAGS}" \
  ${PACKAGE}/cmd/dmaas-operator

