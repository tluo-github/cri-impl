package oci

import (
	"bufio"
	"bytes"
	"github.com/opencontainers/runtime-tools/generate"
)

type RuntimeSpec []byte

type SpecOptions struct {
	Command      string
	Args         []string
	RootPath     string
	RootReadonly bool
}

func NewSpec(options SpecOptions) (RuntimeSpec, error) {
	gen, err := generate.New("linux")
	if err != nil {
		return nil, err
	}
	gen.HostSpecific = true
	gen.SetRootPath(options.RootPath)
	gen.SetRootReadonly(options.RootReadonly)
	gen.SetProcessArgs(append([]string{options.Command}, options.Args...))

	var buf bytes.Buffer
	exprOpts := generate.ExportOptions{}
	if err := gen.Save(bufio.NewWriter(&buf), exprOpts); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil

}
