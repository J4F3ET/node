package comunicacion

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"node/config"
	"os"
	"time"
)

// Mensaje es la estructura universal para la comunicación entre nodos
type Mensaje struct {
	Tipo    string `json:"tipo"`    // "DATA" o "HEARTBEAT"
	ID      int    `json:"id"`      // ID del emisor
	Host    string `json:"host"`    // Hostname del emisor
	Contenido string `json:"contenido,omitempty"`
}

// Estado global compartido para el servidor de escucha
var (
	soyLiderGlobal bool
	miIDGlobal     int
	dominioGlobal  string
)

// gestorComunicacion encapsula el estado y la lógica de envío para mejorar la mantenibilidad.
type gestorComunicacion struct {
	miID        int
	miHost      string
	dominio     string
	liderActual string
	tieneLider  bool
}

// ServicioComunicacion gestiona el ciclo de vida del envío de datos dependiendo del estado del nodo.
func ServicioComunicacion(miID int, miHost string, dominio string, chanLider chan string, chanMensajes chan string) {
	g := &gestorComunicacion{miID: miID, miHost: miHost, dominio: dominio}
	miIDGlobal = miID
	dominioGlobal = dominio

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case nuevoLider := <-chanLider:
			g.actualizarEstado(nuevoLider)
		case msgPersonalizado := <-chanMensajes:
			g.enviar(msgPersonalizado, true)
		case <-ticker.C:
			if !g.tieneLider {
				log.Println("💤 [COM] Estado: STANDBY (Buscando líder...)")
				continue
			}
			g.enviar("Signos vitales normales (Simulado)", false)
		}
	}
}

func (g *gestorComunicacion) actualizarEstado(host string) {
	if host == "" {
		g.liderActual, g.tieneLider = "", false
		soyLiderGlobal = false
		log.Println("🚨 [COM] Líder perdido. Volviendo a STANDBY.")
		return
	}
	g.liderActual, g.tieneLider = host, true
	soyLiderGlobal = (host == g.miHost)
	log.Printf("🔄 [COM] Estado: CON LÍDER (%s)", host)
}

func (g *gestorComunicacion) enviar(contenido string, esManual bool) {
	if !g.tieneLider {
		if esManual { log.Printf("⚠️ [COM] Sin líder. Mensaje descartado: %s", contenido) }
		return
	}

	if g.liderActual == g.miHost {
		if esManual {
			DifundirMensaje(g.miHost, g.miID, contenido, g.miID) // El emisor original es el líder mismo
		} else {
			log.Printf("👑 [COM] Soy el Líder. Procesando datos locales...")
		}
		return
	}

	if err := EnviarDatosMedicos(g.liderActual, g.miHost, g.miID, contenido); err != nil {
		log.Printf("🚨 [COM] Error al enviar a %s: %v", g.liderActual, err)
	}
}


// ServicioEntradaManual lee de la consola y envía mensajes al canal para ser procesados por ServicioComunicacion.
func ServicioEntradaManual(chanMensajes chan string) {
	scanner := bufio.NewScanner(os.Stdin)
	log.Println("💬 [MANUAL] Sistema de mensajes manuales activo. Escribe algo y presiona Enter para enviarlo al líder.")
	for scanner.Scan() {
		texto := scanner.Text()
		if texto != "" {
			chanMensajes <- texto
		}
	}
}

// DifundirMensaje envía el mensaje a todos excepto al emisor original y a sí mismo.
func DifundirMensaje(miHost string, miID int, contenido string, idEmisorOriginal int) {
	log.Printf("📡 [RELAY] Difundiendo mensaje a la red (omitido ID: %d)...", idEmisorOriginal)
	for i := 1; i <= config.MaxNodes; i++ {
		// No enviar a uno mismo ni al que originó el mensaje
		if i == miID || i == idEmisorOriginal {
			continue
		}
		peer := fmt.Sprintf(config.NodeHostnameFormat, config.NodePrefix, i, dominioGlobal)
		go EnviarDatosMedicos(peer, miHost, miID, contenido)
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
				
				// Si somos el líder y NO somos el autor, retransmitimos (Relay)
				if soyLiderGlobal && msg.Host != host {
					DifundirMensaje(host, miIDGlobal, msg.Contenido, msg.ID)
				}
			}
			c.Write([]byte("ACK\n"))
		}(conn)
	}
}

// EnviarDatosMedicos encapsula la lógica de conexión y envío de un mensaje de tipo DATA.
func EnviarDatosMedicos(destino, miHost string, miID int, contenido string) error {
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
		Contenido: contenido,
	}

	err = json.NewEncoder(conn).Encode(msg)
	if err != nil {
		return fmt.Errorf("error de envío: %w", err)
	}
	return nil
}