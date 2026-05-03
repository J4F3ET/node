package coordinacion

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"node/comunicacion"
	"node/config"
	"sync"
	"time"
)

// ServicioCoordinacion implementa el algoritmo de elección de líder (Bully modificado) y el monitoreo de latidos.
func ServicioCoordinacion(miID int, dominio, miHost string, chanLider chan string) {
	var mu sync.Mutex
	liderLocal := ""
	ultimoLatido := time.Now()
	soyLider := false

	// Goroutine para escuchar mensajes entrantes en el puerto de coordinación
	go func() {
		ln, err := net.Listen("tcp4", config.PuertoCoordinacion)
		if err != nil {
			log.Fatalf("🚨 [COORD] Fallo en listener de coordinación: %v", err)
		}
		defer ln.Close()

		for {
			conn, err := ln.Accept()
			if err != nil {
				continue
			}
			go func(c net.Conn) {
				defer c.Close()
				remoteAddr := c.RemoteAddr().String()
				var msg comunicacion.Mensaje
				if err := json.NewDecoder(c).Decode(&msg); err == nil {
					mu.Lock()
					defer mu.Unlock()

					// Lógica de supresión: Si aparece alguien con ID menor, dejo de ser líder
					if msg.ID < miID && soyLider {
						log.Printf("📉 [COORD] Detectado líder con mayor prioridad (%s - %s). Abdicando...", msg.Host, remoteAddr)
						soyLider = false
						liderLocal = msg.Host
						select {
						case chanLider <- liderLocal:
						default:
						}
					}
					
					// Actualizamos el timestamp del último latido si el mensaje viene de un líder válido
					if msg.ID <= miID { // Aceptamos al líder si tiene igual o mayor prioridad
						ultimoLatido = time.Now()
						if liderLocal != msg.Host {
							liderLocal = msg.Host
							select {
							case chanLider <- liderLocal:
							default:
							}
							log.Printf("🔭 [COORD] Siguiendo a nuevo líder: %s (%s)", liderLocal, remoteAddr)
						}
					}

					// Procesar información adicional enviada por el líder vía broadcast
					if msg.Tipo == "HEARTBEAT" && msg.Contenido != "" && msg.Host == liderLocal {
						log.Printf("📢 [BROADCAST-LIDER] %s: %s", msg.Host, msg.Contenido)
					}
				}
			}(conn)
		}
	}()

	ticker := time.NewTicker(1 * time.Second)
	for range ticker.C {
		mu.Lock()
		// Verificamos si el líder ha expirado o si no tenemos ninguno
		since := time.Since(ultimoLatido)
		if !soyLider && (liderLocal == "" || since > config.ElectionTimeout) {
			if liderLocal != "" {
				log.Printf("⏳ [COORD] Tiempo de espera de líder %s agotado (%v). Iniciando elección...", liderLocal, since)
				liderLocal = ""
				select {
				case chanLider <- "":
				default:
				}
			}
			log.Println("🗳️ [COORD] Sin líder. Iniciando elección...")
			soyElMasBajo := true

			// Escaneo en paralelo para evitar bloqueos y detectar al líder más rápido
			results := make(chan int, miID)
			var wg sync.WaitGroup
			for i := 1; i < miID; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					candidato := fmt.Sprintf("hospital-%d.%s", id, dominio)
					conn, err := net.DialTimeout("tcp4", candidato+config.PuertoServicio, config.DefaultTimeout)
					if err == nil {
						remoteIP := conn.RemoteAddr().String()
						log.Printf("🔍 [COORD] Nodo detectado: %s en %s", candidato, remoteIP)
						conn.Close()
						results <- id
					}
				}(i)
			}

			go func() {
				wg.Wait()
				close(results)
			}()

			// Identificamos el ID más bajo de los nodos que respondieron
			minIDFound := miID
			for id := range results {
				if id < minIDFound {
					minIDFound = id
				}
			}

			// Si encontramos un nodo con ID menor, lo seguimos
			if minIDFound < miID {
				liderLocal = fmt.Sprintf("hospital-%d.%s", minIDFound, dominio)
				ultimoLatido = time.Now()
				soyElMasBajo = false
				log.Printf("✅ [COORD] Líder encontrado (ID menor): %s", liderLocal)
				select {
				case chanLider <- liderLocal:
				default:
				}
			}

			// Si somos el ID más bajo disponible, asumimos el liderazgo
			if soyElMasBajo {
				liderLocal = miHost
				if !soyLider {
					soyLider = true
					// El líder debe habilitar el servidor para recibir datos médicos
					go comunicacion.IniciarServidorMedico(miHost)
					select {
					case chanLider <- miHost:
					default:
					}
				}
				log.Printf("👑 [COORD] Yo soy el líder: %s", miHost)
			}
		}
		
		currentSoyLider := soyLider
		mu.Unlock()

		// Si soy líder, notifico a todos los nodos de mi existencia
		if currentSoyLider {
			NotificarPares(miID, dominio, miHost, "Sistema médico sincronizado y operativo")
		}
	}
}

// NotificarPares envía un HEARTBEAT a todos los posibles nodos de la red.
func NotificarPares(miID int, dominio, miHost, info string) {
	// Semáforo para limitar la concurrencia y evitar saturar el stack TCP de Tailscale
	sem := make(chan struct{}, 20)
	for i := 1; i <= config.MaxNodes; i++ {
		if i == miID {
			continue
		}
		peer := fmt.Sprintf("hospital-%d.%s", i, dominio)
		go func(p string) {
			sem <- struct{}{}
			defer func() { <-sem }()

			conn, err := net.DialTimeout("tcp4", p+config.PuertoCoordinacion, config.DefaultTimeout)
			if err != nil {
				return
			}
			defer conn.Close()
			
			msg := comunicacion.Mensaje{
				Tipo: "HEARTBEAT",
				ID:   miID,
				Host: miHost,
				Contenido: info,
			}
			json.NewEncoder(conn).Encode(msg)
		}(peer)
	}
}