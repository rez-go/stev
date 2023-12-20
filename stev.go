package stev

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// LoadFromEnv loads the values and put them into target using default Loader.
func LoadFromEnv(prefix string, target interface{}) error {
	return defaultLoader.LoadFromEnv(prefix, target)
}

var LoadEnv = LoadFromEnv

func Docs(prefix string, structure interface{}) ([]FieldDocs, error) {
	l := defaultLoader
	fieldDocs := []FieldDocs{}
	_, err := l.loadFromEnv(prefix, structure, false, false, "", &fieldDocs)
	if err != nil {
		return nil, err
	}
	return fieldDocs, nil
}

// EnvLookupFunc is a function signature which can be satisfied by os.LookupEnv.
type EnvLookupFunc = func(key string) (value string, ok bool)

// Loader is [TBD]
//
//TODO: config: field error ignore (best effort), no override,
// tag name conversion (e.g., from FieldName to FIELD_NAME), ignore untagged
type Loader struct {
	StructFieldTagKey      string
	NamespaceSeparator     string
	IgnoredStructFieldName string
	SquashStructFieldName  string

	lookupEnv EnvLookupFunc
}

// StructFieldTagKeyDefault is the string we use to identify the struct field tag
// we must process.
const StructFieldTagKeyDefault = "env"

// NamespaceSeparatorDefault is [TBD].
const NamespaceSeparatorDefault = "_"

// IgnoredStructFieldNameDefault is used to indicate struct fields which
// should be ignored.
const IgnoredStructFieldNameDefault = "-"

// SquashStructFieldNameDefault is used for to treat a field which has
// the type of struct as embedded. It's affecting the way we construct the
// key used to lookup the value from environment variables.
const SquashStructFieldNameDefault = "&"

var defaultLoader = Loader{
	StructFieldTagKey:      StructFieldTagKeyDefault,
	NamespaceSeparator:     NamespaceSeparatorDefault,
	IgnoredStructFieldName: IgnoredStructFieldNameDefault,
	SquashStructFieldName:  SquashStructFieldNameDefault,
}

// LoadFromEnv loads values into target from environment variables.
func (l Loader) LoadFromEnv(prefix string, target interface{}) error {
	_, err := l.loadFromEnv(prefix, target, false, false, "", nil)
	if err != nil {
		return fmt.Errorf("stev: %w", err)
	}
	return nil
}

// LoadEnv loads values into target from environment variables.
//
// Deprecated: Use LoadFromEnv.
func (l Loader) LoadEnv(prefix string, target interface{}) error {
	return l.LoadFromEnv(prefix, target)
}

type lookupEnvFunc = func(string) (string, bool)

type fieldKey struct {
	fieldName string
	lookupKey string
}

