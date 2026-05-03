package config

import "time"

const (
	PuertoServicio     = ":5000" // Puerto para recepción de datos médicos
	PuertoCoordinacion = ":5001" // Puerto para intercambio de latidos y elecciones
	MaxNodes           = 199     // Límite superior de IDs de nodos en la red
	DefaultTimeout     = 2 * time.Second // Tiempo de espera para conexiones de red
	
	// Configuración del algoritmo de salud y elección
	HeartbeatInterval = 2 * time.Second // Frecuencia con la que el líder envía señales de vida
	ElectionTimeout   = 10 * time.Second // Tiempo sin latidos antes de declarar muerto al líder
)