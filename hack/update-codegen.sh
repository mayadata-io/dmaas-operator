#!/bin/bash
#
#Copyright 2020 The MayaData Authors.
#Licensed under the Apache License, Version 2.0 (the "License");
#you may not use this file except in compliance with the License.
#You may obtain a copy of the License at
#    https://www.apache.org/licenses/LICENSE-2.0
#Unless required by applicable law or agreed to in writing, software
#distributed under the License is distributed on an "AS IS" BASIS,
#WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#See the License for the specific language governing permissions and
#limitations under the License.

set -e

if [[ -z $GOPATH ]]; then
	echo "Setting GOPATH to ~/go"
	GOPATH=~/go
fi

if [[ ! -d "${GOPATH}/src/k8s.io/code-generator" ]]; then
	echo "k8s.io/code-generator missing from GOPATH"
	echo "Please clone https://github.com/kubernetes/code-generator under '${GOPATH}/src/k8s.io'"
  exit 1
fi


${GOPATH}/src/k8s.io/code-generator/generate-groups.sh all \
	github.com/mayadata-io/dmaas-operator/pkg/generated github.com/mayadata-io/dmaas-operator/pkg/apis \
	"mayadata.io:v1alpha1" \
	--output-base ../../..	\
	--go-header-file ./hack/boilerplate.go.txt

controller-gen \
	crd:crdVersions=v1beta1,preserveUnknownFields=false,trivialVersions=true \
	output:dir=./pkg/generated/crds/manifests \
	paths=./pkg/apis/mayadata.io/v1alpha1/...

#go-bindata -ignore=\\.go  -o pkg/generated/crds/bindata.go -pkg crds ./pkg/generated/crds/manifests/...
