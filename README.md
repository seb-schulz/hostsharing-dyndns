# hostsharing-dyndns

It provides a simple DynDNS service running on https://hostsharing.net/ and running with Fritz!Box.

## Table of Contents

- [Installation](#installation)
- [Contributing](#contributing)
- [License](#license)

## Installation

- Run `make build`
- Copy `hostsharing-dyndns` to fastcgi directory. For example: `scp hostsharing-dyndns xzy00-user:doms/dyndns.example.com/fastcgi-ssl/`
- Provide htaccess file into `doms/dyndns.example.com/htdocs-ssl/`

### Example for htaccess file

```
RewriteRule ^(.*)$ /fastcgi-bin/hostsharing-dyndns/$1 [L]
```

## How to configure DynDNS Updater URL

See <https://avm.de/service/wissensdatenbank/dok/FRITZ-Box-7590/30_Dynamic-DNS-in-FRITZ-Box-einrichten/>.
Example: `https://dyndns.example.com/?user=<username>&passwd=<pass>&ipaddr=<ipaddr>&ip6addr=<ip6addr>`
