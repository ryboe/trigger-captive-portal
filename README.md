# trigger-captive-portal

These things are *the worst*!

![captive_portal_collage](https://user-images.githubusercontent.com/1250684/217684916-d9be848a-5603-46b5-af3c-5042b3753604.jpg)

Sometimes they just won't pop-up, which means you get no internet on that crappy
public Wi-Fi network. These login pages are called "captive portals" and they
have an astonishingly high failure rate. In these situations,
`trigger-captive-portal` can help. It will send a special request to try to
force the pop-up to pop up. It only works on Macs and it requires that you have
Chrome or Chromium installed. It's based on
[`captive-browser`](https://github.com/FiloSottile/captive-browser) by the
brilliant [FiloSottile](https://words.filippo.io/captive-browser/) 🙇‍♂️. If you
use Windows or Linux, I recommend trying [`captive-browser`](https://github.com/FiloSottile/captive-browser).

## Install (macOS only)

This will download a single binary to `/usr/local/bin`. It works on both Intel
and Apple Silicon Macs.

```zsh
curl --retry 3 --retry-max-time 120 -sSL https://github.com/ryboe/trigger-captive-portal/releases/latest/download/trigger-captive-portal.tar.gz | sudo tar -xzf - -C /usr/local/bin trigger-captive-portal
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
