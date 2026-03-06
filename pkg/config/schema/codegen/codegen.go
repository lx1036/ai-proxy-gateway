package main

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"github.com/lx1036/gateway/pkg/config/schema/ast"
	strcaseutil "github.com/lx1036/gateway/pkg/util/strcase"
	"k8s.io/apimachinery/pkg/util/sets"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/stoewer/go-strcase"
)

//go:embed templates/gvk.go.tmpl
var gvkTemplate string

//go:embed templates/gvr.go.tmpl
var gvrTemplate string

//go:embed templates/crdclient.go.tmpl
var crdclientTemplate string

//go:embed templates/types.go.tmpl
var typesTemplate string

//go:embed templates/clients.go.tmpl
var clientsTemplate string

//go:embed templates/kind.go.tmpl
var kindTemplate string

//go:embed templates/collections.go.tmpl
var collectionsTemplate string

type packageImport struct {
	PackageName string
	ImportName  string
}

type colEntry struct {
	Resource *ast.Resource

	// ClientImport represents the import alias for the client. Example: clientnetworkingv1alpha3.
	ClientImport string
	// ClientImport represents the import alias for the status. Example: clientnetworkingv1alpha3.
	StatusImport string
	// IstioAwareClientImport represents the import alias for the API, taking into account Istio storing its API (spec)
	// separate from its client import
	// Example: apiclientnetworkingv1alpha3.
	IstioAwareClientImport string
	// ClientGroupPath represents the group in the client. Example: NetworkingV1alpha3.
	ClientGroupPath string
	// ClientGetter returns the path to get the client from a kube.Client. Example: Istio.
	ClientGetter string
	// ClientTypePath returns the kind name. Basically upper cased "plural". Example: Gateways
	ClientTypePath string
	// SpecType returns the type of the Spec field. Example: HTTPRouteSpec.
	SpecType   string
	StatusType string
}

type inputs struct {
	Entries  []colEntry
	Packages []packageImport
}

func Run() error {
	inp, err := buildInputs()
	if err != nil {
		return err
	}

	// Include synthetic types used for XDS pushes
	kindEntries := append([]colEntry{
		{
			Resource: &ast.Resource{Identifier: "Address", Kind: "Address", Version: "internal", Group: "internal"},
		},
		{
			Resource: &ast.Resource{Identifier: "DNSName", Kind: "DNSName", Version: "internal", Group: "internal"},
		},
	}, inp.Entries...)

	sort.Slice(kindEntries, func(i, j int) bool {
		return strings.Compare(kindEntries[i].Resource.Identifier, kindEntries[j].Resource.Identifier) < 0
	})

	// filter to only types agent needs (to keep binary small)
	agentEntries := []colEntry{}
	for _, e := range inp.Entries {
		if strings.Contains(e.Resource.ProtoPackage, "istio.io") &&
			e.Resource.Kind != "EnvoyFilter" {
			agentEntries = append(agentEntries, e)
		}
	}

	// Build a deduplicated list of Kind names for KebabKind function
	seenKinds := make(map[string]bool)
	var uniqueKinds []string
	for _, e := range inp.Entries {
		if !seenKinds[e.Resource.Kind] {
			seenKinds[e.Resource.Kind] = true
			uniqueKinds = append(uniqueKinds, e.Resource.Kind)
		}
	}
	sort.Strings(uniqueKinds)

	return errors.Join(
		writeTemplate("pkg/config/schema/gvk/resources.gen.go", gvkTemplate, map[string]any{
			"Entries":     inp.Entries,
			"UniqueKinds": uniqueKinds,
			"PackageName": "gvk",
		}),
		writeTemplate("pkg/config/schema/gvr/resources.gen.go", gvrTemplate, map[string]any{
			"Entries":     inp.Entries,
			"PackageName": "gvr",
		}),
		writeTemplate("pkg/config/kube/crdclient/types.gen.go", crdclientTemplate, map[string]any{
			"Entries":     inp.Entries,
			"Packages":    inp.Packages,
			"PackageName": "crdclient",
		}),
		writeTemplate("pkg/config/schema/kubetypes/resources.gen.go", typesTemplate, map[string]any{
			"Entries":     inp.Entries,
			"Packages":    inp.Packages,
			"PackageName": "kubetypes",
		}),
		writeTemplate("pkg/config/schema/kubeclient/resources.gen.go", clientsTemplate, map[string]any{
			"Entries":     inp.Entries,
			"Packages":    inp.Packages,
			"PackageName": "kubeclient",
		}),
		writeTemplate("pkg/config/schema/kind/resources.gen.go", kindTemplate, map[string]any{
			"Entries":     kindEntries,
			"PackageName": "kind",
		}),

		writeTemplate("pkg/config/schema/collections/collections.gen.go", collectionsTemplate, map[string]any{
			"Entries":      inp.Entries,
			"Packages":     inp.Packages,
			"PackageName":  "collections",
			"FilePrefix":   "//go:build !agent",
			"CustomImport": `  "github.com/lx1036/gateway/pkg/config/validation/envoyfilter"`,
		}),
		writeTemplate("pkg/config/schema/collections/collections.agent.gen.go", collectionsTemplate, map[string]any{
			"Entries":      agentEntries,
			"Packages":     inp.Packages,
			"PackageName":  "collections",
			"FilePrefix":   "//go:build agent",
			"CustomImport": "",
		}),
	)
}

