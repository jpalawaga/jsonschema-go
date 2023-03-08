package jsonschema

import (
	"context"
	"reflect"
	"strings"

	"github.com/swaggest/refl"
)

// CollectDefinitions enables collecting definitions with provided func instead of result schema.
func CollectDefinitions(f func(name string, schema Schema)) func(*ReflectContext) {
	return func(rc *ReflectContext) {
		rc.CollectDefinitions = f
	}
}

// DefinitionsPrefix sets up location for newly created references, default "#/definitions/".
func DefinitionsPrefix(prefix string) func(*ReflectContext) {
	return func(rc *ReflectContext) {
		rc.DefinitionsPrefix = prefix
	}
}

// PropertyNameTag sets up which field tag to use for property name, default "json".
func PropertyNameTag(tag string, additional ...string) func(*ReflectContext) {
	return func(rc *ReflectContext) {
		rc.PropertyNameTag = tag
		rc.PropertyNameAdditionalTags = additional
	}
}

// InterceptTypeFunc can intercept type reflection to control or modify schema.
//
// True bool result demands no further processing for the Schema.
//
// Deprecated: use InterceptSchemaFunc.
type InterceptTypeFunc func(reflect.Value, *Schema) (stop bool, err error)

// InterceptSchemaFunc can intercept type reflection to control or modify schema.
//
// True bool result demands no further processing for the Schema.
type InterceptSchemaFunc func(reflect.Value, *Schema) (stop bool, err error)

type InterceptSchemaParams struct {
	Value     reflect.Value
	Schema    *Schema
	Processed bool
}

// InterceptPropertyFunc can intercept field reflection to control or modify schema.
//
// Return ErrSkipProperty to avoid adding this property to parent Schema.Properties.
// Pointer to parent Schema is available in propertySchema.Parent.
//
// Deprecated: use InterceptPropFunc.
type InterceptPropertyFunc func(name string, field reflect.StructField, propertySchema *Schema) error

// InterceptPropFunc can intercept field reflection to control or modify schema.
//
// Return ErrSkipProperty to avoid adding this property to parent Schema.Properties.
// Pointer to parent Schema is available in propertySchema.Parent.
type InterceptPropFunc func(params InterceptPropParams) error

type InterceptPropParams struct {
	_              func() // No cast to 3rd party struct?
	Path           []string
	Name           string
	Field          reflect.StructField
	PropertySchema *Schema
}

// InterceptNullabilityParams defines InterceptNullabilityFunc parameters.
type InterceptNullabilityParams struct {
	OrigSchema Schema
	Schema     *Schema
	Type       reflect.Type
	OmitEmpty  bool
	NullAdded  bool
	RefDef     *Schema
}

// InterceptNullabilityFunc can intercept schema reflection to control or modify nullability state.
// It is called after default nullability rules are applied.
type InterceptNullabilityFunc func(params InterceptNullabilityParams)

// InterceptNullability add hook to customize nullability.
func InterceptNullability(f InterceptNullabilityFunc) func(reflectContext *ReflectContext) {
	return func(rc *ReflectContext) {
		if rc.InterceptNullability != nil {
			prev := rc.InterceptNullability
			rc.InterceptNullability = func(params InterceptNullabilityParams) {
				prev(params)
				f(params)
			}
		} else {
			rc.InterceptNullability = f
		}
	}
}

// InterceptType adds hook to customize schema.
func InterceptType(f InterceptTypeFunc) func(*ReflectContext) {
	return func(rc *ReflectContext) {
		if rc.InterceptType != nil {
			prev := rc.InterceptType
			rc.InterceptType = func(v reflect.Value, s *Schema) (b bool, err error) {
				ret, err := prev(v, s)
				if err != nil || ret {
					return ret, err
				}

				return f(v, s)
			}
		} else {
			rc.InterceptType = f
		}
	}
}

// InterceptProperty adds hook to customize property schema.
//
// Deprecated: use InterceptProp.
func InterceptProperty(f InterceptPropertyFunc) func(*ReflectContext) {
	return InterceptProp(func(params InterceptPropParams) error {
		return f(params.Name, params.Field, params.PropertySchema)
	})
}

func InterceptProp(f InterceptPropFunc) func(reflectContext *ReflectContext) {
	return func(rc *ReflectContext) {
		if rc.interceptProp != nil {
			prev := rc.interceptProp
			rc.interceptProp = func(params InterceptPropParams) error {
				err := prev(params)
				if err != nil {
					return err
				}

				return f(params)
			}
		} else {
			rc.interceptProp = f
		}
	}
}

// InlineRefs prevents references.
func InlineRefs(rc *ReflectContext) {
	rc.InlineRefs = true
}

