package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"time"
)

// Estructura para manejar el estado del nodo
type Nodo struct {
	IPPropia     string
	IPLider      string
	Puerto       string
	EsLider      bool
}

func main() {
	// Uso: go run main.go <IP_PROPIA> <IP_LIDER>
	if len(os.Args) < 3 {
		fmt.Println("❌ Error: Faltan argumentos.")
		fmt.Println("Uso: go run main.go [IP_Tailscale_Local] [IP_Tailscale_Lider]")
		return
	}

	app := Nodo{
		IPPropia: os.Args[1],
		IPLider:  os.Args[2],
		Puerto:   ":5000",
	}

	fmt.Printf("🚀 Iniciando Nodo Médico en: %s\n", app.IPPropia)

	// Lógica de decisión de Rol
	if app.IPPropia == app.IPLider {
		go app.servidorLider()
	} else {
		go app.clienteYMonitoreo()
	}

	// Mantener el proceso vivo
	select {}
}

// --- FUNCIONES DEL LÍDER ---
func (n *Nodo) servidorLider() {
	ln, err := net.Listen("tcp", n.Puerto)
	if err != nil {
		fmt.Printf("❌ No se pudo iniciar líder en %s: %v\n", n.Puerto, err)
		return
	}
	defer ln.Close()
	n.EsLider = true
	fmt.Printf("👑 ROL: LÍDER activo en la IP virtual %s\n", n.IPPropia)

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go n.atenderPeticion(conn)
	}
}

func (n *Nodo) atenderPeticion(conn net.Conn) {
	defer conn.Close()
	msg, _ := bufio.NewReader(conn).ReadString('\n')
	fmt.Printf("📥 Datos recibidos de [%s]: %s", conn.RemoteAddr(), msg)
	conn.Write([]byte("✅ Recibido por el Líder\n"))
}

// --- FUNCIONES DE RÉPLICA / CLIENTE ---
func (n *Nodo) clienteYMonitoreo() {
	for {
		// Intentar conexión de Heartbeat/Latido 
		direccionLider := n.IPLider + n.Puerto
		conn, err := net.DialTimeout("tcp", direccionLider, 2*time.Second)

		if err != nil {
			fmt.Printf("⚠️ Latido perdido con Líder (%s). Intentando reconexión...\n", n.IPLider)
			// Aquí se activaría la lógica de suplantación/failover [cite: 4, 14]
		} else {
			fmt.Printf("💓 Latido estable con: %s\n", n.IPLider)
			// Enviar un dato médico de prueba
			fmt.Fprintf(conn, "HISTORIAL: Paciente-X desde %s\n", n.IPPropia)
			
			resp, _ := bufio.NewReader(conn).ReadString('\n')
			fmt.Printf("🏥 Respuesta líder: %s", resp)
			conn.Close()
		}
		time.Sleep(5 * time.Second)
	}
}