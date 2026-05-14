# Relationship With Productive K3S Core And Infra

Productive K3S CLI sits above the other two repositories.

- Productive K3S Core owns host-level Kubernetes stack installation and validation.
- Productive K3S Infra owns profile-driven scenario provisioning and orchestration.
- Productive K3S CLI owns discovery, bundle resolution, command mapping, and user-facing ergonomics.

The CLI should not duplicate business logic from Core or Infra. Its job is to expose a cleaner top-level interface and route execution into those public contracts.
