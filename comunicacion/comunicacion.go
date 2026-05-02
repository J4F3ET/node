package comunicacion

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"node/config"
	"time"
)

func ServicioComunicacion(miHost string, chanLider chan string) {
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

			err := EnviarDatosMedicos(liderActual, miHost)
			if err != nil {
				log.Printf("🚨 [COM] Líder %s inalcanzable al enviar datos: %v", liderActual, err)
				tieneLider = false
				liderActual = ""
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
			message, err := bufio.NewReader(c).ReadString('\n')
			if err != nil {
				log.Printf("[SERVER] Error al leer datos: %v", err)
				return
			}
			log.Printf("[SERVER] Recibido de %s: %s", c.RemoteAddr(), message)
			c.Write([]byte("ACK\n"))
		}(conn)
	}
}

func EnviarDatosMedicos(destino, miHost string) error {
	conn, err := net.DialTimeout("tcp", destino+config.PuertoServicio, 2*time.Second)
	if err != nil {
		return fmt.Errorf("error de conexión: %w", err)
	}
	defer conn.Close()

	_, err = fmt.Fprintf(conn, "DATA: %s reportando signos vitales.\n", miHost)
	if err != nil {
		return fmt.Errorf("error de envío: %w", err)
	}

	_, err = bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return fmt.Errorf("error de ACK: %w", err)
	}
	return nil
}