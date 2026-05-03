# Documentación del Punto de Entrada: `node.go`

El archivo `node.go` actúa como el orquestador principal del sistema. Es el binario que se ejecuta en cada nodo (ya sea LXC o Windows) y se encarga de inicializar los servicios, validar la identidad del nodo y asegurar que no haya conflictos en la red.

## Responsabilidades Principales

1.  **Gestión de Argumentos:** Captura el `ID` del hospital y el `Dominio` de Tailscale desde la línea de comandos.
2.  **Validación de Identidad:** Asegura que el ID esté dentro del rango permitido (1-199).
3.  **Control de Unicidad:** Implementa una técnica de "autodescubrimiento" para evitar que dos nodos usen el mismo ID simultáneamente.
4.  **Orquestación de Procesos:** Inicia concurrentemente los motores de **Coordinación** (elección de líder) y **Comunicación** (envío de datos).

## Flujo de Ejecución

### 1. Inicialización y Parsing
El nodo requiere dos parámetros obligatorios:
```bash
go run node.go <ID> <DOMINIO>
```
El `miHost` se construye siguiendo el patrón de MagicDNS: `hospital-<ID>.<DOMINIO>`.

### 2. Verificación de Unicidad (Anti-Conflicto)
Antes de arrancar cualquier servicio, el nodo realiza una prueba crítica:
```go
conn, err := net.DialTimeout("tcp", miHost+config.PuertoServicio, 2*time.Second)
```
Si esta conexión tiene éxito, significa que **ya existe un nodo activo** con ese mismo nombre en la Tailnet. El proceso se detiene inmediatamente con un error fatal para evitar corromper la lógica de la red distribuida.

### 3. Comunicación Inter-módulos
Se crea un canal compartido llamado `chanLider`:
- **Emisor:** El módulo de `coordinacion` envía por aquí el nombre del host que ha sido detectado o elegido como líder.
- **Receptor:** El módulo de `comunicacion` escucha este canal para saber a quién debe enviar los datos médicos.

### 4. Ejecución Concurrente
`node.go` finaliza lanzando las dos piezas fundamentales del rompecabezas:

| Componente | Modo | Función |
|------------|------|---------|
| `coordinacion.ServicioCoordinacion` | Goroutine (`go`) | Mantiene la salud del cluster y decide quién es el líder. |
| `comunicacion.ServicioComunicacion` | Hilo Principal | Gestiona el bucle de envío de datos médicos al líder actual. |

## Diagnóstico Rápido
Si el nodo se cierra inmediatamente al iniciar, verifica:
1. Que el ID sea un número entre 1 y 199.
2. Que no haya otra instancia del mismo nodo corriendo en otro contenedor o PC con el mismo ID de hospital.