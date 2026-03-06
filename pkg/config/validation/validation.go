package validation

import "github.com/lx1036/gateway/pkg/config"

var (
	validateFuncs = make(map[string]ValidateFunc)

	// EmptyValidate is a Validate that does nothing and returns no error.
	EmptyValidate = RegisterValidateFunc("EmptyValidate",
		func(config.Config) (error, error) {
			return nil, nil
		},
	)
)

// ValidateFunc defines a validation func for an API proto.
type ValidateFunc func(config config.Config) (error, error)

func IsValidateFunc(name string) bool {
	return GetValidateFunc(name) != nil
}

func GetValidateFunc(name string) ValidateFunc {
	return validateFuncs[name]
}

func RegisterValidateFunc(name string, f ValidateFunc) ValidateFunc {
	// Wrap the original validate function with an extra validate function for object metadata
	//validate := validateMetadata(f)
	//validateFuncs[name] = validate
	validateFuncs[name] = f
	return f
}
