# iina-tcode

This package adds tcode/funscript support to [IINA](https://github.com/iina/iina)

## Installation
As of writing (Sept 6th 2023), IINA's plugin support is still in beta and requires you to enable it by running the following command in Terminal.app or a similar application.

`defaults write com.colliderli.iina iinaEnablePluginSystem true`

This will enable the plugin tab within the IINA settings menu (will require a restart of IINA if it's open):

<img width="800" alt="image" src="https://github.com/saturdaythrowaway/iina-tcode/assets/68406006/b3052d49-3622-4202-9561-381366348e92">

Next, you need to click "Install from GitHub..." and put `saturdaythowaway/iina-tcode` into the input:

<img width="800" alt="image" src="https://github.com/saturdaythrowaway/iina-tcode/assets/68406006/46211fce-0d9f-4bac-b287-0560660f2b81">

You'll get a notice indicating that the plugin needs to access the Filesystem as well as make Network Requests. The filesystem access is so we can load Funscript metadata. The Network Requests are use to communicate with the [`tcode-player`](https://github.com/saturdaythrowaway/iina-tcode/tree/main/cmd) executable which is how IINA communicates with the OSR2/SR6/SSR1/etc...

Once this is all setup, you should be able to load any video with a .funscript in the same folder and it should start replicating the movements on your TCode compatible device. Along with the typical axis (stroke, surge, sway, twist, ...) I've also added the ability to let `tcode-player` know if the funscript is a "hard" or "soft" script by adding the `.hard` or `.soft` prefix before the stroke axis. For example, `some-script.hard.funscript` or `multi-axis.soft.twist.funscript`. If you have multiple versions of the same funscript, you can use `.alt` to let the player know that this is an alternate file. User preferance for soft/hard or normal/alt can be configured within the IINA plugin settings, as well as a way to set the minimum and maximum stroke length: 
<img width="500" alt="image" src="https://github.com/saturdaythrowaway/iina-tcode/assets/68406006/316aaae4-a48e-4c8b-a7a4-05ae1aeb9b59">

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
