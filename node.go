package main

import (
	"fmt"
	"log"
	"net"
	"node/comunicacion"
	"node/config"
	"node/coordinacion"
	"os"
	"strconv"
	"time"
)

func main() {
	// Validación de argumentos de entrada
	if len(os.Args) < 3 {
		fmt.Println("Uso: go run main.go <ID> <DOMINIO>")
		os.Exit(1)
	}

	// Conversión del ID de string a entero
	miID, err := strconv.Atoi(os.Args[1])
	if err != nil || miID < 1 || miID > config.MaxNodes {
		log.Fatalf("Error: ID inválido. Debe estar entre 1 y %d", config.MaxNodes)
	}

	dominio := os.Args[2]
	// Usamos el prefijo configurado para coincidir con el hostname de Tailscale
	miHost := fmt.Sprintf(config.NodeHostnameFormat, config.NodePrefix, miID, dominio)

	// Verificación de unicidad: ¿Alguien más está usando este ID en la red?
	// Intentamos conectar al puerto de servicio de nuestro propio hostname esperado
	conn, err := net.DialTimeout("tcp", miHost+config.PuertoServicio, 2*time.Second)
	if err == nil {
		conn.Close()
		log.Fatalf("❌ ERROR: El ID %d ya está en uso por otro nodo activo en la red (%s). Abortando para evitar conflictos.", miID, miHost)
	}

	// Canal para comunicar cambios de liderazgo entre el módulo de coordinación y el de comunicación
	chanLider := make(chan string, 1) // Canal con buffer para evitar bloqueos

	fmt.Printf("🚀 Nodo [%s] en línea.\n", miHost)

	// Iniciamos los servicios en segundo plano
	go coordinacion.ServicioCoordinacion(miID, dominio, miHost, chanLider)
	comunicacion.ServicioComunicacion(miID, miHost, chanLider)
}