# Módulo de Comunicación: `comunicacion.go`

Este módulo es el encargado de la transmisión y recepción de datos médicos dentro de la red. Gestiona tanto el cliente que envía la información como el servidor que la recibe cuando el nodo actúa como líder.

## Estructura Universal de Mensaje

El sistema utiliza una única estructura JSON para todas las interacciones de red, lo que facilita el parsing y la extensibilidad:

```go
type Mensaje struct {
    Tipo      string // "DATA" (registros médicos) o "HEARTBEAT" (control de coordinación)
    ID        int    // Identificador único del nodo emisor (1-199)
    Host      string // Nombre de dominio Tailscale (MagicDNS) del emisor
    Contenido string // Payload de información (datos simulados o mensajes de estado)
}
```

## Componentes Principales

### 1. Servicio de Comunicación (`ServicioComunicacion`)
Es el bucle principal que reside en el hilo de ejecución más externo del nodo. Su comportamiento cambia dinámicamente según el estado del liderazgo:
- **Estado STANDBY:** Si no hay un líder detectado por el módulo de coordinación, el servicio permanece en espera.
- **Estado SEGUIDOR:** Cada 5 segundos (configurables), genera un paquete de datos médicos y lo envía al líder actual.
- **Estado LÍDER:** Si el nodo es el líder, evita el tráfico de red y simplemente procesa los datos de forma local.

### 2. Servidor Médico (`IniciarServidorMedico`)
Este servidor solo se activa si el nodo gana una elección:
- **Puerto:** Escucha en el puerto `5000` (definido en `config.PuertoServicio`).
- **Protocolo:** Utiliza `tcp4` para asegurar compatibilidad entre Linux (LXC) y Windows.
- **Procesamiento:** Decodifica el JSON entrante, imprime el origen (Nombre e IP) en los logs y responde con un `ACK` para confirmar la recepción.

### 3. Cliente de Envío (`EnviarDatosMedicos`)
Encapsula la lógica de conexión saliente:
- Establece una conexión TCP con el líder usando un timeout de 3 segundos.
- Serializa la información médica en formato JSON.

## Flujo de Trabajo (Workflow)

1.  **Sincronización:** El módulo recibe el nombre del líder a través de un canal (`chanLider`).
2.  **Preparación:** Si el nodo es seguidor, construye el objeto `Mensaje`.
3.  **Transmisión:** Se intenta realizar el `net.Dial` hacia el hostname del líder (ej: `hospital-1.ts.net:5000`).
4.  **Confirmación:** El líder recibe, loguea y cierra la conexión.

## Diagnóstico de Logs

- `🔄 [COM] Estado: CON LÍDER`: El nodo ha reconocido a un líder y está listo para enviar datos.
- `👑 [COM] Soy el Líder`: El nodo ha dejado de enviar datos por red y está procesando localmente.
- `🚨 [COM] Líder inalcanzable`: Error al intentar conectar al puerto 5000 del líder (posible caída del nodo o bloqueo de firewall).

> [!IMPORTANT]
> El puerto `5000` debe estar permitido en las reglas de entrada (Inbound) del firewall para que el líder pueda recolectar los datos de los demás hospitales.