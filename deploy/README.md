# Deploying Resumaker with Ansible

This playbook deploys Resumaker to `resumaker.skinet.org` as:

- a static Go binary at `/opt/resumaker/resumaker`
- a hardened `resumaker.service` systemd unit
- a local PostgreSQL database
- nginx terminating TLS with a Let's Encrypt certificate and proxying to
  `127.0.0.1:8080`

The SSH connection comes from your existing `~/.ssh/config` entry for
`resumaker.skinet.org`; no user or key is stored in the inventory.

## Assumptions

- The target is Debian or Ubuntu.
- The target has SSH access, sudo, and ports 80/443 reachable from the internet.
- DNS for `resumaker.skinet.org` points at the target.
- The controller has Ansible and Go installed. The playbook cross-compiles the
  binary locally for the target's architecture (`amd64` or `arm64`).

## First-time setup

Install the PostgreSQL collection used by the playbook:

```bash
ansible-galaxy collection install -r requirements.yml
```

Create the secrets file:

```bash
cp group_vars/all/vault.yml.example group_vars/all/vault.yml
# edit it, then either leave it plaintext (git-ignored) or encrypt it:
ansible-vault encrypt group_vars/all/vault.yml
```

Fill in:

- `resumaker_master_key` — `openssl rand -base64 32`. Keep it safe; losing it
  makes stored resumes unreadable.
- `resumaker_db_password` — a URL-safe random string such as
  `openssl rand -hex 24`.
- `resumaker_letsencrypt_email`.
- At least one OAuth pair: GitHub or Google.

Register the OAuth callback URLs with the providers:

- GitHub: `https://resumaker.skinet.org/auth/github/callback`
- Google: `https://resumaker.skinet.org/auth/google/callback`

## Deploy

From this directory:

```bash
cd deploy
ansible-playbook site.yml
```

If `vault.yml` is encrypted, or your sudo needs a password:

```bash
ansible-playbook site.yml --ask-vault-pass --ask-become-pass
```

On the first run the playbook serves the site over HTTP, completes the ACME
webroot challenge, then installs the HTTPS nginx configuration. Later runs
skip certificate issuance and use the existing certificate.

## Updating

Re-run the playbook. It rebuilds the binary, replaces it if the checksum changed,
restarts the service, and reloads nginx when configuration changes:

```bash
cd deploy
ansible-playbook site.yml --ask-vault-pass
```

## Checking the deployment

```bash
curl -fsS https://resumaker.skinet.org/healthz
ssh resumaker.skinet.org 'systemctl status resumaker --no-pager'
ssh resumaker.skinet.org 'journalctl -u resumaker -n 50 --no-pager'
ssh resumaker.skinet.org 'nginx -t'
```

## Notes

- Secrets live in `group_vars/all/vault.yml`, which is git-ignored. Never commit
  it; if it is encrypted, keep the vault password outside the repository.
- The app binds only to loopback. nginx is the public listener.
- The playbook assumes the stock Debian/Ubuntu `pg_hba.conf` allows password
  authentication from `127.0.0.1`. If you customised it, allow the `resumaker`
  role to connect over loopback.
- Certbot's systemd timer handles renewal, and a deploy hook reloads nginx after
  a renewal.
- The playbook does not manage a firewall. Ensure SSH, HTTP and HTTPS are allowed
  on the host.
- `site.yml` fails early with a readable message if a required secret is missing
  or the target is not Debian/Ubuntu. Use `--check` cautiously: the build and
  certbot steps are designed for real runs, not full dry runs.
