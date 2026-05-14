# Product

Productive K3S CLI is the user-facing entrypoint for the Productive K3S ecosystem.

It does not replace Productive K3S Core or Productive K3S Infra. Instead, it resolves the correct versioned bundle, prepares it locally, and delegates the requested operation to the public interface exposed by those repositories.

Use this section if you want to understand:

- what `pk3s` is responsible for;
- how it relates to Core and Infra;
- why the CLI uses remote bundles by default;
- when local sibling resolution is still useful.
