# trigger-captive-portal

Trigger the captive portal on that crappy public Wi-Fi network your Mac is
connected to, based on [`captive-browser`](https://github.com/FiloSottile/captive-browser)
by the brilliant [FiloSottile](https://words.filippo.io/captive-browser/) 🙇‍♂️. I
made it because I wanted to understand how `captive-browser` and captive portals
work.

## Differences between `trigger-captive-portal` and `captive-browser`

* Mac only
* updated to use Go modules
* no configs
* falls back to using Chromium if Chrome is not installed
