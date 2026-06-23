---
title: "Productive K3S CLI"
template: "home.html"
hide:
  - navigation
  - toc
eyebrow: "Unified command line for Productive K3S"
eyebrow_es: "Línea de comandos unificada para Productive K3S"
hero_title: "Productive K3S CLI"
hero_title_es: "Productive K3S CLI"
lead: "Productive K3S CLI provides one executable, `pk3s`, for working with Productive K3S Core and Productive K3S Infra through a consistent, release-aware command surface."
lead_es: "Productive K3S CLI ofrece un único ejecutable, `pk3s`, para trabajar con Productive K3S Core y Productive K3S Infra mediante una superficie de comandos consistente y consciente de releases."
sublead: "It resolves versioned bundles, validates their runtime contract, and delegates execution into the public Core and Infra entrypoints instead of reimplementing their logic."
sublead_es: "Resuelve bundles versionados, valida su contrato de runtime y delega la ejecución hacia los entrypoints públicos de Core e Infra en lugar de reimplementar su lógica."
primary_label: "View on GitHub"
primary_label_es: "Ver en GitHub"
primary_url: "https://github.com/productive-k3s/productive-k3s-cli"
secondary_label: "Open README"
secondary_label_es: "Abrir README"
secondary_url: "https://github.com/productive-k3s/productive-k3s-cli/blob/main/README.md"
card_title: "What it does"
card_title_es: "Qué hace"
card_items:
  - Resolves Productive K3S Core and Infra bundles from published releases
  - Supports explicit local sibling mode for development and debugging
  - Exposes one top-level help system and command surface for users
card_items_es:
  - Resuelve bundles de Productive K3S Core e Infra desde releases publicados
  - Soporta modo local explícito con repos hermanos para desarrollo y debugging
  - Expone un único sistema de ayuda y una única superficie de comandos para usuarios
why_title: "Why it exists"
why_title_es: "Por qué existe"
why_options:
  - label: "SPLIT ENTRYPOINTS"
    text: "Separate Core and Infra commands are accurate, but harder for users to discover and operate consistently."
  - label: "REWRITTEN PLATFORM"
    text: "Rewriting Core and Infra logic inside the CLI would create drift, duplication, and fragile maintenance."
why_options_es:
  - label: "ENTRYPOINTS SEPARADOS"
    text: "Los comandos separados de Core e Infra son correctos, pero más difíciles de descubrir y operar consistentemente."
  - label: "PLATAFORMA REESCRITA"
    text: "Reescribir la lógica de Core e Infra dentro del CLI generaría drift, duplicación y mantenimiento frágil."
bridge_note: "Productive K3S CLI is the middle path: one UX layer, explicit bundle resolution, and delegation into the real product contracts."
bridge_note_es: "Productive K3S CLI es el camino intermedio: una sola capa de UX, resolución explícita de bundles y delegación hacia los contratos reales del producto."
bridge_points:
  - Keep Core and Infra authoritative
  - Make published releases first-class
  - Give users one ergonomic entrypoint
bridge_points_es:
  - Mantener a Core e Infra como fuente autoritativa
  - Tratar los releases publicados como ciudadanos de primera clase
  - Dar a los usuarios un único entrypoint ergonómico
scenarios_title: "Typical usage"
scenarios_title_es: "Uso típico"
scenarios:
  - Run remote Multipass and on-prem scenarios from a workstation
  - Validate profiles before provisioning
  - Inspect the exact Core or Infra bundle selected for execution
  - Use the same CLI locally in development and remotely through published releases
scenarios_es:
  - Ejecutar escenarios remotos de Multipass y on-prem desde una workstation
  - Validar profiles antes de aprovisionar
  - Inspeccionar el bundle exacto de Core o Infra elegido para ejecutar
  - Usar el mismo CLI localmente en desarrollo y remotamente mediante releases publicados
principles_title: "Design principles"
principles_title_es: "Principios de diseño"
principles:
  - title: "Delegate, do not duplicate"
    text: "the CLI should route into Core and Infra, not absorb their implementation"
  - title: "Remote first"
    text: "published bundles are the default user path and must be validated as products"
  - title: "Explicit local mode"
    text: "development workflows can still target sibling repositories when requested"
principles_es:
  - title: "Delegar, no duplicar"
    text: "el CLI debe derivar hacia Core e Infra, no absorber su implementación"
  - title: "Remote first"
    text: "los bundles publicados son el camino por defecto para usuarios y deben validarse como producto"
  - title: "Modo local explícito"
    text: "los flujos de desarrollo pueden seguir apuntando a repos hermanos cuando se lo pide"
environments_title: "Where it fits"
environments_title_es: "Dónde encaja"
environments:
  - User laptops and operator workstations
  - CI pipelines validating published Productive K3S bundles
  - Development setups consuming local sibling repositories explicitly
  - Cross-platform CLI distribution for Linux, macOS, and Windows
environments_es:
  - Laptops de usuarios y workstations operativas
  - Pipelines de CI que validan bundles publicados de Productive K3S
  - Setups de desarrollo que consumen explícitamente repos hermanos locales
  - Distribución cross-platform del CLI para Linux, macOS y Windows
not_title: "What it is not"
not_title_es: "Qué no es"
not_items:
  - Not a replacement for Productive K3S Core
  - Not a replacement for Productive K3S Infra
  - Not an excuse to hide version resolution or bundle provenance
not_items_es:
  - No reemplaza a Productive K3S Core
  - No reemplaza a Productive K3S Infra
  - No es una excusa para ocultar la resolución de versiones o la procedencia de bundles
not_note: "It is the UX and release-aware orchestration layer above Core and Infra."
not_note_es: "Es la capa de UX y orquestación consciente de releases por encima de Core e Infra."
---
