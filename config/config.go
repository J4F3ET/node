package config

import "time"

const (
	PuertoServicio     = ":5000"
	PuertoCoordinacion = ":5001"
	MaxNodes           = 199 // Máximo según Readme.md
	DefaultTimeout     = 2 * time.Second
	
	// Tiempos para el algoritmo de salud
	HeartbeatInterval = 2 * time.Second
	ElectionTimeout   = 10 * time.Second
)