# 🎮 gbemu — Game Boy Emulator in Go

A Game Boy (DMG) emulator written in Go, using [Ebitengine](https://ebitengine.org/) for rendering.

## Features

| Component | Status |
|-----------|--------|
| CPU (LR35902) | ✅ Full instruction set + CB prefix |
| MMU / Memory map | ✅ |
| MBC1 / MBC3 / MBC5 | ✅ |
| PPU (BG + Window + Sprites) | ✅ |
| Timer | ✅ |
| Joypad | ✅ |
| Interrupts | ✅ |
| Sound (APU) | 🔜 |
| Game Boy Color | 🔜 |

---

## Quick Start (GitHub Codespaces)

1. **Open the repo in Codespaces** — the `.devcontainer` will install Go + a virtual desktop (noVNC).
2. Wait for `postCreateCommand` to finish (`go mod tidy`).
3. In the terminal:

```bash
make build
ROM=path/to/game.gb make run
```

4. Open the **Ports** tab → click the 🎮 link for port `6080` → noVNC opens in your browser.
5. The game window will appear on the virtual desktop. Enjoy!

> **No ROM?** Try the open-source test ROMs from [gbdev/gb-test-roms](https://github.com/gbdev/gb-test-roms).

---

## Controls

| Key | Button |
|-----|--------|
| Arrow keys | D-pad |
| `Z` | A |
| `X` | B |
| `Enter` | Start |
| `Backspace` | Select |

---

## Build Options

```bash
make build      # native binary
make tidy       # go mod tidy
make wasm       # WebAssembly build → web/gbemu.wasm
make check      # compile all packages
```

---

## Architecture

```
cmd/gbemu/main.go       ← Ebiten game loop
internal/
  cartridge/            ← ROM loading + MBC1/3/5
  mmu/                  ← Memory map + bus
  cpu/                  ← LR35902 (opcodes.go, cb.go)
  ppu/                  ← Scanline renderer
  timer/                ← DIV + TIMA
  joypad/               ← Key polling
```

---

## References

- [Pan Docs](https://gbdev.io/pandocs/) — the definitive GB hardware reference
- [Game Boy CPU Manual](http://marc.rawer.de/Gameboy/Docs/GBCPUman.pdf)
- [Ebitengine docs](https://ebitengine.org/en/documents/)

---

## Super Game Boy (SGB) Support

| Feature | Status |
|---------|--------|
| PAL01/PAL23/PAL03/PAL12 — color palettes | ✅ |
| ATF_SET — per-tile palette mapping | ✅ |
| PAL_SET + PAL_TRN — system palette RAM | ✅ |
| CHR_TRN + PCT_TRN — custom border | ✅ |
| MASK_EN — screen mask modes | ✅ |
| MLT_REQ — multiplayer (2/4 player) | 🔜 |

### Running in SGB mode (default)

```bash
./gbemu --sgb game.gb      # SGB mode (256×224 window with border)
./gbemu --dmg  game.gb     # Force DMG mode (160×144, green tones)
./gbemu        game.gb     # Defaults to SGB
```

Window sizes:
- **SGB**: 256×224 × 3 = 768×672
- **DMG**: 160×144 × 3 = 480×432
