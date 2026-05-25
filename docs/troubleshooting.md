# Troubleshooting

Common issues and how to fix them.

## Quick Diagnostics

Run the doctor command first:

```bash
tube doctor
```

This checks all common issues and provides guidance.

---

## DNS Issues

### "myapp.test" doesn't resolve

**Symptoms:**
- Browser shows "Server not found" or "DNS_PROBE_FINISHED_NXDOMAIN"
- `ping myapp.test` fails with "cannot resolve"

**Check 1: Is DNS configured?**

```bash
tube dns-status
```

If not configured:
```bash
sudo tube setup
```

**Check 2: Is dnsmasq running?**

```bash
tube status
```

If dnsmasq is stopped:
```bash
tube start
```

**Check 3: Test DNS resolution directly**

```bash
dig @127.0.0.1 myapp.test
```

Expected output should show `127.0.0.1` in the ANSWER section.

If it fails:
```bash
# Check if something else is using port 53
sudo lsof -i :53

# Restart dnsmasq
tube restart
```

**Check 4: Flush DNS cache**

```bash
sudo dscacheutil -flushcache
sudo killall -HUP mDNSResponder
```

**Check 5: Verify resolver file**

```bash
cat /etc/resolver/test
```

Should contain:
```
nameserver 127.0.0.1
```

### DNS works in terminal but not browser

**Cause:** Browser may be using its own DNS cache or DoH (DNS over HTTPS).

**Fix for Chrome:**
1. Go to `chrome://net-internals/#dns`
2. Click "Clear host cache"

**Fix for Firefox:**
1. Go to `about:networking#dns`
2. Click "Clear DNS Cache"

**Disable DoH (if enabled):**
- Chrome: Settings → Privacy and Security → Security → Use secure DNS → Off
- Firefox: Settings → Network Settings → Enable DNS over HTTPS → Off

---

## Service Issues

### "nginx: command not found"

**Cause:** nginx is not installed or not in PATH.

**Fix:**
```bash
# Install nginx
brew install nginx

# Or specify full path in config
# ~/.tube/config.yaml
nginx:
  binary: /opt/homebrew/bin/nginx
```

### "dnsmasq: command not found"

**Cause:** dnsmasq is not installed or not in PATH.

**Fix:**
```bash
# Install dnsmasq
brew install dnsmasq

# Or specify full path in config
# ~/.tube/config.yaml
dnsmasq:
  binary: /opt/homebrew/sbin/dnsmasq
```

### "Address already in use" (port 80)

**Cause:** Another process is using port 80.

**Find the process:**
```bash
sudo lsof -i :80
```

**Common culprits:**
- Apache (`httpd`)
- Another nginx instance
- Docker containers

**Fix:**
```bash
# Stop Apache
sudo apachectl stop

# Or kill the process
sudo kill <PID>
```

### "Address already in use" (port 53)

**Cause:** Another DNS service is using port 53.

**Find the process:**
```bash
sudo lsof -i :53
```

**Common culprits:**
- Another dnsmasq instance
- systemd-resolved (Linux)
- mDNSResponder (usually fine)

**Fix:**
```bash
# Kill the conflicting process
sudo kill <PID>

# Or change dnsmasq port (advanced)
```

### Services won't start (permission denied)

**Cause:** Ports 80 and 53 require root privileges.

**Fix 1: Run with sudo**
```bash
sudo tube start
```

**Fix 2: Use higher ports (advanced)**

Edit `~/.tube/config.yaml`:
```yaml
nginx:
  http_port: 8080
dnsmasq:
  port: 5353
```

Note: This requires additional configuration for DNS resolution.

### nginx config test fails

**Check nginx configuration:**
```bash
nginx -t -c ~/.tube/nginx.conf
```

**Common errors:**

1. **"unknown directive"** - Incompatible nginx version
2. **"host not found"** - DNS resolution issue during config test
3. **"no such file"** - Missing log directory

