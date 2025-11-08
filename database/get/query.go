package get

import "fmt"

type Query struct {
	Db      string
	Table   string
	Columns []string // Columns to select, empty means all columns
	Filters []Filter // Conditions to filter the results
}

// or nested filters
//
//	q := get.Query{
//		Db:      "mydb",
//		Table:   "mytable",
//		Columns: []string{"name", "age"},
//		Filters: []Filter{{Field: "age", Operator: ">", Value: 30}},
//		Filters: []Filter{
//			{Column: "age", Operator: ">", Operand2: 30},
//			{Column: "name", Operator: "=", SubFilter: []Filter{
//				{Column: "John", Operator: "=", Operand2: "john"},
//		},
//	}
//
// return array of objects
func EvaluateQuery(q Query) ([]map[string]any, error) {
	//filter primary keys
	pks, err := GetPksFromFilters(q.Db, q.Table, q.Filters)
	if err != nil {
		return nil, err
	}
	//if no pks found, return empty
	if len(pks) == 0 {
		return nil, nil
	}
	//get data with pks
	data, err := GetWithPKs(q.Db, q.Table, pks)
	// fmt.Println("Data:", data)
	for _, row := range data {
		// Convert all values to string for consistent output
		for k, v := range row {
			fmt.Printf("Key: %s, Value: %v\n", k, v)
		}
	}
	if err != nil {
		return nil, err
	}
	//get specific data with columns
	return data, nil

}
