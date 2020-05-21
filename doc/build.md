## Building dmaas-operator

dmaas-operator is written in [go language](https://golang.org/doc/install). Supported Go version is 1.14.

## Building code
To build dmaas-operator, run

```bash
make build
```

Above command will build the binary and place it under `_output/bin/linux/amd64/` or `_output/bin/${GOOS}/${GOARCH}.

## Updating generated code
We are using following tools to generate relevant code(lister,informer,clientset,deepcopy,crd) for `pkg/apis/mayadata.io`.
- [code-generator](https://github.com/kubernetes/code-generator) for lister,informer,clientset,deepcopy
- [controller-tools](https://github.com/kubernetes-sigs/controller-tools) for crd

Make sure that above tools are installed in your system.

To update the generated code, run

```bash
make update
```