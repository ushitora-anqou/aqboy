# AQboy

Yet another Game Boy emulator written in Go.

## Build and Run

### SDL2

    go run -tags sdl2 . ROM-FILE-PATH

### Ebiten (Native)

    go run -tags ebiten . ROM-FILE-PATH

### Ebiten+Wasm

    GOOS=js GOARCH=wasm go build -tags ebiten,wasm -o aqboy.wasm github.com/ushitora-anqou/aqboy
