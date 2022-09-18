package models

import "errors"

var (
	ErrNotFound      = errors.New("not found")
	ErrMultipleItems = errors.New("unexpected multiple items retrieved")
	ErrUnimplemented = errors.New("not yet implemented")
)
