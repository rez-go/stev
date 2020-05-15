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

// LoadEnv loads the values and put them into target using default Loader.
func LoadEnv(prefix string, target interface{}) error {
	return defaultLoader.LoadEnv(prefix, target)
}

// Loader is [TBD]
//
//TODO: config: field error ignore (best effort), no override,
// tag name conversion (e.g., from FieldName to FIELD_NAME), ignore untagged
type Loader struct {
	StructFieldTagKey      string
	NamespaceSeparator     string
	IgnoredStructFieldName string
	SquashStructFieldName  string
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

// LoadEnv loads values into target from environment variables.
func (l Loader) LoadEnv(prefix string, target interface{}) error {
	_, err := l.loadEnv(prefix, target, false, false)
	if err != nil {
		return fmt.Errorf("stev: %w", err)
	}
	return nil
}

type fieldKey struct {
	fieldName string
	lookupKey string
}

func (l Loader) loadEnv(
	prefix string,
	target interface{},
	parentRequired bool,
	reqSuppress bool,
) (loadedAny bool, err error) {
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
			loadedAny, err = l.loadEnv(prefix, structVal.Interface(), parentRequired, true)
			if loadedAny {
				tVal.Set(structVal)
			}
		} else {
			loadedAny, err = l.loadEnv(prefix, tVal.Interface(), parentRequired, reqSuppress)
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
					lookupKey = prefix + fTagName
				}
				if strVal, exists := os.LookupEnv(lookupKey); exists {
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
				fieldPrefix = prefix
			} else {
				if fTagOpts.NoPrefix {
					fieldPrefix = fTagName + nsSep
				} else {
					fieldPrefix = prefix + fTagName + nsSep
				}
			}
			fieldLoaded, err := l.loadEnv(fieldPrefix, fVal.Addr().Interface(),
				fTagOpts.Required || parentRequired, true)
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
			fMap, ok := fVal.Interface().(map[string]interface{})
			if !ok {
				return loadedAny, fmt.Errorf("map requires an instance of type map[string]interface{}")
			}
			var fmBasePrefix string
			if fTagOpts.Squash {
				fmBasePrefix = prefix
			} else {
				if fTagOpts.NoPrefix {
					fmBasePrefix = fTagName + nsSep
				} else {
					fmBasePrefix = prefix + fTagName + nsSep
				}
			}
			for fMapKey, fMapValue := range fMap {
				fmVal := reflect.ValueOf(fMapValue)
				fmType := fmVal.Type()
				if fmType.Kind() != reflect.Ptr {
					return false, fmt.Errorf("requires pointer target (field %s key %s)", fInfo.Name, fMapKey)
				}
				// Notes: might try to instantiate, but we won't support it for now.
				if fmVal.IsNil() && !fmVal.CanSet() {
					return false, fmt.Errorf("requires settable target (field %s key %s)", fInfo.Name, fMapKey)
				}
				fmPrefix := fmBasePrefix + strings.ToUpper(fMapKey) + nsSep
				mapEntryLoaded, err := l.loadEnv(fmPrefix, fmVal.Interface(),
					fTagOpts.Required || parentRequired, true)
				if err != nil {
					return loadedAny, fmt.Errorf("map entry loading failed: %w (field %s key %s)",
						err, fInfo.Name, fMapKey)
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
			lookupKey = prefix + fTagName
		}
		if strVal, exists := os.LookupEnv(lookupKey); exists {
			fieldLoaded, err := l.loadFieldValue(strVal, fVal)
			if err != nil {
				return loadedAny, fmt.Errorf("unable to load field value (field %s key %s): %w",
					fInfo.Name, lookupKey, err)
			}
			loadedAny = loadedAny || fieldLoaded
			continue
		} else {
			if fTagOpts.Required {
				if parentRequired || !reqSuppress {
					return loadedAny, fmt.Errorf("field is required (field %s key %s)",
						fInfo.Name, lookupKey)
				}
				unsatisfiedFields = append(unsatisfiedFields, fieldKey{fInfo.Name, lookupKey})
			}
		}
	}

	if loadedAny && len(unsatisfiedFields) > 0 {
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
		}
	}
	return opts, nil
}
