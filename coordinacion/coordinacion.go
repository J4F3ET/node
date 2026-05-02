package coordinacion

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"node/comunicacion"
	"node/config"
	"time"
)

func ServicioCoordinacion(miID int, dominio, miHost string, chanLider chan string) {
	liderLocal := ""
	checkInterval := 5 * time.Second

	// Escuchar anuncios de otros líderes
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
				scanner := bufio.NewScanner(c)
				if scanner.Scan() {
					nuevoLider := scanner.Text()
					if nuevoLider != liderLocal {
						liderLocal = nuevoLider
						chanLider <- nuevoLider
						log.Printf("[COORD] Nuevo líder detectado vía red: %s", nuevoLider)
					}
				}
			}(conn)
		}
	}()

	ticker := time.NewTicker(checkInterval)
	for range ticker.C {
		if liderLocal == "" {
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
				go comunicacion.IniciarServidorMedico(miHost)
				NotificarPares(miID, dominio, miHost)
				chanLider <- miHost
				log.Printf("[COORD] Yo soy el líder: %s", miHost)
			}
		} else if liderLocal != miHost {
			// Verificar salud del líder actual
			conn, err := net.DialTimeout("tcp", liderLocal+config.PuertoServicio, 1*time.Second)
			if err != nil {
				log.Printf("[COORD] Líder %s caído. Reiniciando elección.", liderLocal)
				liderLocal = ""
				chanLider <- ""
			} else {
				conn.Close()
			}
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
			conn, err := net.DialTimeout("tcp", p+config.PuertoCoordinacion, 200*time.Millisecond)
			if err != nil {
				return
			}
			defer conn.Close()
			fmt.Fprintf(conn, "%s\n", miHost)
		}(peer)
	}
}