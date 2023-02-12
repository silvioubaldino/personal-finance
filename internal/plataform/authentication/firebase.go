package authentication

type firebase struct{}

func NewFirebaseAuth() Auth {
	return firebase{}
}

func (f firebase) ValidToken(key string) (string, error) {
	return "userID", nil
}
