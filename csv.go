package csv

import (
	"bufio"
	"fmt"
	"io"
	"reflect"
	"strconv"
)

type mapping []*field

type field struct {
	index int
	kind  reflect.Kind
}

type Reader struct {
	reader *bufio.Reader
}

func trimQuatations(str []byte) string {
	if str[0] == '"' && str[len(str)-1] == '"' {
		str = str[1 : len(str)-1]
	}

	return string(str)
}

func (r *Reader) parseLine() ([]string, error) {
	lineBytes, _, err := r.reader.ReadLine()
	if err != nil {
		return nil, err
	}

	var result []string
	var i, pos int
	var inQuatations bool

	for ; i < len(lineBytes); i++ {

		if lineBytes[i] == '"' {
			if !inQuatations {
				inQuatations = true
			} else {
				// escaping - if the next rune is also quotation remove current character
				if i+1 != len(lineBytes) && lineBytes[i+1] == '"' {
					lineBytes = append(lineBytes[:i], lineBytes[i+1:]...)
					continue
				}
				inQuatations = false
			}
		}

		if lineBytes[i] == ',' && !inQuatations {
			v := trimQuatations(lineBytes[pos:i])
			result = append(result, v)
			pos = i + 1
		}
	}

	// handle the last string at the end
	if i > pos {
		v := trimQuatations(lineBytes[pos:i])
		result = append(result, v)
	}

	return result, nil
}

func mapColumns(t reflect.Type, columns []string) mapping {
	mapping := make(mapping, len(columns))

	// struct fields
	for colIndex, colName := range columns {
		for fieldIndex := 0; fieldIndex < t.NumField(); fieldIndex++ {
			fieldInfo := t.Field(fieldIndex)
			columnNameTag := fieldInfo.Tag.Get("columnName")

			// skip if field does not containg `columnName` tag
			if columnNameTag == "" {
				break
			}

			if fieldInfo.IsExported() {
				if columnNameTag == colName {
					mapping[colIndex] = &field{fieldIndex, fieldInfo.Type.Kind()}
					break
				}
			}
		}
	}

	return mapping
}

func NewReader(r io.Reader) *Reader {
	br := bufio.NewReader(r)
	return &Reader{br}
}

func setFieldValue(value string, elemField reflect.Value, kind reflect.Kind) error {
	switch kind {
	case reflect.String:
		elemField.SetString(value)
	case reflect.Int32:
		valueInt, err := strconv.ParseInt(value, 0, 32)
		if err != nil {
			return fmt.Errorf("can not parse '%s' as int32", value)
		}
		elemField.SetInt(valueInt)
	default:
		return fmt.Errorf("unsupported type")
	}

	return nil
}

func (r *Reader) ReadAll(i interface{}) error {
	// type extract
	t := reflect.TypeOf(i)
	if t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Slice || t.Elem().Elem().Kind() != reflect.Struct {
		return fmt.Errorf("input type must be a reference to a slice of structs")
	}

	itemType := t.Elem().Elem()
	header, _ := r.parseLine()
	mapping := mapColumns(itemType, header)
	slice := reflect.ValueOf(i).Elem()

	for lineNumber := 1; ; lineNumber++ {
		cols, err := r.parseLine()
		if err == io.EOF {
			return nil
		}

		if err != nil {
			return err
		}

		if len(cols) != len(mapping) {
			return fmt.Errorf("wrong number of columns on line %d", lineNumber)
		}

		item := reflect.New(itemType)

		for i, value := range cols {
			// fmt.Println(i, value, item, mapping)
			field := mapping[i]
			if field != nil {
				elemField := item.Elem().Field(field.index)
				err := setFieldValue(value, elemField, field.kind)
				if err != nil {
					return fmt.Errorf("line %d column %d: %s", lineNumber, i+1, err)
				}
			}
		}

		slice.Set(reflect.Append(slice, item.Elem()))
	}
}
