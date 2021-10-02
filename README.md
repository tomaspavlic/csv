# csv

The package provides a simple typed parser for csv/dsv files.

## Installation
```
go get github.com/tomaspavlic/csv
```

## Usage

### Import
```go
import (
    // import the package
	"github.com/tomaspavlic/csv"
)
```
### Define type
Only exportable fields are considered. The name of the field is used unless Tag `csv` is used to specify actual name of a column in CSV.
```go
// define struct type
type Taxable struct {
	Index       int
	Description string `csv:"Item"`
	Cost        float32
	Tax         float32
	Total       float32
}
```
### Read
Parses each line into a item of given slice

```go
func main() {
	response, _ := http.Get("https://people.sc.fsu.edu/~jburkardt/data/csv/taxables.csv")
    // define a slice with a type
	var taxables []Taxable
    // initiate the reader
	r := csv.NewReader(response.Body)
    // read all lines into the referenced slice 
	err := r.ReadAll(&taxables)

	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(taxables)
}
```

## Reader settings
TimeLayout - layout for parsing a time.Time, see [format](https://golang.org/src/time/format.go) (default time.RFC3339)

Delimiter - change delimiter (default comma ',')

## Supported types
Int, Int8, Int16, Int32, Int64, Float32, Float64, time.Time, String