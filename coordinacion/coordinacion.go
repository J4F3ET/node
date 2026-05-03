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

func ServicioCoordinacion(miID int, dominio, miHost string, chanLider chan string) {
	var mu sync.Mutex
	liderLocal := ""
	ultimoLatido := time.Now()
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
					mu.Lock()
					defer mu.Unlock()

					// Lógica de supresión de líderes duplicados
					if msg.ID < miID && soyLider {
						log.Printf("[COORD] Detectado líder con mayor prioridad (%s). Abdicando...", msg.Host)
						soyLider = false
						liderLocal = msg.Host
						select {
						case chanLider <- liderLocal:
						default:
						}
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
		mu.Lock()
		since := time.Since(ultimoLatido)
		if !soyLider && (liderLocal == "" || since > config.ElectionTimeout) {
			if liderLocal != "" {
				log.Printf("[COORD] Tiempo de espera de líder %s agotado (%v). Iniciando elección...", liderLocal, since)
				liderLocal = ""
				select {
				case chanLider <- "":
				default:
				}
			}
			log.Println("[COORD] Sin líder. Iniciando elección...")
			soyElMasBajo := true

			// Escaneo en paralelo para evitar bloqueos y detectar al líder más rápido
			results := make(chan int, miID)
			var wg sync.WaitGroup
			for i := 1; i < miID; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					candidato := fmt.Sprintf("hospital-%d.%s", id, dominio)
					conn, err := net.DialTimeout("tcp", candidato+config.PuertoServicio, config.DefaultTimeout)
					if err == nil {
						conn.Close()
						results <- id
					}
				}(i)
			}

			go func() {
				wg.Wait()
				close(results)
			}()

			minIDFound := miID
			for id := range results {
				if id < minIDFound {
					minIDFound = id
				}
			}

			if minIDFound < miID {
				liderLocal = fmt.Sprintf("hospital-%d.%s", minIDFound, dominio)
				ultimoLatido = time.Now()
				soyElMasBajo = false
				log.Printf("[COORD] Líder encontrado (ID menor): %s", liderLocal)
				select {
				case chanLider <- liderLocal:
				default:
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
		
		currentSoyLider := soyLider
		mu.Unlock()

		if currentSoyLider {
			NotificarPares(miID, dominio, miHost)
		}
	}
}

func NotificarPares(miID int, dominio, miHost string) {
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

			conn, err := net.DialTimeout("tcp", p+config.PuertoCoordinacion, config.DefaultTimeout)
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