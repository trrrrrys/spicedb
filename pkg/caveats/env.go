package caveats

import (
	"fmt"

	"github.com/google/cel-go/cel"

	"github.com/authzed/spicedb/pkg/caveats/types"
	core "github.com/authzed/spicedb/pkg/proto/core/v1"
)

// Environment defines the evaluation environment for a caveat.
type Environment struct {
	variables map[string]types.VariableType
}

// NewEnvironment creates and returns a new environment for compiling a caveat.
func NewEnvironment() *Environment {
	return &Environment{
		variables: map[string]types.VariableType{},
	}
}

// EnvForVariables returns a new environment constructed for the given variables.
func EnvForVariables(vars map[string]types.VariableType) (*Environment, error) {
	e := NewEnvironment()
	for varName, varType := range vars {
		err := e.AddVariable(varName, varType)
		if err != nil {
			return nil, err
		}
	}
	return e, nil
}

// MustEnvForVariables returns a new environment constructed for the given variables
// or panics.
func MustEnvForVariables(vars map[string]types.VariableType) *Environment {
	env, err := EnvForVariables(vars)
	if err != nil {
		panic(err)
	}
	return env
}

// AddVariable adds a variable with the given type to the environment.
func (e *Environment) AddVariable(name string, varType types.VariableType) error {
	if _, ok := e.variables[name]; ok {
		return fmt.Errorf("variable `%s` already exists", name)
	}

	e.variables[name] = varType
	return nil
}

// EncodedParametersTypes returns the map of encoded parameters for the environment.
func (e *Environment) EncodedParametersTypes() map[string]*core.CaveatTypeReference {
	return types.EncodeParameterTypes(e.variables)
}

// asCelEnvironment converts the exported Environment into an internal CEL environment.
func (e *Environment) asCelEnvironment() (*cel.Env, error) {
	opts := make([]cel.EnvOption, 0, len(e.variables)+len(types.CustomTypes)+2)

	// Add the custom types and functions.
	for _, customTypeOpts := range types.CustomTypes {
		opts = append(opts, customTypeOpts...)
	}
	opts = append(opts, types.CustomMethodsOnTypes...)

	// Set options.
	// DefaultUTCTimeZone: ensure all timestamps are evaluated at UTC
	opts = append(opts, cel.DefaultUTCTimeZone(true))

	// OptionalTypes: enable optional typing syntax, e.g. `sometype?.foo`
	// See: https://github.com/google/cel-spec/wiki/proposal-246
	opts = append(opts, cel.OptionalTypes(cel.OptionalTypesVersion(0)))

	// EnableMacroCallTracking: enables tracking of call macros so when we call AstToString we get
	// back out the expected expressions.
	// See: https://github.com/google/cel-go/issues/474
	opts = append(opts, cel.EnableMacroCallTracking())

	for name, varType := range e.variables {
		opts = append(opts, cel.Variable(name, varType.CelType()))
	}
	return cel.NewEnv(opts...)
}
