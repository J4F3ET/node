# Arquitectura del Sistema: Nodos de Monitoreo Médico

Este diagrama describe la arquitectura interna de un nodo y cómo interactúan múltiples instancias dentro de la **Tailnet**.

```mermaid
flowchart TB
    subgraph Tailscale ["🌐 Red Privada Tailscale (MagicDNS)"]
        
        subgraph Leader ["👑 Nodo Líder (ID Menor)"]
            L_Main["⚙️ node.go"]
            L_Config["📄 config.go"]
            L_Coord(("🔄 Coordinación\n:5001"))
            L_Comm(("📡 Comunicación\n:5000"))
            L_Server["🖥️ Servidor Médico"]
            
            L_Config -.-> L_Main
            L_Main --> L_Coord & L_Comm
            L_Coord -. "chanLider\n(Soy Yo)" .-> L_Comm
            L_Comm === L_Server
        end

        subgraph Follower ["🏥 Nodo Seguidor (ID Mayor)"]
            F_Main["⚙️ node.go"]
            F_Config["📄 config.go"]
            F_Coord(("🔄 Coordinación\n:5001"))
            F_Comm(("📡 Comunicación\n:5000"))
            F_Client["💻 Cliente Médico"]
            
            F_Config -.-> F_Main
            F_Main --> F_Coord & F_Comm
            F_Coord -. "chanLider\n(ID Líder)" .-> F_Comm
            F_Comm === F_Client
        end

        %% Conexiones de Red entre Nodos
        L_Coord == "1. Heartbeat Broadcast\n(TCP/5001)" ===> F_Coord
        F_Coord -. "3. Elección por Timeout\n(Bully)" .-> L_Coord
        F_Client == "2. Envío de Datos JSON\n(TCP/5000)" ===> L_Server

    end

    %% Estilos Personalizados
    classDef leader fill:#e6f4ea,stroke:#1e8e3e,stroke-width:2px,color:#000;
    classDef follower fill:#e8f0fe,stroke:#1a73e8,stroke-width:2px,color:#000;
    classDef tailscale fill:#f8f9fa,stroke:#80868b,stroke-width:2px,stroke-dasharray: 5 5,color:#000;
    classDef module fill:#ffffff,stroke:#cccccc,stroke-width:1px,color:#333333;

    class Leader leader;
    class Follower follower;
    class Tailscale tailscale;
    class L_Main,L_Config,F_Main,F_Config,L_Server,F_Client module;
```
## Descripción de Componentes

1.  **Capa de Orquestación (`node.go`):** Inicializa el nodo, valida que el hostname `hospital-ID` sea único en la red y lanza los servicios concurrentes.
2.  **Módulo de Coordinación:**
    - Implementa el algoritmo de Bully. 
    - El **Líder** notifica su presencia mediante latidos constantes al puerto `5001`.
    - El **Seguidor** vigila el `ElectionTimeout`. Si se agota, escanea a los nodos con ID menor.
3.  **Módulo de Comunicación:** 
    - Utiliza un canal (`chanLider`) para recibir actualizaciones de estado.
    - Si es seguidor, actúa como cliente enviando JSON al puerto `5000`.
    - Si es líder, activa el servidor de recepción de datos.