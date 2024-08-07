package types

const (
	URL         = "https://vodaplus.ru"
	CATALOG_URL = "https://vodaplus.ru/catalog"
	PAGE        = "?PAGEN_1="
)

type Product struct {
	Name  string
	URL   string
	Price string
}

type Catalog struct {
	Name     string
	URL      string
	Products []*Product
}
