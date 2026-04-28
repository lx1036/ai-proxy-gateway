package bootstrap

import (
	// Register embed
	_ "embed"
	"fmt"
	"strings"

	"encoding/base64"
	"text/template"
)

const (
	envoyCfgFileName = "bootstrap.yaml"
)

//go:embed bootstrap.yaml.tpl
var bootstrapTmplStr string

var bootstrapTmpl = template.Must(template.New(envoyCfgFileName).Funcs(template.FuncMap{
	"base64": func(data []byte) string {
		return base64.StdEncoding.EncodeToString(data)
	},
}).Parse(bootstrapTmplStr))

type BootstrapConfig struct {
	renderStr string

	parameters bootstrapParameters
}

func (b *BootstrapConfig) render() error {
	buf := new(strings.Builder)
	if err := bootstrapTmpl.Execute(buf, b.parameters); err != nil {
		return fmt.Errorf("failed to render bootstrap config: %w", err)
	}

	b.renderStr = buf.String()
	return nil
}

func GetRenderBootstrapConfig() (string, error) {

}
