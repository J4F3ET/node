# Sistema de Monitoreo Médico Distribuido

Este proyecto implementa un sistema distribuido en Go para la coordinación y envío de datos médicos en tiempo real. Utiliza un algoritmo de elección de líder basado en prioridades (ID más bajo) y comunicación TCP sobre una red privada Tailscale.

## Requisitos previos
- Proxmox VE
- Cuenta en Tailscale

## Configuración del Nodo (LXC)
- Primero crea LXC con el siguiente comando:
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
> **NOTE**
> - La app se ejecutará en el puerto 5000
> - De acuerdo al numero y dominio de la tailnet el nodo se encontraria como  `hospital-<NUMERO_DE_NODO>.<DOMINIO-TAILNET>:5000`