func buildInputs() (inputs, error) {
	abs, _ := filepath.Abs("pkg/config/schema/metadata.yaml")
	b, err := os.ReadFile(abs)
	if err != nil {
		fmt.Printf("unable to read input file: %v", err)
		return inputs{}, err
	}

	m, err := ast.Parse(string(b))
	if err != nil {
		fmt.Printf("failed parsing input file: %v", err)
		return inputs{}, err
	}

	entries := make([]colEntry, 0, len(m.Resources))
	for _, r := range m.Resources {
		spl := strings.Split(r.Proto, ".")
		tname := spl[len(spl)-1]
		stat := strings.Split(r.StatusProto, ".")
		statName := stat[len(stat)-1]
		e := colEntry{
			Resource:               r,
			ClientImport:           toImport(r.ProtoPackage),
			StatusImport:           toImport(r.StatusProtoPackage),
			IstioAwareClientImport: toIstioAwareImport(r.ProtoPackage, r.Version),
			ClientGroupPath:        toGroup(r.ProtoPackage, r.Version),
			ClientGetter:           toGetter(r.ProtoPackage),
			ClientTypePath:         toTypePath(r),
			SpecType:               tname,
		}
		if r.StatusProtoPackage != "" {
			e.StatusType = statName
		}
		entries = append(entries, e)
	}

	sort.Slice(entries, func(i, j int) bool {
		return strings.Compare(entries[i].Resource.Identifier, entries[j].Resource.Identifier) < 0
	})

	names := sets.New[string]()
	for _, r := range m.Resources {
		if r.ProtoPackage != "" {
			names.Insert(r.ProtoPackage)
		}
		if r.StatusProtoPackage != "" {
			names.Insert(r.StatusProtoPackage)
		}
	}

	packages := make([]packageImport, 0, names.Len())
	for p := range names {
		packages = append(packages, packageImport{p, toImport(p)})
	}
	sort.Slice(packages, func(i, j int) bool {
		return strings.Compare(packages[i].PackageName, packages[j].PackageName) < 0
	})

	return inputs{
		Entries:  entries,
		Packages: packages,
	}, nil
}

func writeTemplate(path, tmpl string, i any) error {
	t, err := applyTemplate(tmpl, i)
	if err != nil {
		return fmt.Errorf("apply template %v: %v", path, err)
	}
	dst, _ := filepath.Abs(path)
	if err = os.WriteFile(dst, []byte(t), os.ModePerm); err != nil {
		return fmt.Errorf("write template %v: %v", path, err)
	}
	c := exec.Command("goimports", "-w", "-local", "istio.io", dst)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func applyTemplate(tmpl string, i any) (string, error) {
	t := template.New("tmpl").Funcs(template.FuncMap{
		"contains":  strings.Contains,
		"kebabcase": camelCaseToKebabCase,
	})

	t2 := template.Must(t.Parse(tmpl))

	var b bytes.Buffer
	if err := t2.Execute(&b, i); err != nil {
		return "", err
	}

	return b.String(), nil
}

// camelCaseToKebabCase wraps strcase.CamelCaseToKebabCase for use in templates.
func camelCaseToKebabCase(s string) string {
	return strcaseutil.CamelCaseToKebabCase(s)
}

func toImport(p string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(p, "/", ""), ".", ""), "-", "")
}

func toIstioAwareImport(protoPackage string, version string) string {
	p := strings.Split(protoPackage, "/")
	base := strings.Join(p[:len(p)-1], "")
	imp := strings.ReplaceAll(strings.ReplaceAll(base, ".", ""), "-", "") + version
	if strings.Contains(protoPackage, "istio.io") {
		return "api" + imp
	}
	return imp
}

func toTypePath(r *ast.Resource) string {
	k := r.Kind
	g := r.Plural
	res := strings.Builder{}
	for i, c := range g {
		if i >= len(k) {
			res.WriteByte(byte(c))
		} else {
			if k[i] == bytes.ToUpper([]byte{byte(c)})[0] {
				res.WriteByte(k[i])
			} else {
				res.WriteByte(byte(c))
			}
		}
	}
	return res.String()
}

func toGetter(protoPackage string) string {
	if strings.Contains(protoPackage, "istio.io") {
		return "Istio"
	} else if strings.Contains(protoPackage, "sigs.k8s.io/gateway-api-inference-extension") {
		return "GatewayAPIInference"
	} else if strings.Contains(protoPackage, "sigs.k8s.io/gateway-api") {
		return "GatewayAPI"
	} else if strings.Contains(protoPackage, "k8s.io/apiextensions-apiserver") {
		return "Ext"
	}
	return "Kube"
}

func toGroup(protoPackage string, version string) string {
	p := strings.Split(protoPackage, "/")
	e := len(p) - 1
	if strings.Contains(protoPackage, "sigs.k8s.io/gateway-api/apisx") {
		// Custom naming for "X" types
		return "Experimental" + strcase.UpperCamelCase(version)
	} else if strings.Contains(protoPackage, "sigs.k8s.io/gateway-api-inference-extension") {
		// Gateway has one level of nesting with custom name
		return "Inference" + strcase.UpperCamelCase(version)
	} else if strings.Contains(protoPackage, "sigs.k8s.io/gateway-api") {
		// Gateway has one level of nesting with custom name
		return "Gateway" + strcase.UpperCamelCase(version)
	}
	// rest have two levels of nesting
	return strcase.UpperCamelCase(p[e-1]) + strcase.UpperCamelCase(version)
}
