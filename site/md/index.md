# updu Documentation

Welcome to the documentation for **updu**! Here you will find detailed guides on how to configure and use the various monitor types available in updu.

## Overview

updu is a lightweight, self-hosted uptime monitoring solution designed specifically for homelabs and small infrastructure. It’s built to run essentially anywhere (even on a Raspberry Pi Zero W) while providing all the essential monitoring features you need without the bloat.

## Available Monitor Types

Select a monitor type below to learn more about its configuration options:

- **[HTTP / HTTPS](/docs/http/index.html)** - Monitor web endpoints, status codes, and response bodies.
- **[TCP Port](/docs/tcp/index.html)** - Verify services are accepting connections on specific ports.
- **[DNS](/docs/dns/index.html)** - Validate DNS record resolution.
- **[ICMP / Ping](/docs/icmp/index.html)** - Check low-level host reachability.
- **[SSH](/docs/ssh/index.html)** - Verify SSH connectivity to remote machines.
- **[SSL Certificate](/docs/ssl/index.html)** - Track certificate expiry dates.
- **[JSON API](/docs/api/index.html)** - Deep-check API responses by validating JSON fields.

## Configuration

All monitor configuration in updu is done directly through the embedded web dashboard. When you add a new monitor, you'll be presented with specific fields relevant to that monitor type. The guides in this documentation explain each of these fields in detail.
