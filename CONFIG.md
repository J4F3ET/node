# Documentación de Configuración: `config.go`

El paquete `config` centraliza todos los parámetros operativos del nodo. Al mantener estas variables en un solo lugar, garantizamos que tanto el módulo de **Coordinación** como el de **Comunicación** utilicen los mismos criterios de red y tiempos de espera.

## Definición de Puertos

| Constante | Valor | Descripción |
|-----------|-------|-------------|
| `PuertoServicio` | `:5000` | Puerto TCP donde el líder recibe los paquetes JSON con datos médicos. |
| `PuertoCoordinacion` | `:5001` | Puerto TCP utilizado para el intercambio de latidos (heartbeats) y procesos de elección. |

## Límites de la Red

- **`MaxNodes` (199):** Define el rango máximo de búsqueda de nodos en la Tailnet (de `hospital-1` hasta `hospital-199`). Este valor es crítico para el bucle de escaneo durante las elecciones y la notificación de latidos.

## Tiempos de Red (Timeouts)

- **`DefaultTimeout` (2s):** Es el tiempo máximo que un nodo esperará para establecer una conexión TCP inicial. Un tiempo muy bajo puede ignorar nodos lentos, mientras que uno muy alto retrasa el proceso de elección.

## Algoritmo de Salud (Liderazgo)

Estos parámetros definen la agresividad y la estabilidad del algoritmo de elección:

1.  **`HeartbeatInterval` (2s):** 
    - Define cada cuánto tiempo el Líder envía un mensaje de "estoy vivo" a todos los demás nodos.
    - **Impacto:** Si se reduce, aumenta el tráfico de red pero detecta fallos más rápido.

2.  **`ElectionTimeout` (10s):** 
    - Es el tiempo de gracia que un seguidor concede al líder. Si pasan 10 segundos sin recibir un latido, el seguidor asume que el líder ha muerto e inicia una nueva elección.
    - **Regla de Oro:** Siempre debe ser significativamente mayor que el `HeartbeatInterval` (normalmente 3x o 5x) para evitar falsos positivos por congestión de red.

## Mejores Prácticas

> [!IMPORTANT]
> Todos los nodos de la red **deben compartir la misma configuración**. 
> - Si un nodo tiene un `ElectionTimeout` más corto que los demás, intentará robar el liderazgo constantemente.
> - Si los puertos difieren, los nodos quedarán aislados en la red.

Si realizas cambios en este archivo, asegúrate de recompilar y desplegar el binario en todos los contenedores LXC y estaciones de trabajo Windows.