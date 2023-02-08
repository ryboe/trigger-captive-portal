# trigger-captive-portal

Trigger the captive portal on that crappy public Wi-Fi network your Mac is
connected to, based on [`captive-browser`](https://github.com/FiloSottile/captive-browser)
by the brilliant [FiloSottile](https://words.filippo.io/captive-browser/) 🙇‍♂️. I
made it because I wanted to understand how `captive-browser` and captive portals
work.

## Install

This will download the macOS binary to `/usr/local/bin`, which is in your
`$PATH`. It's a universal binary, so it will work on both Intel and Apple
Silicon Macs.

```zsh
curl --retry 3 --retry-max-time 120 -sSL https://github.com/ryboe/trigger-captive-portal/releases/latest/download/trigger-captive-portal | sudo tar -xzf - -C /usr/local/bin
```

## Usage

Just type this command in your terminal. You may need to close Chrome before
running it.

```zsh
trigger-captive-portal
```

## Differences between `trigger-captive-portal` and `captive-browser`

* Mac only
* updated to use Go modules
* no configs
* falls back to using Chromium if Chrome is not installed
