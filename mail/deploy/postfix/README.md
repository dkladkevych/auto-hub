# Postfix + PostgreSQL Integration for Auto-Hub Mail

Postfix is configured to look up domains and mailboxes directly from the Auto-Hub PostgreSQL database using the existing `domains` and `mailboxes` tables.

## How it works

- **`virtual_mailbox_domains`** → `pgsql-virtual-domains.cf`  
  Queries `domains` table. If `is_active = 0`, Postfix rejects mail for the domain.

- **`virtual_mailbox_maps`** → `pgsql-virtual-mailbox.cf`  
  Queries `mailboxes` table and returns `domain/user/` (relative path under `/var/mail/vhosts`).  
  If `is_active = 0` or `can_receive = 0`, Postfix rejects with `550 5.1.1`.

- **`virtual_transport = virtual`**  
  Postfix delivers directly to Maildir using the path returned by `virtual_mailbox_maps`.  
  The trailing `/` returned by the query tells Postfix to use **Maildir** format.

- **`virtual_uid_maps = static:5000`** / **`virtual_gid_maps = static:5000`**  
  All delivered files are owned by `vmail:vmail` (uid/gid 5000).

- **`virtual_alias_maps`** (optional)  
  Currently set to identity mapping. Can be extended later for catch-all or role addresses.

## Prerequisites

- PostgreSQL running with the Auto-Hub database
- `vmail` user/group created (uid/gid 5000)
- `/var/mail/vhosts` exists and is writable by `vmail`
- The application already creates `maildir_path` entries like `/var/mail/vhosts/<domain>/<local>/`

## Exact Ubuntu 25.10 Setup Commands

### 1. Install Postfix with PostgreSQL support

```bash
sudo apt update
sudo apt install -y postfix postfix-pgsql
```

During installation, select **"Internet Site"** and set the system mail name (e.g., `mail.auto-hub.ca`).

### 2. Create the `vmail` user

```bash
sudo groupadd -g 5000 vmail
sudo useradd -g vmail -u 5000 vmail -d /var/mail -m
sudo mkdir -p /var/mail/vhosts
sudo chown vmail:vmail /var/mail/vhosts
sudo chmod 755 /var/mail/vhosts
```

### 3. Copy map files to Postfix

Replace `YOUR_DB_PASSWORD` in each file, then copy:

```bash
sudo cp pgsql-virtual-domains.cf  /etc/postfix/
sudo cp pgsql-virtual-mailbox.cf  /etc/postfix/
sudo cp pgsql-virtual-alias.cf    /etc/postfix/
sudo chmod 640 /etc/postfix/pgsql-*.cf
sudo chown root:postfix /etc/postfix/pgsql-*.cf
```

### 4. Update `main.cf`

Run these commands to safely update Postfix config:

```bash
# Disable local delivery for virtual domains
sudo postconf -e "mydestination = localhost"
sudo postconf -e "inet_interfaces = all"

# PostgreSQL maps
sudo postconf -e "virtual_mailbox_domains = pgsql:/etc/postfix/pgsql-virtual-domains.cf"
sudo postconf -e "virtual_mailbox_maps = pgsql:/etc/postfix/pgsql-virtual-mailbox.cf"
sudo postconf -e "virtual_alias_maps = pgsql:/etc/postfix/pgsql-virtual-alias.cf"

# Delivery settings
sudo postconf -e "virtual_transport = virtual"
sudo postconf -e "virtual_mailbox_base = /var/mail/vhosts"
sudo postconf -e "virtual_uid_maps = static:5000"
sudo postconf -e "virtual_gid_maps = static:5000"
sudo postconf -e "virtual_minimum_uid = 5000"
sudo postconf -e "virtual_mailbox_limit = 0"

# Basic relay restrictions (adjust if you add SPF/DKIM/DMARC later)
sudo postconf -e "smtpd_recipient_restrictions = permit_mynetworks, reject_unauth_destination"
```

### 5. Restart Postfix

```bash
sudo systemctl restart postfix
sudo postfix check
```

### 6. Test lookups

```bash
# Test domain lookup (should return "1")
sudo postmap -q auto-hub.ca pgsql:/etc/postfix/pgsql-virtual-domains.cf

# Test mailbox lookup (should return "auto-hub.ca/support/")
sudo postmap -q support@auto-hub.ca pgsql:/etc/postfix/pgsql-virtual-mailbox.cf

# Test disabled mailbox (should return nothing)
# After setting is_active=0 or can_receive=0 for a mailbox:
# sudo postmap -q disabled@auto-hub.ca pgsql:/etc/postfix/pgsql-virtual-mailbox.cf
```

### 7. Test incoming mail

From an external Gmail account, send an email to `support@auto-hub.ca`.  
It should appear in `/var/mail/vhosts/auto-hub.ca/support/new/` within seconds.

```bash
ls -la /var/mail/vhosts/auto-hub.ca/support/new/
```

## Rejection Behavior

| Condition | Postfix Response | Reason |
|---|---|---|
| Domain `is_active = 0` | `450 4.1.1` or `550 5.1.2` | Domain not in `virtual_mailbox_domains` |
| Mailbox `is_active = 0` | `550 5.1.1` | Recipient not in `virtual_mailbox_maps` |
| Mailbox `can_receive = 0` | `550 5.1.1` | Recipient not in `virtual_mailbox_maps` |

## Notes

- `virtual_mailbox_maps` returns `domain/user/` (relative path). Postfix appends this to `virtual_mailbox_base = /var/mail/vhosts`, producing `/var/mail/vhosts/domain/user/`. The trailing `/` triggers **Maildir** delivery.
- Dovecot IMAP will read the same Maildir. Ensure Dovecot runs as `vmail:vmail`.
- Outbound SMTP (127.0.0.1:25) remains unchanged and unaffected.
- If you prefer LMTP delivery to Dovecot instead of Postfix `virtual`, replace `virtual_transport = virtual` with `virtual_transport = lmtp:unix:private/dovecot-lmtp` and remove `virtual_uid_maps` / `virtual_gid_maps`.
