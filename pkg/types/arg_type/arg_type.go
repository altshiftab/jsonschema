package arg_type

// ArgType is an enumeration of the possible schema part types.
type ArgType int

const (
	ArgTypeBool ArgType = iota + 1
	ArgTypeString
	ArgTypeStrings
	ArgTypeStringOrStrings
	ArgTypeInt
	ArgTypeFloat
	ArgTypeSchema
	ArgTypeSchemas
	ArgTypeMapSchema
	ArgTypeSchemaOrSchemas
	ArgTypeMapArrayOrSchema
	ArgTypeAny
)

