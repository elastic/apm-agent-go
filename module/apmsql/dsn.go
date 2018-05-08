package apmsql

// DSNInfo contains information from a database-specific data source name.
type DSNInfo struct {
	// Database is the name of the specific database identified by the DSN.
	Database string

	// User is the username that the DSN specifies for authenticating the
	// database connection.
	User string
}

// DSNParserFunc is the type of a function that can be used for parsing a
// data source name, and returning the corresponding Info.
type DSNParserFunc func(dsn string) DSNInfo

func genericDSNParser(string) DSNInfo {
	return DSNInfo{}
}
