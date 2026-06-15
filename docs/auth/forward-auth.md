# Forward Authentication

updu supports delegating authentication to a trusted reverse proxy (like Authelia, Authentik, Pomerium, or Cloudflare Access) using Forward Authentication.

When forward auth is enabled and a request comes from a trusted proxy IP, updu will automatically extract the user's identity from HTTP headers and create a session for them. If the user does not exist in updu's database, they will be provisioned automatically.

## Configuration

Forward auth is configured via environment variables or in `updu.conf`:

```env
# Enable forward authentication
UPDU_FORWARD_AUTH_ENABLED=true

# (REQUIRED) You MUST configure trusted proxies.
# Comma-separated CIDRs whose headers are trusted.
UPDU_TRUSTED_PROXY_CIDRS=10.0.0.0/8,192.168.0.0/16,127.0.0.1/32

# (Optional) Header mapping
# Default: Remote-User
UPDU_FORWARD_AUTH_USER_HEADER=Remote-User

# Default: Remote-Groups
UPDU_FORWARD_AUTH_GROUP_HEADER=Remote-Groups

# Default: Remote-Email
UPDU_FORWARD_AUTH_EMAIL_HEADER=Remote-Email

# (Optional) The group name that grants admin access in updu.
# Default: updu-admins
UPDU_FORWARD_AUTH_ADMIN_GROUP=updu-admins
```

### Important Security Note

**You must configure `UPDU_TRUSTED_PROXY_CIDRS`**.

If you enable forward authentication but fail to restrict which IP addresses are trusted, anyone could spoof the `Remote-User` header and gain admin access to your updu instance. updu will ignore forward-auth headers if they come from an IP address not listed in `UPDU_TRUSTED_PROXY_CIDRS`.

## Behavior

- **Automatic Provisioning**: If a user logs in via the proxy and does not exist in updu, an account will be created for them automatically.
- **Roles**: If the user's groups (from `UPDU_FORWARD_AUTH_GROUP_HEADER`) include the `UPDU_FORWARD_AUTH_ADMIN_GROUP`, they will be granted the `admin` role. Otherwise, they receive the `viewer` role.
- **Fallbacks**: If forward authentication is enabled but the request does not come from a trusted proxy or lacks the required headers, updu will fall back to local session/bearer token authentication.
- **Password Changes**: Users authenticated via forward auth cannot change their password within updu's UI, as their credentials are managed by the external proxy.
