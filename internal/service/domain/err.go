package domain

import "errors"

var (
	ErrPersonNotFound      = errors.New("person not found")
	ErrPersonAlreadyExists = errors.New("person already exists")
	ErrNameSurnameRequired = errors.New("name and surname are required")
)
