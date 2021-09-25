package csv

import (
	"bufio"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type mapping struct {
	field       field
	columnIndex int
}

type field struct {
	columnName string
	index      int
	kind       reflect.Kind
}

type Reader struct {
	reader  *bufio.Reader
	mapping []mapping

	// Parsing options
	Delimiter  byte
	TimeLayout string
}

const (
	defaultDelimiter  = ','
	defaultTimeLayout = time.RFC3339
	csvTag            = "csv"
)

func NewReader(r io.Reader) *Reader {
	br := bufio.NewReader(r)
	return &Reader{
		reader:     br,
		Delimiter:  defaultDelimiter,
		TimeLayout: defaultTimeLayout}
}

func (r Reader) trim(str string) string {
	str = strings.Trim(str, string([]byte{'\t', ' '}))
	if str[0] == '"' && str[len(str)-1] == '"' {
		str = str[1 : len(str)-1]
	}

	return string(str)
}

func (r Reader) parseLine() ([]string, error) {
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

		if lineBytes[i] == r.Delimiter && !inQuatations {
			v := r.trim(string(lineBytes[pos:i]))
			result = append(result, v)
			pos = i + 1
		}
	}

	// handle the last string at the end
	if i > pos {
		v := r.trim(string(lineBytes[pos:i]))
		result = append(result, v)
	}

	return result, nil
}

func (r *Reader) mapFields(t reflect.Type) error {
	columns, err := r.parseLine()
	if err != nil {
		return err
	}
	fields := r.getFields(t)

FieldsLoop:
	for _, field := range fields {

		for colIndex, colName := range columns {
			if field.columnName == colName {
				r.mapping = append(r.mapping, mapping{field, colIndex})
				continue FieldsLoop
			}
		}

		return fmt.Errorf("column %s was not found", field.columnName)
	}

	return nil
}

func (r Reader) getFields(t reflect.Type) (fields []field) {
	for fieldIndex := 0; fieldIndex < t.NumField(); fieldIndex++ {
		fieldInfo := t.Field(fieldIndex)

		if fieldInfo.IsExported() {
			colName := fieldInfo.Tag.Get(csvTag)

			if colName == "" {
				colName = fieldInfo.Name
			}

			fields = append(fields, field{colName, fieldIndex, fieldInfo.Type.Kind()})
		}
	}

	return fields
}

func (r Reader) setFieldValue(value string, elemField reflect.Value, kind reflect.Kind) error {
	switch kind {
	case reflect.String:
		elemField.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		valueInt, err := strconv.ParseInt(value, 0, 64)
		if err != nil {
			return fmt.Errorf("can not parse '%s' as int", value)
		}
		elemField.SetInt(valueInt)
	case reflect.Float32, reflect.Float64:
		float, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("can not parse '%s' as float", value)
		}
		elemField.SetFloat(float)
	case reflect.ValueOf(time.Time{}).Kind():
		t, err := time.Parse(r.TimeLayout, value)
		if err != nil {
			return fmt.Errorf("can not parse '%s' as time.Time", value)
		}

		elemField.Set(reflect.ValueOf(t))
	default:
		return fmt.Errorf("unsupported type %s", kind)
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
	err := r.mapFields(itemType)
	if err != nil {
		return err
	}

	slice := reflect.ValueOf(i).Elem()
	for lineNumber := 1; ; lineNumber++ {
		cols, err := r.parseLine()
		if err == io.EOF {
			return nil
		}

		if err != nil {
			return err
		}

		item := reflect.New(itemType)

		for i, m := range r.mapping {
			elemField := item.Elem().Field(m.field.index)
			value := cols[m.columnIndex]
			err := r.setFieldValue(value, elemField, elemField.Kind())
			if err != nil {
				return fmt.Errorf("line %d column %d: %s", lineNumber, i+1, err)
			}
		}

		slice.Set(reflect.Append(slice, item.Elem()))
	}
}
