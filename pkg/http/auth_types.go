package http

type Auth struct {
}

type BasicAuth struct {
	Auth
	Username string
	Password string
}

type OAuth struct {
	Auth
	Token string
}
