package comunicacion

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"node/config"
	"time"
)

// Mensaje es la estructura universal para la comunicación entre nodos
type Mensaje struct {
	Tipo    string `json:"tipo"`    // "DATA" o "HEARTBEAT"
	ID      int    `json:"id"`      // ID del emisor
	Host    string `json:"host"`    // Hostname del emisor
	Contenido string `json:"contenido,omitempty"`
}

func ServicioComunicacion(miID int, miHost string, chanLider chan string) {
	var liderActual string
	tieneLider := false
	dataSendInterval := 5 * time.Second

	ticker := time.NewTicker(dataSendInterval)
	defer ticker.Stop()

	for {
		select {
		case nuevoLider := <-chanLider:
			if nuevoLider == "" {
				liderActual = ""
				tieneLider = false
				log.Println("🚨 [COM] Líder perdido. Volviendo a STANDBY.")
			} else {
				liderActual = nuevoLider
				tieneLider = true
				log.Printf("🔄 [COM] Estado: CON LÍDER (%s)", liderActual)
			}
		case <-ticker.C:
			if !tieneLider {
				log.Println("💤 [COM] Estado: STANDBY (Buscando líder...)")
				continue
			}

			if liderActual == miHost {
				log.Println("👑 [COM] Soy el Líder. Procesando datos locales...")
				continue
			}

			err := EnviarDatosMedicos(liderActual, miHost, miID)
			if err != nil {
				log.Printf("🚨 [COM] Líder %s inalcanzable al enviar datos: %v", liderActual, err)
				// No reseteamos el líder aquí; la coordinación decidirá si el líder murió
				// por la ausencia de Heartbeats (ElectionTimeout).
			}
		}
	}
}

func IniciarServidorMedico(host string) {
	ln, err := net.Listen("tcp", config.PuertoServicio)
	if err != nil {
		log.Printf("Error al iniciar el servidor médico en %s: %v", host, err)
		return
	}
	defer ln.Close()
	log.Printf("🏥 [SERVER] Escuchando datos médicos en %s", host)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("[SERVER] Error al aceptar conexión: %v", err)
			continue
		}
		go func(c net.Conn) {
			defer c.Close()
			var msg Mensaje
			if err := json.NewDecoder(c).Decode(&msg); err == nil {
				log.Printf("[SERVER] Datos médicos de %s (ID: %d): %s", msg.Host, msg.ID, msg.Contenido)
			}
			c.Write([]byte("ACK\n"))
		}(conn)
	}
}

func EnviarDatosMedicos(destino, miHost string, miID int) error {
	conn, err := net.DialTimeout("tcp", destino+config.PuertoServicio, 2*time.Second)
	if err != nil {
		return fmt.Errorf("error de conexión: %w", err)
	}
	defer conn.Close()

	msg := Mensaje{
		Tipo:      "DATA",
		ID:        miID,
		Host:      miHost,
		Contenido: "Signos vitales normales (Simulado)",
	}

	err = json.NewEncoder(conn).Encode(msg)
	if err != nil {
		return fmt.Errorf("error de envío: %w", err)
	}
	return nil
}