# iina-tcode

This package adds tcode/funscript support to [IINA](https://github.com/iina/iina)

## Setup & Build

```sh
go build -o tcode-player ./cmd
npm i
npm run build
```

In order to load the plugin locally while developing new features,
it's recommended to create a `iinaplugin-dev` symlink under the pluign folder:

```
ln -s iina-tcode ~/Library/Application\ Support/com.colliderli.iina/plugins/iina-tcode.iinaplugin-dev
```
