# Módulo de Coordinación: `coordinacion.go`

Este módulo es el cerebro del sistema distribuido. Implementa un algoritmo de **Bully (Abusón) modificado** para asegurar que siempre haya un único líder activo en la red, priorizando siempre al nodo con el identificador (ID) más bajo.

## Mecanismos Principales

### 1. Detección de Fallos (Heartbeats)
El sistema no depende de una conexión persistente, sino de un flujo constante de mensajes UDP/TCP simulando un latido:
- Todos los nodos escuchan en el puerto `5001`.
- Si un nodo no recibe señales del líder durante el tiempo definido en `ElectionTimeout` (10s), asume que el líder ha caído e inicia una elección.

### 2. Algoritmo de Elección
Cuando se activa una elección, el nodo sigue estos pasos:
1. **Escaneo de Superiores:** Intenta conectar con todos los IDs menores al suyo (del 1 al ID-1).
2. **Concurrencia:** El escaneo se realiza en paralelo mediante goroutines para minimizar el tiempo de espera.
3. **Resolución:**
   - Si un nodo con ID menor responde, el nodo actual se convierte en seguidor (`🔭`).
   - Si ningún nodo con ID menor responde, el nodo actual se proclama líder (`👑`).

### 3. Notificación de Pares (Broadcast)
Una vez que un nodo se convierte en líder, comienza a ejecutar la función `NotificarPares`:
- Envía un mensaje de tipo `HEARTBEAT` a todos los nodos posibles en la red (1-199).
- Utiliza un **semáforo de control** (limitado a 20 conexiones simultáneas) para evitar saturar la interfaz de red de Tailscale o el sistema operativo.

## Significado de los Logs (Iconografía)

Para facilitar el monitoreo visual en la consola, el módulo utiliza los siguientes iconos:

| Icono | Evento | Descripción |
|:---:|:---|:---|
| 📉 | **Abdicación** | El nodo detecta a alguien con más prioridad (ID menor) y cede el mando. |
| 🔭 | **Seguimiento** | El nodo identifica un nuevo líder y comienza a monitorizarlo. |
| ⏳ | **Timeout** | El líder actual no responde y el tiempo de espera se ha agotado. |
| 🗳️ | **Elección** | Se inicia el proceso de votación/escaneo en la red. |
| 🔍 | **Detección** | Se encontró un nodo activo durante el escaneo. |
| ✅ | **Resolución** | Se ha confirmado quién es el nuevo líder. |
| 👑 | **Liderazgo** | El nodo local toma el control del sistema. |

## Consideraciones de Seguridad

- **Supresión de Duplicados:** Si por un error de red dos nodos creen ser líderes, el que tenga el ID más alto abdicará inmediatamente al recibir un latido del que tenga el ID más bajo.
- **Red de Tailscale:** El módulo asume que `MagicDNS` está activo para resolver nombres como `hospital-1.dominio.ts.net`.

> [!TIP]
> El puerto `5001` debe estar abierto en el Firewall de Windows y Linux para que la coordinación funcione. Si este puerto está bloqueado, el nodo entrará en un bucle infinito de elecciones al no poder "ver" a sus pares.