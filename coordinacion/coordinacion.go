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

// coordinador mantiene el estado interno de la coordinación para reducir complejidad.
type coordinador struct {
	mu           sync.Mutex
	miID         int
	miHost       string
	dominio      string
	liderLocal   string
	ultimoLatido time.Time
	soyLider     bool
	chanLider    chan string
}

// ServicioCoordinacion orquesta el ciclo de vida del algoritmo de elección.
func ServicioCoordinacion(miID int, dominio, miHost string, chanLider chan string) {
	c := &coordinador{
		miID:         miID,
		miHost:       miHost,
		dominio:      dominio,
		chanLider:    chanLider,
		ultimoLatido: time.Now(),
	}

	// Iniciar servidor de escucha en segundo plano
	go c.servidorCoordinacion()

	// Bucle de monitoreo de salud
	ticker := time.NewTicker(1 * time.Second)
	for range ticker.C {
		c.revisarSaludLider()
	}
}

// servidorCoordinacion maneja el listener TCP para mensajes de control.
func (c *coordinador) servidorCoordinacion() {
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
		go c.manejarConexion(conn)
	}
}

// manejarConexion procesa mensajes individuales de otros nodos.
func (c *coordinador) manejarConexion(conn net.Conn) {
	defer conn.Close()
	remoteAddr := conn.RemoteAddr().String()
	var msg comunicacion.Mensaje
	if err := json.NewDecoder(conn).Decode(&msg); err != nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Caso 1: Supresión si aparece alguien con ID menor (mayor prioridad)
	if msg.ID < c.miID && c.soyLider {
		log.Printf("📉 [COORD] Detectado líder con mayor prioridad (%s - %s). Abdicando...", msg.Host, remoteAddr)
		c.soyLider = false
		c.actualizarLider(msg.Host)
	}

	// Caso 2: Latido válido de un líder con igual o mayor prioridad
	if msg.ID <= c.miID {
		c.ultimoLatido = time.Now()
		if c.liderLocal != msg.Host {
			c.actualizarLider(msg.Host)
			log.Printf("🔭 [COORD] Siguiendo a nuevo líder: %s (%s)", c.liderLocal, remoteAddr)
		}
	}

	// Logs de broadcast informativo
	if msg.Tipo == "HEARTBEAT" && msg.Contenido != "" && msg.Host == c.liderLocal {
		log.Printf("📢 [BROADCAST-LIDER] %s: %s", msg.Host, msg.Contenido)
	}
}

// revisarSaludLider verifica tiempos de espera y dispara elecciones o latidos.
func (c *coordinador) revisarSaludLider() {
	c.mu.Lock()
	defer c.mu.Unlock()

	since := time.Since(c.ultimoLatido)
	if !c.soyLider && (c.liderLocal == "" || since > config.ElectionTimeout) {
		if c.liderLocal != "" {
			log.Printf("⏳ [COORD] Tiempo de espera de líder %s agotado. Iniciando elección...", c.liderLocal)
		}
		c.iniciarEleccion()
	}

	if c.soyLider {
		c.notificarPares("Sistema médico sincronizado y operativo")
	}
}

// iniciarEleccion realiza un escaneo paralelo para encontrar líderes superiores (ID menor).
func (c *coordinador) iniciarEleccion() {
	log.Println("🗳️ [COORD] Iniciando proceso de elección...")
	c.liderLocal = ""
	c.actualizarLider("")

	results := make(chan int, c.miID)
	var wg sync.WaitGroup

	// Escaneamos solo IDs menores al nuestro
	for i := 1; i < c.miID; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			candidato := fmt.Sprintf(config.NodeHostnameFormat, config.NodePrefix, id, c.dominio)
			conn, err := net.DialTimeout("tcp4", candidato+config.PuertoServicio, config.DefaultTimeout)
			if err == nil {
				log.Printf("🔍 [COORD] Nodo superior detectado: %s", candidato)
				conn.Close()
				results <- id
			}
		}(i)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	minIDFound := c.miID
	for id := range results {
		if id < minIDFound {
			minIDFound = id
		}
	}

	if minIDFound < c.miID {
		// Encontramos a alguien mejor
		nuevoLider := fmt.Sprintf(config.NodeHostnameFormat, config.NodePrefix, minIDFound, c.dominio)
		c.liderLocal = nuevoLider
		c.ultimoLatido = time.Now()
		c.soyLider = false
		c.actualizarLider(nuevoLider)
		log.Printf("✅ [COORD] Líder encontrado (ID %d): %s", minIDFound, nuevoLider)
	} else {
		// Somos el mejor ID disponible
		c.proclamarseLider()
	}
}

// proclamarseLider realiza la transición a estado de liderazgo.
func (c *coordinador) proclamarseLider() {
	c.liderLocal = c.miHost
	if !c.soyLider {
		c.soyLider = true
		go comunicacion.IniciarServidorMedico(c.miHost)
		c.actualizarLider(c.miHost)
	}
	log.Printf("👑 [COORD] Yo soy el líder: %s", c.miHost)
}

// actualizarLider envía de forma no bloqueante el nuevo líder al canal.
func (c *coordinador) actualizarLider(host string) {
	select {
	case c.chanLider <- host:
	default:
		// Si el canal está lleno, ya hay una actualización pendiente
	}
}

// notificarPares envía un HEARTBEAT a todos los posibles nodos de la red.
func (c *coordinador) notificarPares(info string) {
	// Semáforo para limitar la concurrencia y evitar saturar el stack TCP de Tailscale
	sem := make(chan struct{}, 20)
	for i := 1; i <= config.MaxNodes; i++ {
		if i == c.miID {
			continue
		}
		peer := fmt.Sprintf(config.NodeHostnameFormat, config.NodePrefix, i, c.dominio)
		go func(p string) {
			sem <- struct{}{}
			defer func() { <-sem }()

			conn, err := net.DialTimeout("tcp4", p+config.PuertoCoordinacion, config.DefaultTimeout)
			if err != nil {
				return
			}
			defer conn.Close()
			
			msg := comunicacion.Mensaje{
				Tipo:      "HEARTBEAT",
				ID:        c.miID,
				Host:      c.miHost,
				Contenido: info,
			}
			json.NewEncoder(conn).Encode(msg)
		}(peer)
	}
}