**Fix missing directories:**
```bash
mkdir -p ~/.tube/logs ~/.tube/pids
```

---

## Project Issues

### Project shows as "stopped" but app is running

**Cause:** tube checks if the port is responding. Your app might:
- Be starting slowly
- Not be binding to the expected port
- Be binding to a different interface

**Check:**
```bash
# Verify your app is running on the right port
curl http://localhost:3000

# Check what's listening on the port
lsof -i :3000
```

### "Connection refused" when accessing myapp.test

**Cause 1:** Your application isn't running.

**Fix:**
```bash
# Start your application
cd /path/to/myapp
npm run dev  # or whatever starts your app
```

**Cause 2:** nginx isn't running.

**Fix:**
```bash
tube status
# If stopped:
tube start
```

**Cause 3:** nginx config not updated after adding project.

**Fix:**
```bash
tube restart
```

### Wrong application responds

**Cause:** Multiple projects on same port, or project name conflict.

**Check your projects:**
```bash
tube list
```

**Check nginx config:**
```bash
cat ~/.tube/nginx.conf | grep server_name
```

---

## GUI Issues

### Menu bar icon doesn't appear

**Cause 1:** App crashed on startup.

**Check logs:**
```bash
tube-gui 2>&1 | head -50
```

**Cause 2:** Running in unsupported environment (SSH, headless).

**Fix:** The menu bar app requires a graphical environment.

### Dashboard not loading

**Check 1: Is the port in use?**
```bash
lsof -i :3249
```

**Check 2: Is tube-gui running?**
```bash
pgrep -f tube-gui
```

**Check 3: Try a different port**

Edit `~/.tube/config.yaml`:
```yaml
proxy:
  dashboard_port: 3250
```

### Dashboard shows stale data

**Fix:** Refresh the page. Auto-refresh is not yet implemented.

---

## Configuration Issues

### "tunnel_prefix cannot be empty"

**Cause:** Config file is missing required fields.

**Fix:** Re-initialize config:
```bash
rm ~/.tube/config.yaml
tube init
```

Or manually add the field:
```yaml
tunnel_prefix: dev-
```

### Config changes not taking effect

**Fix:** Restart services after config changes:
```bash
tube restart
```

### "invalid project name"

**Valid project names must:**
- Be 1-63 characters
- Contain only lowercase letters, numbers, hyphens
- Not start or end with a hyphen

**Examples:**
- ✅ `myapp`
- ✅ `my-app`
- ✅ `app123`
- ❌ `-myapp` (starts with hyphen)
- ❌ `myapp-` (ends with hyphen)
- ❌ `MyApp` (uppercase)
- ❌ `my_app` (underscore)

---

## Build Issues

### "go: command not found"

**Fix:**
```bash
# Install Go
brew install go

# Verify
go version
```

### Build fails with import errors

**Fix:**
```bash
# Download dependencies
go mod download

# Tidy dependencies
go mod tidy
```

### "cgo: C compiler not found"

Some dependencies require a C compiler.

**Fix:**
```bash
# Install Xcode command line tools
xcode-select --install
```

---

## Getting More Help

### Check Service Logs

```bash
# nginx logs
tail -f ~/.tube/logs/nginx-error.log
tail -f ~/.tube/logs/nginx-access.log

# dnsmasq logs
tail -f ~/.tube/logs/dnsmasq.log
```

### Report an Issue

If you're still stuck:

1. Run `tube doctor` and save the output
2. Check existing issues: https://github.com/steig/tube/issues
3. Create a new issue with:
   - macOS version (`sw_vers`)
   - tube version (`tube version`)
   - Doctor output
   - Steps to reproduce
   - Error messages

---

## Reset Everything

If all else fails, start fresh:

```bash
# Stop services
tube stop

# Remove DNS configuration
sudo tube uninstall

# Remove tube configuration
rm -rf ~/.tube

# Re-initialize
tube init
sudo tube setup

# Add projects and start
tube add myapp 3000
tube start
```
