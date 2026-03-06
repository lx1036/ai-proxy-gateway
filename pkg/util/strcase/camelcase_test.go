package strcase

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestCamelCase(t *testing.T) {
	cases := map[string]string{
		"":              "",
		"foo":           "Foo",
		"foobar":        "Foobar",
		"fooBar":        "FooBar",
		"foo_bar":       "FooBar",
		"foo-bar":       "FooBar",
		"foo_Bar":       "FooBar",
		"foo9bar":       "Foo9Bar",
		"HTTP-API-Spec": "HTTPAPISpec",
		"http-api-spec": "HttpApiSpec",
		"_foo":          "XFoo",
		"-foo":          "XFoo",
		"_Foo":          "XFoo",
		"-Foo":          "XFoo",
	}

	for k, v := range cases {
		t.Run(k, func(t *testing.T) {
			g := NewWithT(t)

			a := CamelCase(k)
			g.Expect(a).To(Equal(v))
		})
	}
}

func TestCamelCaseToKebabCase(t *testing.T) {
	cases := map[string]string{
		"":                   "",
		"Foo":                "foo",
		"FooBar":             "foo-bar",
		"foo9bar":            "foo9bar",
		"HTTPAPISpec":        "http-api-spec",
		"HTTPAPISpecBinding": "http-api-spec-binding",
	}

	for k, v := range cases {
		t.Run(k, func(t *testing.T) {
			g := NewWithT(t)

			a := CamelCaseToKebabCase(k)
			g.Expect(a).To(Equal(v))
		})
	}
}
