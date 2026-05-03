# Documentación de Configuración: `config.go`

El paquete `config` centraliza todos los parámetros operativos del nodo. Al mantener estas variables en un solo lugar, garantizamos que tanto el módulo de **Coordinación** como el de **Comunicación** utilicen los mismos criterios de red y tiempos de espera.

## Parámetros de Red y Puertos

| Constante | Valor | Descripción |
|-----------|-------|-------------|
| `PuertoServicio` | `:5000` | Puerto para envío de datos médicos y recepción de mensajes para **Relay**. |
| `PuertoCoordinacion` | `:5001` | Puerto TCP utilizado para el intercambio de latidos (heartbeats) y procesos de elección. |
| `NodePrefix` | `"hospital"` | Prefijo utilizado para identificar los nodos en la Tailnet. |
| `NodeHostnameFormat` | `"%s-%d.%s"` | Patrón para construir el FQDN (ej: `hospital-1.tailnet-domain.ts.net`). |

## Límites de la Red

- **`MaxNodes` (199):** Define el límite superior para:
    1. **Descubrimiento:** Escaneo de nodos superiores durante una elección.
    2. **Latidos:** Rango de difusión del pulso de vida del líder.
    3. **Relay:** Alcance de la retransmisión de mensajes manuales a toda la red.

## Tiempos de Red (Timeouts)

- **`DefaultTimeout` (2s):** Tiempo de espera para `net.Dial`. Crucial para que el escaneo de 199 nodos no bloquee el sistema si muchos están offline.
- **Retransmisión:** Las operaciones de difusión utilizan goroutines independientes para no verse afectadas por este timeout.

## Algoritmo de Salud (Liderazgo)

Estos parámetros definen la agresividad y la estabilidad del algoritmo de elección:

1.  **`HeartbeatInterval` (2s):** 
    - Define cada cuánto tiempo el Líder envía un mensaje de "estoy vivo" a todos los demás nodos.
    - **Impacto:** Frecuencia con la que se sincroniza el estado `CON LÍDER` en los seguidores.

2.  **`ElectionTimeout` (10s):** 
    - Es el tiempo de gracia que un seguidor concede al líder. Si pasan 10 segundos sin recibir un latido, el seguidor asume que el líder ha muerto e inicia una nueva elección.
    - **Nota:** Debe ser al menos 5 veces el `HeartbeatInterval` para tolerar latencia en la red virtual de Tailscale.

## Mejores Prácticas

> [!IMPORTANT]
> Todos los nodos de la red **deben compartir la misma configuración**. 
> - Si un nodo tiene un `ElectionTimeout` más corto que los demás, intentará robar el liderazgo constantemente.
> - Si los puertos difieren, los nodos quedarán aislados en la red.

Si realizas cambios en este archivo, asegúrate de recompilar y desplegar el binario en todos los contenedores LXC y estaciones de trabajo Windows.