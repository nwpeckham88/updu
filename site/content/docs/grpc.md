# gRPC Health Monitor

The gRPC monitor in updu calls the standard `grpc.health.v1.Health/Check` RPC against your service and considers the monitor Up only when the response status is `SERVING`. This is the same probe used by Kubernetes' gRPC liveness checks and Istio service health, so any server that already exposes the standard health protocol works without changes.

## Configuration Options

When setting up a gRPC monitor, you can configure the following options:

### Basic Settings

- **Name:** A descriptive name for your monitor.
- **Group:** Optional group assignment for organizing monitors.
- **Interval (seconds):** How frequently updu should perform the check.
- **Timeout (seconds):** Maximum time updu waits for the gRPC client to dial and complete the health RPC.

### gRPC Specific Settings

- **Hostname:** The IP address or domain name of the gRPC server (e.g., `payments.internal.example`).
- **Port:** The TCP port the gRPC server listens on (commonly `50051`).
- **Service:** (Optional) The fully-qualified service name passed in the `HealthCheckRequest` (e.g., `payments.v1.PaymentService`). Leave empty to query overall server health.
- **TLS:** Enable when the server requires a TLS transport. updu uses TLS 1.2 or higher.
- **Skip TLS verify:** (Optional) Disable certificate verification when TLS is enabled. Use only for self-signed development clusters.
- **Authority:** (Optional) Override the `:authority` pseudo-header. Useful when probing through a host-routing proxy or service mesh.

## Example Use Cases

- **Internal microservices:** Probe each backend's standard health service to catch upstream rollouts that miss readiness gates.
- **Service mesh sidecars:** Verify Envoy or Istio-fronted services are reachable end-to-end through the mesh by overriding the `:authority` header.
- **Multi-tenant gRPC servers:** Track per-service health (`Service` field) on a server that hosts several gRPC services in a single process.

## Notes

- The monitor reports the returned health status (e.g., `SERVING`, `NOT_SERVING`, `UNKNOWN`) in result metadata for richer dashboards and alerting.
- Servers that do not implement `grpc.health.v1.Health` will return `Unimplemented` and the check will report Down — implement the standard health service or use a TCP monitor instead.
