//go:generate gotoolkit gen .

//go:gogen: plugin:templategen -template ./templates/model.tmpl -output $FILE_model.go
//go:gogen: plugin:templategen -template ./templates/query.tmpl -output $FILE_query.go

package multi_template

// @Define(name=Table, tableName="orders", schema="public")
// @Define(name=Fields, id=int64, status=string, total=float64)
type Order struct {
	ID     int64
	Status string
	Total  float64
}

// @Define(name=Table, tableName="products", schema="public")
// @Define(name=Fields, id=int64, name=string, price=float64, stock=int)
type Product struct {
	ID    int64
	Name  string
	Price float64
	Stock int
}
