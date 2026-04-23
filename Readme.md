Como crear un nodo
- Primero crea LXC con el siguiente comando:
```bash
pct create 110 local:vztmpl/debian-12-standard_12.7-1_amd64.tar.zst \
  -hostname nodo-tailscale \
  -cores 1 \
  -memory 512 \
  -net0 name=eth0,bridge=vmbr0,ip=dhcp \
  -unprivileged 0 \
  -features nesting=1
```
o si quieres 
```bash 
bash -c "$(curl -fsSL https://raw.githubusercontent.com/community-scripts/ProxmoxVE/main/ct/debian.sh)"
```
> **NOTE**
> - En el comando anterior debes reemplazar `110` por el ID que deseas asignar a tu contenedor
- Luego instala Tailscale dentro del contenedor: Dentro de la shell de PROMOX
```bash
bash -c "$(curl -fsSL https://raw.githubusercontent.com/community-scripts/ProxmoxVE/main/tools/addon/add-tailscale-lxc.sh)"
```
- Luego ejecuta reinicia el contenedor con el siguiente comando:
```bash
pct restart 110
```
- Luego ejecuta el siguiente comando para levantar el nodo dentro de la tailnet de Tailscale:
```bash 
pct exec 110 -- tailscale up --auth-key=********************** --hostname hospital_<NUMERO_DE_NODO>
```
> **NOTE**
> - En el comando anterior debes reemplazar `**********************` por tu auth key de Tailscale, y `nodo-tailscale-110` por el hostname que deseas asignar a tu nodo dentro de la tailnet.


- Luego ejecuta el siguiente comando para obtener la IP de Tailscale:
```bash
pct exec 110 -- tailscale ip -4
```
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
> - De acuerdo al numero y dominio de la tailnet el nodo se encontraria como  `hospital_<NUMERO_DE_NODO>.<DOMINIO-TAILNET>:5000`

