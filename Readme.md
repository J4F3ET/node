# Sistema de Monitoreo Médico Distribuido

Este proyecto implementa un sistema distribuido en Go para la coordinación y envío de datos médicos en tiempo real. Utiliza un algoritmo de elección de líder basado en prioridades (ID más bajo) y comunicación TCP sobre una red privada Tailscale.

## 🚀 Características Principales
- **Alta Disponibilidad:** Elección automática de líder si el actual falla.
- **Prioridad por ID:** El nodo con el ID más bajo siempre es el líder preferido (Algoritmo de Bully modificado).
- **Seguridad de Red:** Túneles cifrados punto a punto mediante **Tailscale**.
- **Resolución Dinámica:** Uso de **MagicDNS** para evitar el hardcoding de direcciones IP.
- **Multiplataforma:** Compatible con Linux (contenedores LXC) y Windows.

## 🏗️ Estructura del Proyecto
- `/comunicacion`: Gestión de envío y recepción de JSON médicos (Puerto 5000).
- `/coordinacion`: Lógica de elección, latidos y estado del cluster (Puerto 5001).
- `/config`: Constantes globales, tiempos de espera y límites.
- `node.go`: Punto de entrada principal y validación de identidad.

## Requisitos previos
- Proxmox VE
- Cuenta en Tailscale
- Go 1.20 o superior

## 🛠️ Ejecución
Para iniciar un nodo, utiliza el siguiente comando:
```bash
go run node.go <ID_DEL_NODO> <DOMINIO_TAILNET>
```
Ejemplo: `go run node.go 5 tail5afc32.ts.net`

## Configuración del Nodo (LXC)
- Crea el contenedor LXC:
```bash
pct create 110 local:vztmpl/debian-12-standard_12.7-1_amd64.tar.zst \
  -hostname nodo-tailscale \
  -cores 1 \
  -memory 512 \
  -net0 name=eth0,bridge=vmbr0,ip=dhcp \
  -unprivileged 1 \
  -features nesting=1 \
  -storage local-lvm
pct start 110
```
o si quieres 
```bash 
bash -c "$(curl -fsSL https://raw.githubusercontent.com/community-scripts/ProxmoxVE/main/ct/debian.sh)"
```
> **NOTE**
> - En el comando anterior debes reemplazar `110` por el ID que deseas asignar a tu contenedor
- Luego actualiza el contenedor con el siguiente comando:
```bash
pct exec 110 -- apt update && apt upgrade -y
```
- Luego instala Tailscale dentro del contenedor: Dentro de la shell de PROMOX
```bash
bash -c "$(curl -fsSL https://raw.githubusercontent.com/community-scripts/ProxmoxVE/main/tools/addon/add-tailscale-lxc.sh)"
```
- Luego ejecuta reinicia el contenedor con el siguiente comando:
```bash
pct reboot 110
```
- Luego ejecuta el siguiente comando para levantar el nodo dentro de la tailnet de Tailscale:
```bash 
pct exec 110 -- tailscale up --auth-key=********************** --hostname hospital-<NUMERO_DE_NODO>
```
> **NOTE**
> - En el comando anterior debes reemplazar `**********************` por tu auth key de Tailscale, y `nodo-tailscale-110` por el hostname que deseas asignar a tu nodo dentro de la tailnet. Los numeros de nodos van desde el 1 al 199
- Luego instala git dentro del contenedor:
```bash
pct exec 110 -- apt update && apt install git -y
```
- Luego clona el repositorio del nodo:
```bash
pct exec 110 -- git clone  https://github.com/J4F3ET/node
```
- Luego instala go
```bash
pct exec 110 -- apt install golang-go -y 
```
- Luego ejecuta el siguiente comando para compilar el nodo:
```bash
pct exec 110 -- bash -c "go build ~/node/node.go <NUMERO_DE_NODO> <DOMINIO-TAILNET> "
pct exec 110 -- bash -c "go build ~/node/node.go 1 tail5afc32.ts.net "
```

## Solución de Problemas de Conectividad (Windows -> LXC)
Si no puedes hacer ping a un nodo específico desde Windows:

1. **Prueba de IP Directa:** Intenta `ping 100.x.x.x` (IP de Tailscale) en lugar de usar el hostname. Si funciona, es un problema de MagicDNS.
2. **Verificar TUN en Proxmox:** Asegúrate de que el archivo `/etc/pve/lxc/<ID>.conf` contenga:
   ```text
   lxc.cgroup2.devices.allow: c 10:200 rwm
   lxc.mount.entry: /dev/net/tun dev/net/tun none bind,create=file
   ```
3. **Estado del Túnel:** Ejecuta `tailscale status` en Windows y en el LXC. Ambos deben verse como "active" o "idle".
4. **Re-autenticación:** Si el nodo aparece pero no hay tráfico, intenta:
   `tailscale up --reset --hostname hospital-<NODO>` dentro del LXC.
5. **Rutas de Windows:** En CMD (Admin) de Windows, verifica que la ruta esté presente: `route print -4` (debe aparecer el rango `100.64.0.0/10` apuntando a la interfaz de Tailscale).
6. **Firewall Interno (LXC):** Asegúrate de que `ufw` o `iptables` permitan los puertos:
   ```bash
   # En el contenedor LXC si usas UFW
   ufw allow 5000/tcp
   ufw allow 5001/tcp
   ```
   En **Windows**, crea una regla de entrada para los puertos TCP 5000 y 5001.

## Arquitectura
1. **Coordinación:** Puerto 5001 (Heartbeats JSON).
2. **Servicio:** Puerto 5000 (Datos Médicos JSON).
> **NOTE**
> - La app se ejecutará en el puerto 5000
> - De acuerdo al numero y dominio de la tailnet el nodo se encontraria como  `hospital-<NUMERO_DE_NODO>.<DOMINIO-TAILNET>:5000`