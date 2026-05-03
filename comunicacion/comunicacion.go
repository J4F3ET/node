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

// ServicioComunicacion gestiona el ciclo de vida del envío de datos dependiendo del estado del nodo.
func ServicioComunicacion(miID int, miHost string, chanLider chan string) {
	var liderActual string
	tieneLider := false
	dataSendInterval := 5 * time.Second

	ticker := time.NewTicker(dataSendInterval)
	defer ticker.Stop()

	for {
		select {
		// Escucha actualizaciones del estado del líder desde el módulo de coordinación
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
			// Si no hay líder conocido, permanecemos en espera
			if !tieneLider {
				log.Println("💤 [COM] Estado: STANDBY (Buscando líder...)")
				continue
			}

			// Si yo soy el líder, no me envío datos a mí mismo, solo proceso localmente
			if liderActual == miHost {
				log.Println("👑 [COM] Soy el Líder. Procesando datos locales...")
				continue
			}

			// Envío de datos médicos al líder actual
			err := EnviarDatosMedicos(liderActual, miHost, miID)
			if err != nil {
				log.Printf("🚨 [COM] Líder %s inalcanzable al enviar datos: %v", liderActual, err)
				// No reseteamos el líder aquí; la coordinación decidirá si el líder murió
				// por la ausencia de Heartbeats (ElectionTimeout).
			}
		}
	}
}

// IniciarServidorMedico levanta el servidor TCP que recibe los JSON de datos médicos.
func IniciarServidorMedico(host string) {
	// Escuchamos explícitamente en tcp4 para evitar problemas de dual-stack en Windows
	ln, err := net.Listen("tcp4", config.PuertoServicio)
	if err != nil {
		log.Printf("🚨 [SERVER] Error crítico al iniciar servidor médico en %s: %v", host, err)
		return
	}
	defer ln.Close()
	log.Printf("🏥 [SERVER] Escuchando datos médicos en %s", host)

	for {
		// Aceptamos conexiones entrantes
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("[SERVER] Error al aceptar conexión: %v", err)
			continue
		}
		go func(c net.Conn) {
			defer c.Close()
			remoteAddr := c.RemoteAddr().String()
			var msg Mensaje
			// Decodificamos el mensaje JSON recibido
			if err := json.NewDecoder(c).Decode(&msg); err == nil {
				log.Printf("[SERVER] Datos médicos de %s (%s) (ID: %d): %s", msg.Host, remoteAddr, msg.ID, msg.Contenido)
			}
			c.Write([]byte("ACK\n"))
		}(conn)
	}
}

// EnviarDatosMedicos encapsula la lógica de conexión y envío de un mensaje de tipo DATA.
func EnviarDatosMedicos(destino, miHost string, miID int) error {
	conn, err := net.DialTimeout("tcp", destino+config.PuertoServicio, 3*time.Second)
	if err != nil {
		return fmt.Errorf("error de conexión a %s: %w", destino, err)
	}
	defer conn.Close()

	// Construcción del payload médico
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