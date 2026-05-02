package coordinacion

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"node/comunicacion"
	"node/config"
	"time"
)

func ServicioCoordinacion(miID int, dominio, miHost string, chanLider chan string) {
	liderLocal := ""
	var ultimoLatido time.Time
	soyLider := false

	// Escuchar latidos de líderes (Broadcast simulado)
	go func() {
		ln, err := net.Listen("tcp", config.PuertoCoordinacion)
		if err != nil {
			log.Fatalf("Fallo en listener de coordinación: %v", err)
		}
		defer ln.Close()

		for {
			conn, err := ln.Accept()
			if err != nil {
				continue
			}
			go func(c net.Conn) {
				defer c.Close()
				var msg comunicacion.Mensaje
				if err := json.NewDecoder(c).Decode(&msg); err == nil {
					// Lógica de supresión de líderes duplicados
					if msg.ID < miID && soyLider {
						log.Printf("[COORD] Detectado líder con mayor prioridad (%s). Abdicando...", msg.Host)
						soyLider = false
						liderLocal = msg.Host
						chanLider <- liderLocal
					}
					
					if msg.ID <= miID { // Aceptamos al líder si tiene igual o mayor prioridad
						ultimoLatido = time.Now()
						if liderLocal != msg.Host {
							liderLocal = msg.Host
							select {
							case chanLider <- liderLocal:
							default:
							}
							log.Printf("[COORD] Siguiendo a nuevo líder: %s", liderLocal)
						}
					}
				}
			}(conn)
		}
	}()

	ticker := time.NewTicker(1 * time.Second)
	for range ticker.C {
		if !soyLider && (liderLocal == "" || time.Since(ultimoLatido) > config.ElectionTimeout) {
			if liderLocal != "" {
				log.Printf("[COORD] Tiempo de espera de líder %s agotado. Iniciando elección...", liderLocal)
				liderLocal = ""
				select {
				case chanLider <- "":
				default:
				}
			}
			log.Println("[COORD] Sin líder. Iniciando elección...")
			soyElMasBajo := true

			for i := 1; i < miID; i++ {
				candidato := fmt.Sprintf("hospital-%d.%s", i, dominio)
				conn, err := net.DialTimeout("tcp", candidato+config.PuertoServicio, config.DefaultTimeout)
				if err == nil {
					conn.Close()
					liderLocal = candidato
					chanLider <- candidato
					soyElMasBajo = false
					log.Printf("[COORD] Líder encontrado (ID menor): %s", liderLocal)
					break
				}
			}

			if soyElMasBajo {
				liderLocal = miHost
				if !soyLider {
					soyLider = true
					go comunicacion.IniciarServidorMedico(miHost)
					select {
					case chanLider <- miHost:
					default:
					}
				}
				log.Printf("[COORD] Yo soy el líder: %s", miHost)
			}
		}

		if soyLider {
			NotificarPares(miID, dominio, miHost)
		}
	}
}

func NotificarPares(miID int, dominio, miHost string) {
	for i := 1; i <= config.MaxNodes; i++ {
		if i == miID {
			continue
		}
		peer := fmt.Sprintf("hospital-%d.%s", i, dominio)
		go func(p string) {
			conn, err := net.DialTimeout("tcp", p+config.PuertoCoordinacion, 100*time.Millisecond)
			if err != nil {
				return
			}
			defer conn.Close()
			
			msg := comunicacion.Mensaje{
				Tipo: "HEARTBEAT",
				ID:   miID,
				Host: miHost,
			}
			json.NewEncoder(conn).Encode(msg)
		}(peer)
	}
}