package environment

import "os"

const (
	Production = "production"
	Staging    = "staging"
	Local      = "local"
)

func GetEnvironment() string {
	environment := os.Getenv("ENVIRONMENT")
	if environment == "" {
		return Local
	}
	return environment
}

func IsProduction() bool {
	return GetEnvironment() == Production
}
