## Building dmaas-operator

dmaas-operator is written in [go language](https://golang.org/doc/install). Supported Go version is 1.14.

## Updating generated code
We are using following tools to generate relevant code(lister,informer,clientset,deepcopy,crd) for `pkg/apis/mayadata.io`.
- [code-generator](https://github.com/kubernetes/code-generator) for lister,informer,clientset,deepcopy
- [controller-tools](https://github.com/kubernetes-sigs/controller-tools) for crd

Make sure that above tools are installed in your system.

To update the generated code, run

```bash
make update
```

## Building code
To build dmaas-operator, run

```bash
make build
```

Above command will build the binary and place it under `_output/bin/linux/amd64/` or `_output/bin/${GOOS}/${GOARCH}.

## Building docker image
To build docker image, run

```bash
make image
```

Above command will build the docker image with name `mayadata.io/dmaas-operator-amd64:ci`.

Build will auto-generate prefix `-amd64` and tag `ci`.
- `-amd64` is based on host system architecture and can be configured via `GOARCH`.
- `ci` is for image generated from master.
    - If you are building image from custom branch then image tag will be `branchname-ci`.
    - If you are building image from tag then image tag will be tag name.

To build image using your custom docker repo name, run

```bash
make image REGISTRY=your-repo-name
```