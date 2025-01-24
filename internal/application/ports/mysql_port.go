package ports

type MySQLPort interface {
	FetchData() ([]map[string]interface{}, error)
	Close() error
}
