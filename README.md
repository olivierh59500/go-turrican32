# Turrican32 Go

A native Go/Ebitengine conversion of **T32k**, the Turrican-style game created
for the 32K game competition at Mekka & Symposium 2000.

The game runs at 60 Hz in its original 320 × 240 viewport. The level contains
783 objects across 21 screens, with platforms, moving enemies, turrets,
collectible gems, weapon upgrades, health bonuses and R restart points.
The original sprites, bitmap font and level data are decoded from the game
binary. The procedural cloud and rock textures are generated in Go from the
embedded texture programs.

## Run on desktop

Go 1.26 or newer is required.

```sh
go run .
```

Use `go run . -play` to enter the level directly or `go run . -mute` to disable
audio. All runtime assets are embedded; the original executable is not needed
to play.

| Action | Keyboard |
| --- | --- |
| Move | Left/Right arrows or A/D |
| Jump | Up arrow, W or Space |
| Fire / start | Ctrl |
| Start | Enter |
| Return to title / restart | R |
| Pause | P |
| Mute | M |
| Return to title / quit from title | Escape |

Release the jump button early for a shorter jump. Weapon pickups increase the
number of projectiles. Collecting an R marker sets the next respawn position.

## Android

The ARM64 Android application uses a virtual joystick and a separate Fire
button. Push the joystick upward to jump. Fire also starts the game from the
title. Reset and Pause are available in the side margins. The screen stays
awake while the game is visible.

```sh
./scripts/run-android.sh --build-only
./scripts/run-android.sh
```

Building requires Android SDK 36, an Android NDK supported by Ebitengine,
Java 17 and the Go toolchain. Set `ANDROID_HOME` and `JAVA_HOME` if their
installations are not detected. The second command installs and launches the
application on an authorized device connected through ADB. Set
`ANDROID_SERIAL` when more than one device is connected.

The package name is `com.olivierh.turrican32`. The generated debug APK is
`android/app/build/outputs/apk/debug/app-debug.apk`.

## Graphics and audio

DCK v1.0.14 provides the image-slot renderer, scrolling background, music
playback and capture/export facilities. Gameplay, collisions and the native
asset decoder are implemented in Go in this project. The simulation uses the
original movement constants, jump release, projectile directions and camera
smoothing, independently of the display refresh rate.

The soundtrack uses **Turrican World 1-1** on the title and **Turrican II:
The Desert Rocks** during play. These are YM replacements for the original
C64 SID soundtrack. Audio is opened through DCK.

## Captures

```sh
go run . -play -capture captures/level -frame 120
go run ./cmd/video -duration 3m -poster-at 25s
```

The video command records a deterministic gameplay presentation with its
soundtrack to `recordings/turrican32.mp4` and writes a PNG poster. It requires
FFmpeg. A WebM copy suitable for the portfolio can be produced with:

```sh
ffmpeg -i recordings/turrican32.mp4 \
  -c:v libvpx-vp9 -crf 36 -b:v 0 -cpu-used 2 -threads 4 \
  -c:a libopus -b:a 96k recordings/turrican32.webm
```

## Source layout and checks

- `internal/engine`: platform physics, entities, pickups and restart state.
- `internal/game`: DCK composition, title, viewport and input handling.
- `internal/controls`: virtual joystick with contact ownership and a dead zone.
- `internal/extract`: sprite, font and procedural texture decoding.
- `assets`: embedded graphics, level and YM music.
- `mobile` and `android`: Android binding and application shell.

```sh
go test ./...
go vet ./...
```

Tests cover the native spawn and floor height, jump release, weapon/gem
separation, projectile banks and respawn/reset behaviour. A fixture containing
73 samples executed by the original x86 floor routine checks its one-pixel
platform separation. This is a focused collision check; it does not establish
frame-for-frame equivalence for every gameplay sequence. The title movement
and explosion visuals are recreated, rather than exact translations of their
original routines.

## Credits

The original production credits Myth for the game engine, Arthus for graphics,
Arthus and Kojote for design, TmbINC for the title, Ryg for the texture generator
and KB for C64 music and SID emulation. Turrican was created by Manfred Trenz;
the replacement YM tracks are by Jochen Hippel. This conversion is by
Olivier Houte / Malakh Software.
