package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"
)

const Puerto = ":5000"

func main() {
	// Uso: go run main.go [mi_id] [dominio_tailnet]
	if len(os.Args) < 3 {
		fmt.Println("Uso: go run main.go <MI_ID_NUMERICO> <DOMINIO_TAILNET>")
		return
	}

	miID, _ := strconv.Atoi(os.Args[1])
	dominio := os.Args[2]
	miHost := fmt.Sprintf("hospital_%d.%s", miID, dominio)

	fmt.Printf("🏥 Nodo [%s] iniciado.\n", miHost)

	for {
		liderEncontrado := ""
		
		// Buscamos candidatos con ID menor al nuestro (jerarquía)
		// Si miID es 1, este bucle no se ejecuta y salta directo a ser líder.
		for i := 1; i < miID; i++ {
			candidato := fmt.Sprintf("hospital_%d.%s", i, dominio)
			fmt.Printf("🔍 ¿Es [%s] el líder? Probando... ", candidato)
			
			conn, err := net.DialTimeout("tcp", candidato+Puerto, 1*time.Second)
			if err == nil {
				conn.Close()
				fmt.Println("✅ SÍ")
				liderEncontrado = candidato
				break
			}
			fmt.Println("❌ No")
		}

		if liderEncontrado == "" {
			// No hay nadie mejor que yo disponible, asumo el liderazgo
			fmt.Printf("👑 No se hallaron líderes menores. [%s] asume la coordinación.\n", miHost)
			iniciarServidor(miHost)
			// Si iniciarServidor retorna (error), el bucle for principal reintenta
		} else {
			// Encontré a alguien, me quedo como cliente reportando datos
			fmt.Printf("📡 Conectado al líder actual: [%s]\n", liderEncontrado)
			mantenerComoCliente(miHost, liderEncontrado)
		}
		
		time.Sleep(2 * time.Second)
	}
}

func iniciarServidor(host string) {
	ln, err := net.Listen("tcp", Puerto)
	if err != nil {
		fmt.Printf("⚠️ Error: Puerto ocupado o falla de red: %v\n", err)
		return
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		go func(c net.Conn) {
			defer c.Close()
			msg, _ := bufio.NewReader(c).ReadString('\n')
			if msg != "" {
				fmt.Printf("📥 [LÍDER] Reporte recibido: %s", msg)
				c.Write([]byte("ACK: Datos guardados\n"))
			}
		}(conn)
	}
}

func mantenerComoCliente(miHost, hostLider string) {
	for {
		conn, err := net.DialTimeout("tcp", hostLider+Puerto, 2*time.Second)
		if err != nil {
			fmt.Printf("🚨 Se perdió conexión con el líder %s. Buscando nuevo candidato...\n", hostLider)
			return // Sale para que el main() busque un nuevo líder
		}
		
		// Simulación de envío de datos médicos
		fmt.Fprintf(conn, "INFO: Nodo %s funcionando correctamente\n", miHost)
		
		resp, _ := bufio.NewReader(conn).ReadString('\n')
		fmt.Printf("🏥 Respuesta líder: %s", resp)
		conn.Close()
		
		time.Sleep(5 * time.Second)
	}
}