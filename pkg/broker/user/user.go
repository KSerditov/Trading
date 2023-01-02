package user

type User struct {
	Username     string `json:"username" bson:"username"`
	PasswordHash string `json:"-" bson:"-"`
	ID           string `json:"id" bson:"id"`
}

type LoginForm struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
