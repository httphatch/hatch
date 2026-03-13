---
title: "Troubleshooting"
description: "Diagnose and fix common Hatch issues: DNS, certificates, ports, daemon, and config."
category: "advanced"
order: 2
lastUpdated: 2025-03-05
---

# Troubleshooting

## Run the doctor

The first step for any issue:

```bash
hatch doctor
```

This checks config validity, DNS resolver, CA certificates, launchd plist, port availability, and stale projects. Each check reports pass or fail with a hint for how to fix it.

## Common issues

### DNS not resolving

**Symptoms:** Browser shows "server not found" for `*.test` domains.

**Diagnosis:**
```bash
cat /etc/resolver/test
dig myapp.test @127.0.0.1 -p 5053
```

**Fixes:**
- Run `hatch init` to reinstall the resolver file
- Ensure the daemon is running: `hatch up`
- Flush the macOS DNS cache: `sudo dscacheutil -flushcache; sudo killall -HUP mDNSResponder`

### Certificate warnings in browser

**Symptoms:** Browser shows "Your connection is not private" or a certificate warning.

**Fixes:**
- Run `hatch trust` to re-add the root CA to Keychain
- If you recently ran `hatch clean` and re-initialized, restart the browser to clear cached certs
- In Firefox, import the CA manually: Settings > Privacy & Security > Certificates > View Certificates > Import (`~/.hatch/certs/rootCA.pem`)

See [HTTPS and Certificates](/docs/concepts/https-and-certificates) for how the certificate chain works.

### Port conflicts

**Symptoms:** `hatch up` fails with a port-in-use message, or `hatch status` shows a service as unhealthy with a port conflict.

`hatch status` automatically checks ports when a service is unhealthy and shows the process name, PID, and suggested `kill` command.

**Fixes:**
- Kill the process shown in the conflict summary, then run `hatch up` again
- Or change Hatch's ports in `~/.hatch/config.yml`:
  ```yaml
  settings:
    http_port: 8080
    https_port: 8443
  ```

### Daemon failed to start

**Symptoms:** `hatch up` shows progress lines then prints `daemon process exited` or `timeout waiting for daemon after 120s`.

During startup, `hatch up` shows each subsystem as it initializes (DNS server, Caddy, tunnel domains, etc.). If the daemon process crashes, you will see `daemon process exited; check logs with: hatch logs`. If it stalls, you will see periodic `Waiting for daemon...` messages until the 120-second timeout.

**Fixes:**
- Run `hatch logs` to see what went wrong
- Run `hatch doctor` to check for port conflicts or config errors
- If logs show a port conflict, stop the conflicting process and run `hatch up` again

### Stale PID file

**Symptoms:** Daemon won't start, or `hatch status` shows unexpected state.

**Fixes:**
- Run `hatch down` then `hatch up`
- If that doesn't work, remove the PID file manually:
  ```bash
  rm ~/.hatch/hatch.pid
  hatch up
  ```

### Daemon not starting on boot

**Symptoms:** After a reboot, `*.test` domains don't resolve until you manually run `hatch up`.

**Fixes:**
- Ensure `auto_start: true` in your config
- Run `hatch up` to reinstall the launchd plist with the correct settings

### Service showing as unhealthy

**Symptoms:** Dashboard or `hatch status` shows a red health indicator.

**Fixes:**
- If the hint says "not listening", your dev server is not running. Start it on the configured port.
- If the hint names a process and PID, that process is holding the port. Kill it or reconfigure.
- Check the proxy URL in your config matches your dev server's address
- View logs for details: `hatch logs -f`
- See [Debug a Crashing Process](/docs/guides/debug-crashing-process) if the service keeps restarting.

### Config changes not taking effect

**Fixes:**
- Hatch watches the config file and reloads automatically. If it's not working:
  ```bash
  hatch restart
  ```
- Run `hatch config validate` to check for syntax errors

### "Permission denied" errors

**Fixes:**
- The DNS resolver and CA trust operations require `sudo`. Hatch will prompt for your password.
- If running in a non-interactive shell, run `hatch init` in a terminal first.

## Getting help

If none of the above resolves your issue:

1. Check logs: `hatch logs -n 200`
2. Run with verbose output: `hatch -v status`
3. [Open an issue](https://github.com/httphatch/hatch/issues) with the output of `hatch doctor` and relevant logs.
