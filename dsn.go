package froach

// DSN returns the DSN to connect to the cockroach server. Even if the server isn't running, a value will be returned
func DSN() (string, error) {
	v := "postgresql://root@localhost:26258/defaultdb?sslmode=disable"
	return v, nil
}
