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

// Loader is [TBD]
//
//TODO: config: prefix, tag name, field error ignore, no override,
// tag name conversion (e.g., from FieldName to FIELD_NAME), ignore untagged
type Loader struct {
	StructFieldTagName          string
	NamespaceSeparator          string
	IgnoredStructFieldTagName   string
	AnonymousStructFieldTagName string
}

// LoadEnv loads values into target from environment variables.
func (l Loader) LoadEnv(prefix string, target interface{}) error {
	_, err := l.loadEnv(prefix, target)
	return err
}

func (l Loader) loadEnv(prefix string, target interface{}) (loadedAny bool, err error) {
	tagName := l.StructFieldTagName
	nsSep := l.NamespaceSeparator

	tType := reflect.TypeOf(target)
	if tType.Kind() != reflect.Ptr {
		return false, errors.New("stev: requires pointer target " + tType.Kind().String())
	}
	tVal := reflect.ValueOf(target)
	if tVal.IsNil() && !tVal.CanSet() {
		return false, errors.New("stev: target is not settable")
	}

	tValDef := tVal.Elem()
	tType = tValDef.Type()
	if tType.Kind() == reflect.Ptr {
		if tValDef.IsNil() {
			structVal := reflect.New(tType.Elem())
			loadedAny, err = l.loadEnv(prefix, structVal.Interface())
			if loadedAny {
				tValDef.Set(structVal)
			}
		} else {
			loadedAny, err = l.loadEnv(prefix, tValDef.Interface())
		}
		return
	}

	for i := 0; i < tType.NumField(); i++ {
		fInfo := tType.Field(i)
		fVal := tValDef.Field(i)
		if fInfo.PkgPath != "" {
			continue
		}

		fTag := fInfo.Tag.Get(tagName)
		var fTagName string
		var fTagOpts fieldOptions
		if fTag != "" {
			fTagParts := strings.SplitN(fTag, ",", 2)
			fTagName = fTagParts[0]
			if len(fTagParts) > 1 {
				fTagOpts, _ = parseFieldOptions(fTagParts[1])
			}
		}
		if fTagName != "" {
			if fTagName == l.IgnoredStructFieldTagName {
				continue
			}
			if fTagName == l.AnonymousStructFieldTagName {
				fTagName = ""
				fTagOpts.Anonymous = true
			}
		} else {
			if !fInfo.Anonymous {
				fTagName = l.convertFieldName(fInfo.Name)
			} else {
				fTagOpts.Anonymous = true
			}
		}

		fType := fInfo.Type
		if fType.Kind() == reflect.Struct || (fType.Kind() == reflect.Ptr && fType.Elem().Kind() == reflect.Struct) {
			if strVal, exists := os.LookupEnv(prefix + fTagName); exists {
				fieldLoaded, err := l.loadFieldValue(strVal, fInfo.Name, fType, fVal)
				if err != nil {
					return loadedAny, err
				}
				loadedAny = loadedAny || fieldLoaded
				continue
			}

			var fieldPrefix string
			if fTagOpts.Anonymous {
				fieldPrefix = prefix
			} else {
				fieldPrefix = prefix + fTagName + nsSep
			}
			fieldLoaded, err := l.loadEnv(fieldPrefix, fVal.Addr().Interface())
			if err != nil {
				return loadedAny, fmt.Errorf("stev: unable to load value for field %s: %w", fInfo.Name, err)
			}
			if fieldLoaded && fTagOpts.Required {
				return loadedAny, errors.New("stev: field is required: " + fTagName)
			}
			loadedAny = loadedAny || fieldLoaded
			continue
		}

		if fTagOpts.Anonymous {
			return loadedAny, errors.New("stev: anonymous can only be used to field which type is struct")
		}

		if strVal, exists := os.LookupEnv(prefix + fTagName); exists {
			fieldLoaded, err := l.loadFieldValue(strVal, fInfo.Name, fType, fVal)
			if err != nil {
				return loadedAny, err
			}
			loadedAny = loadedAny || fieldLoaded
			continue
		} else {
			if fTagOpts.Required {
				return loadedAny, errors.New("stev: field is required: " + fTagName)
			}
		}
	}

	return
}

func (l Loader) loadFieldValue(
	strVal string, fieldName string, fieldType reflect.Type, fieldValue reflect.Value,
) (loaded bool, err error) {
	if fieldType.Kind() == reflect.Ptr {
		valType := fieldType.Elem()
		if fieldValue.IsNil() {
			valInst := reflect.New(valType)
			loaded, err = l.loadFieldValue(strVal, fieldName, valType, valInst.Elem())
			if loaded {
				fieldValue.Set(valInst)
			}
		} else {
			loaded, err = l.loadFieldValue(strVal, fieldName, valType, fieldValue.Elem())
		}
		return
	}

	switch fieldValue.Interface().(type) {
	case time.Duration:
		d, err := time.ParseDuration(strVal)
		if err != nil {
			return false, errors.New("stev: unable to parse duration for field " + fieldName)
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
			return false, fmt.Errorf("stev: value conversion error for field %s: %w",
				fieldName, err)
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
			return false, fmt.Errorf("stev: value conversion error for field %s: %w",
				fieldName, err)
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
			return false, fmt.Errorf("stev: value conversion error for field %s: %w",
				fieldName, err)
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
			return false, fmt.Errorf("stev: value conversion error for field %s: %w",
				fieldName, err)
		}
		fieldValue.SetUint(v)
		return true, nil
	case reflect.String:
		fieldValue.SetString(strVal)
		return true, nil
	default:
		return false, fmt.Errorf("stev: unsupported type for field %s: %v",
			fieldName, fieldType.Name())
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

// StructFieldTagNameDefault is the string we use to identify the struct field tag
// we must process.
const StructFieldTagNameDefault = "env"

// NamespaceSeparatorDefault is [TBD].
const NamespaceSeparatorDefault = "_"

// IgnoredStructFieldTagNameDefault is used to indicate struct fields which
// should be ignored.
const IgnoredStructFieldTagNameDefault = "-"

// AnonymousStructFieldTagNameDefault is used for to treat a field which has
// the type of struct as embedded. It's affecting the way we construct the
// key used to lookup the value from environment variables.
const AnonymousStructFieldTagNameDefault = "&"

var defaultLoader = Loader{
	StructFieldTagName:          StructFieldTagNameDefault,
	NamespaceSeparator:          NamespaceSeparatorDefault,
	IgnoredStructFieldTagName:   IgnoredStructFieldTagNameDefault,
	AnonymousStructFieldTagName: AnonymousStructFieldTagNameDefault,
}

// LoadEnv loads the values and put them into target using default Loader.
func LoadEnv(prefix string, target interface{}) error {
	return defaultLoader.LoadEnv(prefix, target)
}

type fieldOptions struct {
	Anonymous bool
	Required  bool
}

func parseFieldOptions(str string) (fieldOptions, error) {
	if str == "" {
		return fieldOptions{}, nil
	}
	opts := fieldOptions{}
	parts := strings.Split(str, ",")
	for _, s := range parts {
		switch s {
		case "anonymous":
			opts.Anonymous = true
		case "required":
			opts.Required = true
		}
	}
	return opts, nil
}
