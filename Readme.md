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
- Luego instala Tailscale dentro del contenedor: Dentro de la shell de PROMOX
```bash
bash -c "$(curl -fsSL https://raw.githubusercontent.com/community-scripts/ProxmoxVE/main/tools/addon/add-tailscale-lxc.sh)"
```
- Luego ejecuta reinicia el contenedor con el siguiente comando:
```bash
pct restart 110
```
- Luego ejecuta el siguiente comando para obtener el estado de Tailscale:
```bash 
pct exec 110 -- tailscale up --auth-key=********************** --hostname nodo-tailscale-110
```
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