func (l Loader) loadFromEnv(
	lookupPrefix string,
	target interface{},
	parentIsRequired bool,
	reqCancel bool,
	fieldPath string,
	fieldDocs *[]FieldDocs,
) (loadedAny bool, err error) {
	lookupEnv := l.lookupEnv
	if lookupEnv == nil {
		lookupEnv = os.LookupEnv
	}
	docsMode := fieldDocs != nil

	tagName := l.StructFieldTagKey
	nsSep := l.NamespaceSeparator

	tVal := reflect.ValueOf(target)
	tType := tVal.Type()
	if tType.Kind() != reflect.Ptr {
		return false, errors.New("requires pointer target")
	}
	if tVal.IsNil() && !tVal.CanSet() {
		return false, errors.New("requires settable target")
	}

	tVal = tVal.Elem()
	tType = tVal.Type()
	if tType.Kind() == reflect.Ptr {
		if tVal.IsNil() {
			structVal := reflect.New(tType.Elem())
			loadedAny, err = l.loadFromEnv(lookupPrefix, structVal.Interface(),
				parentIsRequired, true, fieldPath, fieldDocs)
			if loadedAny {
				tVal.Set(structVal)
			}
		} else {
			loadedAny, err = l.loadFromEnv(lookupPrefix, tVal.Interface(),
				parentIsRequired, reqCancel, fieldPath, fieldDocs)
		}
		return
	}

	// Holds the list of fields which flagged as required but value was not provided
	var unsatisfiedFields []fieldKey

	for i := 0; i < tType.NumField(); i++ {
		fInfo := tType.Field(i)
		fVal := tVal.Field(i)
		if fInfo.PkgPath != "" {
			continue
		}

		fTag := fInfo.Tag.Get(tagName)
		var fTagName string
		var fTagOpts fieldTagOpts
		if fTag != "" {
			fTagParts := strings.SplitN(fTag, ",", 2)
			fTagName = fTagParts[0]
			if len(fTagParts) > 1 {
				fTagOpts, _ = parseFieldTagOpts(fTagParts[1])
			}
		}
		if fTagName != "" {
			if fTagName == l.IgnoredStructFieldName {
				continue
			}
			if fTagName == l.SquashStructFieldName {
				fTagName = ""
				fTagOpts.Squash = true
			}

			if strings.HasPrefix(fTagName, "!") {
				if fTagOpts.Squash {
					// Note that this should be possible but it'll be
					// quite complex (and there's probably no use case)
					return false, fmt.Errorf("cannot combine noprefix with squash (field %s)", fTagName)
				}
				fTagOpts.NoPrefix = true
				fTagName = strings.TrimPrefix(fTagName, "!")
				if fTagName == "" {
					fTagName = l.convertFieldName(fInfo.Name)
				}
			}
		} else {
			if !fInfo.Anonymous {
				fTagName = l.convertFieldName(fInfo.Name)
			} else {
				fTagOpts.Squash = true
			}
		}

		fType := fInfo.Type
		if fType.Kind() == reflect.Struct || (fType.Kind() == reflect.Ptr && fType.Elem().Kind() == reflect.Struct) {
			if fTagName != "" {
				var lookupKey string
				if fTagOpts.NoPrefix {
					lookupKey = fTagName
				} else {
					lookupKey = lookupPrefix + fTagName
				}

				//TODO: docs for field which type is an opaque struct. we will
				// provide an unmarshaller interface for field value types.
				// if the type of a field value conforms the interface, then
				// we'll treat it as opaque.

				if strVal, exists := lookupEnv(lookupKey); exists {
					fieldLoaded, err := l.loadFieldValue(strVal, fVal)
					if err != nil {
						return loadedAny, fmt.Errorf("unable to load field value (field %s key %s): %w",
							fInfo.Name, lookupKey, err)
					}
					loadedAny = loadedAny || fieldLoaded
					continue
				}
			}

			var fieldPrefix string
			if fTagOpts.Squash {
				fieldPrefix = lookupPrefix
			} else {
				if fTagOpts.NoPrefix {
					fieldPrefix = fTagName + nsSep
				} else {
					fieldPrefix = lookupPrefix + fTagName + nsSep
				}
			}
			fieldLoaded, err := l.loadFromEnv(fieldPrefix, fVal.Addr().Interface(),
				fTagOpts.Required || parentIsRequired, true, fieldPath+"."+fInfo.Name, fieldDocs)
			if err != nil {
				return loadedAny, fmt.Errorf("unable to load field value (field %s key %s*): %w",
					fInfo.Name, fieldPrefix, err)
			}
			if !fieldLoaded && fTagOpts.Required {
				return loadedAny, fmt.Errorf("field is required (field %s key %s*)",
					fInfo.Name, fieldPrefix)
			}
			loadedAny = loadedAny || fieldLoaded
			continue
		}

		if fType.Kind() == reflect.Map && fTagOpts.Map {
			if fType.Key().Kind() != reflect.String {
				return loadedAny, fmt.Errorf("map requires an instance of map with string key")
			}
			var fmBasePrefix string
			if fTagOpts.Squash {
				fmBasePrefix = lookupPrefix
			} else {
				if fTagOpts.NoPrefix {
					fmBasePrefix = fTagName + nsSep
				} else {
					fmBasePrefix = lookupPrefix + fTagName + nsSep
				}
			}
			for _, entryKey := range fVal.MapKeys() {
				mapEntryKey := entryKey.Interface().(string)
				mapEntryVal := fVal.MapIndex(entryKey).Interface()
				rmeVal := reflect.ValueOf(mapEntryVal)
				rmeType := rmeVal.Type()
				if rmeType.Kind() != reflect.Ptr {
					return false, fmt.Errorf("requires pointer target (field %s key %s)", fInfo.Name, mapEntryKey)
				}
				// Notes: might try to instantiate, but we won't support it for now.
				if rmeVal.IsNil() && !rmeVal.CanSet() {
					return false, fmt.Errorf("requires settable target (field %s key %s)", fInfo.Name, mapEntryKey)
				}
				fmPrefix := fmBasePrefix + strings.ToUpper(mapEntryKey) + nsSep
				mapEntryLoaded, err := l.loadFromEnv(fmPrefix, rmeVal.Interface(),
					fTagOpts.Required || parentIsRequired, true,
					fieldPath+"."+fInfo.Name+"["+mapEntryKey+": "+rmeType.String()+"]", fieldDocs)
				if err != nil {
					return loadedAny, fmt.Errorf("map entry loading failed: %w (field %s key %s)",
						err, fInfo.Name, mapEntryKey)
				}

				loadedAny = loadedAny || mapEntryLoaded
			}

			continue
		}

		if fTagOpts.Squash {
			return loadedAny, fmt.Errorf("squash can only be used to "+
				"field which type is struct or pointer "+
				"to struct (field %s)", fInfo.Name)
		}

		var lookupKey string
		if fTagOpts.NoPrefix {
			lookupKey = fTagName
		} else {
			lookupKey = lookupPrefix + fTagName
		}
		if !fTagOpts.DocsHidden && fieldDocs != nil {
			var desc string
			var descriptor *FieldDocsDescriptor
			var availableValues map[string]EnumValueDocs
			if fd, ok := target.(fieldDocsDescriptorProvider); ok {
				descriptor = fd.FieldDocsDescriptor(fInfo.Name)
				if descriptor == nil {
					descriptor = fd.FieldDocsDescriptor(fTagName)
				}
				if descriptor != nil {
					desc = descriptor.Description
					availableValues = descriptor.AvailableValues
				}
			}
			if desc == "" {
				if fd, ok := target.(namespacedFieldDescriptionsProvider); ok {
					fieldDescs := fd.StevFieldDescriptions()
					desc, ok = fieldDescs[fInfo.Name]
					if !ok {
						desc = fieldDescs[fTagName]
					}
				}
			}
			if desc == "" {
				if fd, ok := target.(fieldDescriptionsProvider); ok {
					fieldDescs := fd.FieldDescriptions()
					desc, ok = fieldDescs[fInfo.Name]
					if !ok {
						desc = fieldDescs[fTagName]
					}
				}
			}
			//TODO: use our own interface for converting the values from/to string
			var defVal string
			if fType.Kind() == reflect.Ptr && fVal.IsNil() {
				defVal = ""
			} else if !fVal.IsZero() {
				defVal = fmt.Sprintf("%v", fVal.Interface())
			}
			*fieldDocs = append(*fieldDocs, FieldDocs{
				LookupKey:       lookupKey,
				DataType:        fType.String(),
				Required:        fTagOpts.Required,
				Description:     strings.TrimSpace(desc),
				Value:           defVal,
				Path:            fieldPath + "." + fInfo.Name,
				AvailableValues: availableValues,
			})
		}
		if strVal, exists := lookupEnv(lookupKey); exists {
			fieldLoaded, err := l.loadFieldValue(strVal, fVal)
			if err != nil {
				return loadedAny, fmt.Errorf("unable to load field value (field %s key %s): %w",
					fInfo.Name, lookupKey, err)
			}
			loadedAny = loadedAny || fieldLoaded
			continue
		} else {
			if !docsMode && fTagOpts.Required {
				if parentIsRequired || !reqCancel {
					return loadedAny, fmt.Errorf("field is required (field %s key %s)",
						fInfo.Name, lookupKey)
				}
				unsatisfiedFields = append(unsatisfiedFields, fieldKey{fInfo.Name, lookupKey})
			}
		}
	}

	if !docsMode && loadedAny && len(unsatisfiedFields) > 0 {
		return loadedAny, fmt.Errorf("fields are required %v", unsatisfiedFields)
	}

	return
}

