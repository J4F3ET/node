package main

import (
	"fmt"
	"log"
	"node/comunicacion"
	"node/config"
	"node/coordinacion"
	"os"
	"strconv"
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

	chanLider := make(chan string)

	fmt.Printf("🚀 Nodo [%s] en línea.\n", miHost)

	go coordinacion.ServicioCoordinacion(miID, dominio, miHost, chanLider)
	comunicacion.ServicioComunicacion(miHost, chanLider)
}