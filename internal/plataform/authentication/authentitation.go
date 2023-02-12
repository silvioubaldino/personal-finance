package authentication

type Auth interface {
	ValidToken(key string) (string, error)
}