func (l Loader) loadFieldValue(
	strVal string, fieldValue reflect.Value,
) (loaded bool, err error) {
	fieldType := fieldValue.Type()
	if fieldType.Kind() == reflect.Ptr {
		valType := fieldType.Elem()
		if fieldValue.IsNil() {
			valInst := reflect.New(valType)
			loaded, err = l.loadFieldValue(strVal, valInst.Elem())
			if loaded {
				fieldValue.Set(valInst)
			}
		} else {
			loaded, err = l.loadFieldValue(strVal, fieldValue.Elem())
		}
		return
	}

	switch fieldValue.Interface().(type) {
	case time.Duration:
		d, err := time.ParseDuration(strVal)
		if err != nil {
			return false, err
		}
		fieldValue.Set(reflect.ValueOf(&d).Elem())
		return true, nil
	}

	switch fieldType.Kind() {
	case reflect.Bool:
		if strVal == "" {
			fieldValue.SetBool(true)
			return true, nil
		}
		v, err := strconv.ParseBool(strVal)
		if err != nil {
			return false, err
		}
		fieldValue.SetBool(v)
		return true, nil
	case reflect.Float32, reflect.Float64:
		if strVal == "" {
			fieldValue.SetFloat(0)
			return true, nil
		}
		v, err := strconv.ParseFloat(strVal, fieldType.Bits())
		if err != nil {
			return false, err
		}
		fieldValue.SetFloat(v)
		return true, nil
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if strVal == "" {
			fieldValue.SetInt(0)
			return true, nil
		}
		v, err := strconv.ParseInt(strVal, 0, fieldType.Bits())
		if err != nil {
			return false, err
		}
		fieldValue.SetInt(v)
		return true, nil
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if strVal == "" {
			fieldValue.SetUint(0)
			return true, nil
		}
		v, err := strconv.ParseUint(strVal, 0, fieldType.Bits())
		if err != nil {
			return false, err
		}
		fieldValue.SetUint(v)
		return true, nil
	case reflect.String:
		fieldValue.SetString(strVal)
		return true, nil
	default:
		return false, fmt.Errorf("unsupported field value type %q", fieldType.Name())
	}
}

