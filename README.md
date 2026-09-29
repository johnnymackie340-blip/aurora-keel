# Aurora Keel

A night-sea courier. You pilot a brass-and-glass skiff across a black inland sea under a permanent aurora. Sing the glass buoys, then reach the moon-dock before the freeze seals the lane.

No guns. The only trick is the keel-hop: a short lift over ice, diving herons, and shock rings.

## Play on Windows

`AuroraKeel.exe` is a standalone 64-bit game. Double-click it. Scores save under your user config folder, in `AuroraKeel/scores.json`.

| Key | Action |
| --- | --- |
| W or Up | Thrust |
| S or Down | Reverse |
| A or Left | Steer left |
| D or Right | Steer right |
| Space | Keel-hop |
| Enter | Confirm |
| Esc | Pause or back |

## Charts

1. Glassmouth — learn the keel
2. Heron Narrows — ice, herons, krill
3. Leviathan Road — shock rings
4. Mirror Whale — ring six buoys, then the ice gate opens
5. Uncharted Night — endless. Buoys buy back seconds.

## Build

Requires Go 1.23 or newer.

```bash
# Windows exe
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w -H windowsgui" -o AuroraKeel.exe .

# Browser
GOOS=js GOARCH=wasm go build -o keel.wasm .
```

Copy `keel.wasm` and `$(go env GOROOT)/lib/wasm/wasm_exec.js` next to a page that creates a canvas with id `keel`.
