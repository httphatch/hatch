# Troubleshooting

## Run the Doctor

The first step for any issue is to run the diagnostic command:

```bash
hatch doctor
```

This checks config validity, DNS resolver, CA certificates, launchd plist, port availability, and stale projects. Each check reports pass (✓) or fail (✗) with a hint for how to fix it.

## Common Issues

### DNS Not Resolving

**Symptoms:** Browser shows "server not found" for `*.test` domains.

**Diagnosis:**
```bash
# Check if the resolver file exists
cat /etc/resolver/test

# Test DNS resolution
dig myapp.test @127.0.0.1 -p 5053
```

**Fixes:**
- Run `hatch init` to reinstall the resolver file
- Ensure the daemon is running: `hatch up`
- Flush the macOS DNS cache: `sudo dscacheutil -flushcache; sudo killall -HUP mDNSResponder`

### Certificate Warnings in Browser

**Symptoms:** Browser shows "Your connection is not private" or a certificate warning.

**Diagnosis:**
```bash
hatch doctor    # check "Root CA trusted" status
```

**Fixes:**
- Run `hatch trust` to re-add the root CA to Keychain
- If you recently ran `hatch clean` and re-initialized, old browser tabs may cache the previous cert — restart the browser
- In Firefox, you may need to import the CA manually in Settings → Privacy & Security → Certificates → View Certificates → Import (`~/.hatch/certs/rootCA.pem`)

### Port Conflicts

**Symptoms:** `hatch up` fails or `hatch doctor` reports ports unavailable.

**Diagnosis:**
```bash
# Check what's using port 80 or 443
sudo lsof -i :80
sudo lsof -i :443
```

**Fixes:**
- Stop the conflicting process (Apache, Nginx, another proxy)
- Or change Hatch's ports in `~/.hatch/config.yml`:
  ```yaml
  settings:
    http_port: 8080
    https_port: 8443
  ```

### Stale PID File

**Symptoms:** Daemon won't start, or `hatch status` shows unexpected state.

**Diagnosis:**
```bash
cat ~/.hatch/hatch.pid
ps -p $(cat ~/.hatch/hatch.pid)
```

**Fixes:**
- Run `hatch down` then `hatch up`
- If that doesn't work, remove the PID file manually:
  ```bash
  rm ~/.hatch/hatch.pid
  hatch up
  ```

### Daemon Not Starting on Boot

**Symptoms:** After a reboot, `*.test` domains don't resolve until you manually run `hatch up`.

**Diagnosis:**
```bash
hatch doctor    # check launchd plist status
```

**Fixes:**
- Ensure `auto_start: true` in your config
- Run `hatch up` to reinstall the launchd plist with the correct settings

### Service Showing as Unhealthy

**Symptoms:** Dashboard or `hatch status` shows a red health indicator.

**Fixes:**
- Make sure your local dev server is actually running on the configured port
- Check the proxy URL in your config matches your dev server's address
- View logs for details: `hatch logs -f`

### Config Changes Not Taking Effect

**Symptoms:** You edited `config.yml` but the proxy config didn't update.

**Fixes:**
- Hatch watches the config file and reloads automatically. If it's not working:
  ```bash
  hatch restart
  ```
- Run `hatch config validate` to check for syntax errors

### "Permission Denied" Errors

**Symptoms:** `hatch init` or `hatch up` fails with permission errors.

**Fixes:**
- The DNS resolver and CA trust operations require `sudo`. Hatch will prompt for your password.
- If you're running in a non-interactive shell, run `hatch init` in a terminal first.

## Getting Help

If none of the above resolves your issue:

1. Check logs: `hatch logs -n 200`
2. Run with verbose output: `hatch -v status`
3. [Open an issue](https://github.com/httphatch/hatch/issues) with the output of `hatch doctor` and relevant logs.