func (l Loader) convertFieldName(fieldName string) string {
	if fieldName == "" {
		return ""
	}
	var outRunes []rune
	prevIsUpper := true
	for _, r := range fieldName {
		if unicode.IsUpper(r) || unicode.IsDigit(r) {
			if prevIsUpper {
				outRunes = append(outRunes, r)
				continue
			}
			outRunes = append(outRunes, '_', r)
			prevIsUpper = true
		} else {
			if prevIsUpper && len(outRunes) >= 2 {
				cR := outRunes[len(outRunes)-2]
				if unicode.IsUpper(cR) || unicode.IsDigit(cR) {
					tR := outRunes[len(outRunes)-1]
					outRunes[len(outRunes)-1] = '_'
					outRunes = append(outRunes, tR)
				}
			}
			outRunes = append(outRunes, r)
			prevIsUpper = false
		}
	}
	tagName := strings.ToUpper(string(outRunes))
	return tagName
}

type fieldTagOpts struct {
	NoPrefix bool
	Squash   bool
	Required bool
	Map      bool // Only for maps

	// Don't show the entry in the docs. This could be useful for
	// tuning fields to prevent them from distracting from the necessary
	// fields.
	DocsHidden bool
}

func parseFieldTagOpts(str string) (fieldTagOpts, error) {
	if str == "" {
		return fieldTagOpts{}, nil
	}
	opts := fieldTagOpts{}
	parts := strings.Split(str, ",")
	for _, s := range parts {
		switch s {
		case "anonymous", "squash":
			opts.Squash = true
		case "required":
			opts.Required = true
		case "map":
			opts.Map = true
		case "docs_hidden":
			opts.DocsHidden = true
		}
	}
	return opts, nil
}

type FieldDocs struct {
	LookupKey   string
	DataType    string
	Required    bool
	Description string

	// The value as provided through the skeleton. This might be
	// the default or the suggested value.
	Value string

	// Some fields' value is based on, e.g., registered components. This
	// field contains the available options, e.g., which component to use.
	//
	// The key is the option.
	AvailableValues map[string]EnumValueDocs

	Path string
}

// FieldDocsDescriptor provides detailed information for a field.
type FieldDocsDescriptor struct {
	Description string
	// The key is the the available value.
	AvailableValues map[string]EnumValueDocs
}

type SelfDocsDescriptor struct {
	// SortDesc should be a line of 50 characters or less.
	ShortDesc string
}

// LoadSelfDocsDescriptor attempts to load the docs-descriptor of a struct
// that implements SelfDocsDescriptor method.
func LoadSelfDocsDescriptor(
	skeleton interface{},
) *SelfDocsDescriptor {
	if fd, ok := skeleton.(selfDocsDescriptorProvider); ok {
		v := fd.SelfDocsDescriptor()
		return &v
	}
	return nil
}

// EnumValueDocs holds information about an enumerated value.
type EnumValueDocs struct {
	// SortDesc should be a line of 50 characters or less.
	ShortDesc string
}

type namespacedFieldDescriptionsProvider interface {
	//NOTE: deprecated
	StevFieldDescriptions() map[string]string
}

type fieldDescriptionsProvider interface {
	//NOTE: deprecated
	FieldDescriptions() map[string]string
}

type fieldDocsDescriptorProvider interface {
	FieldDocsDescriptor(fieldName string) *FieldDocsDescriptor
}

type selfDocsDescriptorProvider interface {
	SelfDocsDescriptor() SelfDocsDescriptor
}
