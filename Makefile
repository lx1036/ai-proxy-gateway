





goimports:
	go install golang.org/x/tools/cmd/goimports@latest


deepcopy-gen:
#	go install k8s.io/code-generator/cmd/deepcopy-gen@latest
	deepcopy-gen --output-file zz_generated.deepcopy.go ./pkg/envoygateway/ir