// RootNullable enables nullability (by pointer) for root schema, disabled by default.
func RootNullable(rc *ReflectContext) {
	rc.RootNullable = true
}

// RootRef enables referencing root schema.
func RootRef(rc *ReflectContext) {
	rc.RootRef = true
}

// StripDefinitionNamePrefix checks if definition name has any of provided prefixes
// and removes first encountered.
func StripDefinitionNamePrefix(prefix ...string) func(rc *ReflectContext) {
	return func(rc *ReflectContext) {
		rc.DefName = func(t reflect.Type, defaultDefName string) string {
			for _, p := range prefix {
				s := strings.TrimPrefix(defaultDefName, p)
				if s != defaultDefName {
					return s
				}
			}

			return defaultDefName
		}
	}
}

// PropertyNameMapping enables property name mapping from a struct field name.
func PropertyNameMapping(mapping map[string]string) func(rc *ReflectContext) {
	return func(rc *ReflectContext) {
		rc.PropertyNameMapping = mapping
	}
}

// ProcessWithoutTags enables processing fields without any tags specified.
func ProcessWithoutTags(rc *ReflectContext) {
	rc.ProcessWithoutTags = true
}

// SkipEmbeddedMapsSlices disables shortcutting into embedded maps and slices.
func SkipEmbeddedMapsSlices(rc *ReflectContext) {
	rc.SkipEmbeddedMapsSlices = true
}

// SkipUnsupportedProperties skips properties with unsupported types (func, chan, etc...) instead of failing.
func SkipUnsupportedProperties(rc *ReflectContext) {
	rc.SkipUnsupportedProperties = true
}

// ReflectContext accompanies single reflect operation.
type ReflectContext struct {
	// Context allows communicating user data between reflection steps.
	context.Context

	// DefName returns custom definition name for a type, can be nil.
	DefName func(t reflect.Type, defaultDefName string) string

	// CollectDefinitions is triggered when named schema is created, can be nil.
	// Non-empty CollectDefinitions disables collection of definitions into resulting schema.
	CollectDefinitions func(name string, schema Schema)

	// DefinitionsPrefix defines location of named schemas, default #/definitions/.
	DefinitionsPrefix string

	// PropertyNameTag enables property naming from a field tag, e.g. `header:"first_name"`.
	PropertyNameTag string

	// PropertyNameAdditionalTags enables property naming from first available of multiple tags
	// if PropertyNameTag was not found.
	PropertyNameAdditionalTags []string

	// PropertyNameMapping enables property name mapping from a struct field name, e.g. "FirstName":"first_name".
	// Only applicable to top-level properties (including embedded).
	PropertyNameMapping map[string]string

	// ProcessWithoutTags enables processing fields without any tags specified.
	ProcessWithoutTags bool

	// UnnamedFieldWithTag enables a requirement that name tag is present
	// when processing _ fields to set up parent schema, e.g.
	//   _ struct{} `header:"_" additionalProperties:"false"`.
	UnnamedFieldWithTag bool

	// EnvelopNullability enables `anyOf` enveloping of "type":"null" instead of injecting into definition.
	EnvelopNullability bool

	InlineRefs   bool
	RootRef      bool
	RootNullable bool

	// SkipEmbeddedMapsSlices disables shortcutting into embedded maps and slices.
	SkipEmbeddedMapsSlices bool

	// InterceptType is called before and after type processing.
	// So it may be called twice for the same type, first time with empty Schema and
	// second time with fully processed schema.
	//
	// Deprecated: use InterceptSchema.
	InterceptType InterceptTypeFunc

	// interceptSchema is called before and after type Schema processing.
	// So it may be called twice for the same type, first time with empty Schema and
	// second time with fully processed schema.
	interceptSchema InterceptTypeFunc

	// Deprecated: Use interceptProp.
	InterceptProperty InterceptPropertyFunc

	interceptProp        InterceptPropFunc
	InterceptNullability InterceptNullabilityFunc

	// SkipNonConstraints disables parsing of `default` and `example` field tags.
	SkipNonConstraints bool

	// SkipUnsupportedProperties skips properties with unsupported types (func, chan, etc...) instead of failing.
	SkipUnsupportedProperties bool

	Path           []string
	definitions    map[refl.TypeString]*Schema // list of all definition objects
	definitionRefs map[refl.TypeString]Ref
	typeCycles     map[refl.TypeString]bool
	rootDefName    string
}

func (rc *ReflectContext) getDefinition(ref string) *Schema {
	for ts, r := range rc.definitionRefs {
		if r.Path+r.Name == ref {
			return rc.definitions[ts]
		}
	}

	return &Schema{}
}
