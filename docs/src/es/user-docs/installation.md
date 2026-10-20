# Instalación

Instalación en Unix-like:

```bash
curl -fsSL https://raw.githubusercontent.com/productive-k3s/productive-k3s-cli/main/install.sh | bash
```

El instalador deja el ejecutable `pk3s` en el `PATH` del usuario. Los bundles de Core e Infra se resuelven después, cuando se ejecuta un comando.
El script usa la version de release explicita del repositorio, descarga por HTTPS el
archivo correspondiente al sistema operativo y arquitectura junto con `checksums.txt`,
y exige una unica entrada SHA256 valida. Verifica el digest antes de extraer y reemplaza
el binario instalado de forma atomica solo cuando todos los controles pasan.
