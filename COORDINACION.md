# Módulo de Coordinación: `coordinacion.go`

Este módulo es el núcleo inteligente del sistema distribuido. Implementa un algoritmo de **Bully (Abusón) modificado** mediante una arquitectura orientada a estados para asegurar la alta disponibilidad, priorizando siempre al nodo con el identificador (ID) más bajo.

## Arquitectura Interna

A diferencia de versiones anteriores, el módulo ahora utiliza una estructura privada `coordinador` que encapsula el estado del nodo:

- **Estado de Salud:** Rastrea el `ultimoLatido` y el nombre del `liderLocal`.
- **Concurrencia:** Utiliza `sync.Mutex` para proteger el acceso al estado durante las elecciones y la recepción de mensajes.
- **Comunicación Interna:** Sincroniza el estado con el módulo de comunicación mediante un canal no bloqueante (`chanLider`) que garantiza siempre la entrega del líder más reciente (Last-Write-Wins).

## Mecanismos de Liderazgo

### 1. Monitoreo de Salud (`revisarSaludLider`)
Un temporizador (Ticker) ejecuta esta función cada segundo para:
- Verificar si el tiempo transcurrido desde el último latido supera el `ElectionTimeout`.
- Disparar una nueva elección si el líder actual se considera "muerto" o no existe.
- Si el nodo local es el líder, dispara la difusión de latidos (`notificarPares`).

### 2. Algoritmo de Elección (`iniciarEleccion`)
Cuando se inicia una elección, el nodo realiza un escaneo paralelo:
1. **Escaneo de Prioridad:** Intenta conectar con los puertos de servicio de todos los IDs inferiores (de `1` a `ID-1`) usando el formato dinámico `config.NodeHostnameFormat`.
2. **Resolución de Conflictos:** 
   - Si detecta un nodo activo con ID menor, lo establece como su nuevo líder (`✅`).
   - Si nadie con mayor prioridad responde, el nodo ejecuta `proclamarseLider()`, activa su servidor médico y toma el control (`👑`).

### 3. Gestión de Mensajes (`manejarConexion`)
El servidor de coordinación escucha en el puerto `5001` (`tcp4`) y procesa:
- **Supresión de Líderes:** Si el nodo es líder pero recibe un mensaje de un ID menor, abdica inmediatamente (`📉`) para mantener la jerarquía.
- **Sincronización de Estado:** Al detectar un cambio de líder, actualiza `liderLocal` de forma atómica antes de notificar al canal, evitando bucles infinitos de re-seguimiento.
- **Logs de Latidos:** Muestra información en tiempo real de los mensajes `HEARTBEAT` recibidos del líder actual.

### 4. Difusión de Latidos (`notificarPares`)
Este mecanismo resuelve la incertidumbre sobre qué nodos están activos en la red:
- **Escaneo de Rango:** El líder intenta conectar con todos los posibles hosts de la red (basado en `config.MaxNodes`, por defecto 1-199).
- **Conexión Efímera:** Se utiliza un `DialTimeout` corto (2 segundos). Si un nodo no está activo, el error se ignora silenciosamente y se continúa con el siguiente.
- **Optimización de Recursos:** Implementa un **semáforo de concurrencia** (límite de 20 goroutines) para evitar saturar el stack TCP de la interfaz virtual (Tailscale) durante la ráfaga de latidos.

### 5. Actualización del Canal (`actualizarLider`)
Para evitar bloqueos por lentitud en el módulo de comunicación, esta función:
1. Vacía cualquier valor antiguo presente en el canal `chanLider`.
2. Inserta el nuevo host de forma no bloqueante.

## Significado de los Logs (Iconografía)

El sistema utiliza iconos distintivos para permitir un diagnóstico rápido de la red:

| Icono | Evento | Descripción |
|:---:|:---|:---|
| 📉 | **Abdicación** | El nodo detecta a alguien con más prioridad (ID menor) y cede el mando. |
| 🔭 | **Seguimiento** | El nodo identifica un nuevo líder y comienza a monitorizarlo. |
| ⏳ | **Timeout** | El líder actual no responde y el tiempo de espera se ha agotado. |
| 🗳️ | **Elección** | Se inicia el proceso de votación/escaneo en la red. |
| 🔍 | **Detección** | Se encontró un nodo activo durante el escaneo. |
| ✅ | **Resolución** | Se ha confirmado quién es el nuevo líder. |
| 📢 | **Broadcast** | Información adicional recibida del líder a través de la red. |
| 👑 | **Liderazgo** | El nodo local toma el control del sistema. |

## Configuración e Integración

- **Prefijo Dinámico:** Utiliza `config.NodePrefix` para construir los nombres de host.
- **Formato de Hostname:** La resolución depende de `config.NodeHostnameFormat` (ej. `hospital-%d.%s`), asegurando que el escaneo sea compatible con la convención de nombres de Tailscale.
- **Tolerancia a Fallos:** El uso de un semáforo de concurrencia (límite de 20) en `notificarPares` previene el agotamiento de recursos en el stack TCP de la red virtual.

## Consideraciones Técnicas

- **Red de Tailscale:** Es imperativo que `MagicDNS` esté habilitado en la Tailnet.
- **Dual Stack:** Se fuerza el uso de `tcp4` para evitar inconsistencias de resolución de nombres entre nodos Linux y Windows.

> [!TIP]
> El puerto `5001` debe estar abierto en los firewalls de entrada. Si un nodo no puede recibir tráfico en este puerto, se aislará y creerá ser el líder perpetuamente, causando conflictos en la red.