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
	if len(os.Args) < 3 {
		fmt.Println("Uso: go run main.go <ID> <DOMINIO>")
		os.Exit(1)
	}

	miID, err := strconv.Atoi(os.Args[1])
	if err != nil || miID < 1 || miID > config.MaxNodes {
		log.Fatalf("Error: ID inválido. Debe estar entre 1 y %d", config.MaxNodes)
	}

	dominio := os.Args[2]
	// Usamos hospital- para coincidir con el hostname de Tailscale y los subpaquetes
	miHost := fmt.Sprintf("hospital-%d.%s", miID, dominio)

	// Verificación de unicidad: ¿Alguien más está usando este ID en la red?
	conn, err := net.DialTimeout("tcp", miHost+config.PuertoServicio, 2*time.Second)
	if err == nil {
		conn.Close()
		log.Fatalf("❌ ERROR: El ID %d ya está en uso por otro nodo activo en la red (%s). Abortando para evitar conflictos.", miID, miHost)
	}

	chanLider := make(chan string, 1) // Canal con buffer para evitar bloqueos

	fmt.Printf("🚀 Nodo [%s] en línea.\n", miHost)

	go coordinacion.ServicioCoordinacion(miID, dominio, miHost, chanLider)
	comunicacion.ServicioComunicacion(miID, miHost, chanLider)
}