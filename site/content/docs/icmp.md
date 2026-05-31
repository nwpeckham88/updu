# ICMP / Ping Monitor

The ICMP monitor (often referred to as a "ping" check) verifies the basic reachability of a device over an IP network.

## Configuration Options

When setting up an ICMP monitor, you can configure the following options:

### Basic Settings

- **Name:** A descriptive name for your monitor.
- **Group:** Optional group assignment for organizing monitors.
- **Interval (seconds):** How frequently updu should send the ICMP echo requests.
- **Timeout (seconds):** The maximum time updu will wait for a reply before considering the ping failed.

### ICMP Specific Settings

- **Host / IP:** The hostname or IP address of the target machine (e.g., `8.8.8.8` or `router.local`).
- **Packet Count:** The number of ICMP echo requests to send during each check interval. (Default is usually 1, but sending multiple can help mitigate transient packet loss).
- **Packet Size (bytes):** The size of the payload attached to the ICMP packet. Useful for testing MTU issues or specific network constraints.

> **Note:** ICMP checks require that the device running updu has the necessary permissions to open raw sockets (or uses `setcap cap_net_raw+ep`). The official Docker image handles this automatically.

## Example Use Cases

- **Host Availability:** Ensure your home server or Raspberry Pi is alive on the network, even if its web services have crashed.
- **Gateway Check:** Ping your ISP router to verify that your basic internet connection is online.
