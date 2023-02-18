package authentication

import (
	"github.com/stretchr/testify/mock"
)

type Mock struct {
	mock.Mock
}

func (m *Mock) ValidToken(key string) (string, error) {
	args := m.Called(key)
	return args.String(0), args.Error(1)
}
