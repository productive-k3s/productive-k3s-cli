# Relación Con Productive K3S Core E Infra

Productive K3S CLI se ubica por encima de los otros dos repositorios.

- Productive K3S Core es dueño de la instalación y validación de la stack Kubernetes a nivel host.
- Productive K3S Infra es dueño del aprovisionamiento y la orquestación de escenarios basados en profiles.
- Productive K3S CLI es dueño del discovery, la resolución de bundles, el mapeo de comandos y la ergonomía orientada a usuarios.

El CLI no debería duplicar lógica de negocio de Core o Infra. Su trabajo es exponer una interfaz superior más limpia y derivar la ejecución hacia esos contratos públicos.
