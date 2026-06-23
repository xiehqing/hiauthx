package routes

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

const responseTimeFormat = "2006-01-02 15:04:05"

var timeType = reflect.TypeOf(time.Time{})

func formatResponseData(data any) any {
	return formatValue(reflect.ValueOf(data))
}

func formatValue(value reflect.Value) any {
	if !value.IsValid() {
		return nil
	}
	if value.Kind() == reflect.Interface {
		if value.IsNil() {
			return nil
		}
		return formatValue(value.Elem())
	}
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}
		if value.Type().Elem() == timeType {
			t := value.Elem().Interface().(time.Time)
			return t.Format(responseTimeFormat)
		}
		return formatValue(value.Elem())
	}
	if value.Type() == timeType {
		t := value.Interface().(time.Time)
		return t.Format(responseTimeFormat)
	}

	switch value.Kind() {
	case reflect.Struct:
		return formatStruct(value)
	case reflect.Slice, reflect.Array:
		result := make([]any, 0, value.Len())
		for i := 0; i < value.Len(); i++ {
			result = append(result, formatValue(value.Index(i)))
		}
		return result
	case reflect.Map:
		result := make(map[string]any, value.Len())
		iter := value.MapRange()
		for iter.Next() {
			result[fmt.Sprint(iter.Key().Interface())] = formatValue(iter.Value())
		}
		return result
	default:
		return value.Interface()
	}
}

func formatStruct(value reflect.Value) map[string]any {
	result := make(map[string]any)
	valueType := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := valueType.Field(i)
		if field.PkgPath != "" {
			continue
		}

		fieldValue := value.Field(i)
		if field.Anonymous {
			if embedded, ok := formatValue(fieldValue).(map[string]any); ok {
				for key, val := range embedded {
					result[key] = val
				}
			}
			continue
		}

		name, omitEmpty, skip := jsonFieldName(field)
		if skip {
			continue
		}
		if omitEmpty && fieldValue.IsZero() {
			continue
		}
		result[name] = formatValue(fieldValue)
	}
	return result
}

func jsonFieldName(field reflect.StructField) (string, bool, bool) {
	tag := field.Tag.Get("json")
	if tag == "-" {
		return "", false, true
	}
	if tag == "" {
		return field.Name, false, false
	}
	parts := strings.Split(tag, ",")
	name := parts[0]
	if name == "" {
		name = field.Name
	}
	omitEmpty := false
	for _, part := range parts[1:] {
		if part == "omitempty" {
			omitEmpty = true
			break
		}
	}
	return name, omitEmpty, false
}
