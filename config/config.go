package config

import "time"

const (
	PuertoServicio     = ":5000"
	PuertoCoordinacion = ":5001"
	MaxNodes           = 199 // Máximo según Readme.md
	DefaultTimeout     = 500 * time.Millisecond
)