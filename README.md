# rolodex
An SSH directory for your terminal with multi-method authentication support.

## Features

- **Multiple Authentication Methods**: Support for SSH agent, identity files, OS keyring, and passwords
- **Automatic Priority Chain**: Tries more secure methods first, falls back gracefully
- **Cross-Platform**: Works on Windows, macOS, and Linux
- **Config Management**: Create and delete host configurations

## Upcoming Features
- **SSH Config File Support**: Support for SSH config file (e.g. `~/.ssh/config`)
- **Folders**: Sort hosts into groups

## Authentication Methods

Rolodex supports multiple authentication methods with automatic fallback.  Configure any combination of the following (at least one required):

### Priority Order

1. **SSH Agent** (Most Secure) - Uses running SSH agent with loaded keys
2. **Identity File** - SSH private key files (RSA, Ed25519, ECDSA, DSA)
3. **OS Keyring** - Windows Credential Manager, macOS Keychain, Linux Secret Service
4. **Password** (Least Secure) - Plain password authentication

## Configuration

Create a `config.json` file in the project root:

```json
{
  "hosts": [
    {
      "name": "Server",
      "host": "server.example.com",
      "port": 22,
      "user": "root",
      
      "ssh_agent": true,
      "identity_file": "~/.ssh/id_ed25519",
      "identity_passphrase": "",
      "keyring_service": "rolodex",
      "keyring_account": "root@server",
      "password": "fallback_password"
    }
  ]
}
```

### Configuration Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Display name for the host |
| `host` | string | Yes | Hostname or IP address |
| `port` | int | Yes | SSH port (usually 22) |
| `user` | string | Yes | SSH username |
| `ssh_agent` | bool | No | Use SSH agent if available |
| `identity_file` | string | No | Path to SSH private key (supports `~\` expansion) |
| `identity_passphrase` | string | No | Passphrase for encrypted identity file |
| `keyring_service` | string | No | OS keyring service name |
| `keyring_account` | string | No | OS keyring account identifier |
| `password` | string | No | SSH password |

### Example Configurations

**SSH Agent Only:**
```json
{
  "name": "Server",
  "host": "server.example.com",
  "port": 22,
  "user": "root",
  "ssh_agent": true
}
```

**Identity File with Passphrase:**
```json
{
  "name": "Server",
  "host": "server.example.com",
  "port": 22,
  "user": "root",
  "identity_file": "~/.ssh/id_rsa",
  "identity_passphrase": "my_passphrase"
}
```

**Password Only (Backward Compatible):**
```json
{
  "name": "Server",
  "host": "server.example.com",
  "port": 22,
  "user": "root",
  "password": "my_password"
}
```

**Maximum Fallback (All Methods):**
```json
{
  "name": "All Methods Server",
  "host": "server.example.com",
  "port": 22,
  "user": "root",
  "ssh_agent": true,
  "identity_file": "~/.ssh/id_ed25519",
  "identity_passphrase": "my_passphrase",
  "keyring_service": "rolodex",
  "keyring_account": "root@server",
  "password": "my_password"
}
```

## Installation

```bash
git clone https://github.com/nathanlytang/rolodex.git
cd rolodex
go mod download
go build
```

## Usage

1. Copy `config.example.json` to `config.json`
2. Edit `config.json` with your SSH hosts and [authentication details](#example-configurations).  Alternatively you can add hosts interactively within the program.
3. Run `./rolodex`

## Tips

Rolodex automatically logs all connection attempts and debugging information to date-based files in the `logs/` directory.  If you encounter connection issues, check today's log file for detailed diagnostic information.

To use the program anywhere, add it to your PATH.

### Security Best Practices

1. **Prefer SSH Agent**: Most secure, keys never touch disk in decrypted form
2. **Use Identity Files**: Better than passwords, supports key rotation
3. **Use Encrypted Keys**: Protect identity files with passphrases
4. **OS Keyring**: Store passwords in system keyring instead of config file
5. **Avoid Plain Passwords**: Only use as last resort or for legacy systems
