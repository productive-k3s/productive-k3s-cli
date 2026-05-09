# Alcance y responsabilidades del CLI

Productive K3S CLI es una capa de orquestación orientada al usuario para el ecosistema Productive K3S.

El CLI no reemplaza los scripts públicos expuestos por Productive K3S Core o Productive K3S Infra. En cambio, los envuelve detrás de una experiencia de línea de comandos unificada.

## Responsabilidades

El CLI es responsable de:

- resolver versiones compatibles de los bundles Core e Infra;
- descargar o ubicar los bundles requeridos;
- validar la metadata de los bundles;
- exponer una estructura de comandos consistente;
- delegar la ejecución en los entrypoints de cada bundle;
- preservar comportamiento determinístico entre releases.

## Fuera de alcance

El CLI no debe duplicar la lógica de instalación, validación, OpenTofu, Ansible o escenarios específicos que ya se mantiene en los repositorios Core e Infra.
