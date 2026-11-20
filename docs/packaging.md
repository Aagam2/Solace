# Packaging and installation

Solace is a Wails v3 desktop application. Linux is the first supported packaging target. Windows and macOS build scaffolds are maintained, but those packages must be built and signed on their target platforms before release.

The checked-in `build/solace-icon.svg` is the editable source for the application identity. `build/appicon.png` is its 1024 px raster source for Wails; builds generate `build/windows/icon.ico` and `build/darwin/icons.icns`. The Linux package installs a 128 px launcher icon, while `frontend/public/solace-icon.svg` and the compact React mark keep browser and in-app identity consistent. Keep the generous transparent safe area when changing the artwork so platform masks do not clip it.

Linux uses `app.solace.desktop` as the GTK/Wayland application ID. The package installs `/usr/share/applications/app.solace.desktop.desktop`, whose desktop ID and `StartupWMClass` match that runtime identity, and installs `solace.png` into the hicolor icon theme. Post-install and post-remove hooks refresh the desktop and icon caches. This matching is required for GNOME to group the window under the Solace launcher instead of displaying Wails’ fallback identity or a generic icon. Running an unpackaged binary before installing its desktop entry can still bypass shell integration; install the generated package before judging launcher behavior.

## Build requirements

All platforms require:

- Go 1.26 or newer.
- Node.js 22 or newer and npm.
- Wails CLI v3 (`wails3`).
- The checked-in frontend dependencies from `frontend/package-lock.json`.

Linux native builds additionally require a C compiler, GTK 4 development headers, and WebKitGTK 6.0 development headers. On Ubuntu 24.04 this normally includes `build-essential`, `libgtk-4-dev`, and `libwebkitgtk-6.0-dev`. Run `wails3 doctor` before building.

No network connection is required to run an installed Solace application. A first build may need network access if Go or npm dependencies are not already cached.

## Development and production builds

From the repository root:

```bash
wails3 dev -config ./build/config.yml -port 9245
```

The production binary is created at `bin/solace`:

```bash
wails3 build
```

Linux package tasks are available individually so a builder does not need to install every packaging tool:

```bash
wails3 task linux:create:deb
wails3 task linux:create:rpm
wails3 task linux:create:appimage
```

Do not use the Docker cross-build task merely for a normal native build; its image is intentionally large. Windows and macOS use the checked-in `build/windows` and `build/darwin` tasks on those platforms.

## Linux runtime requirements

The native package declares GTK 4 and WebKitGTK 6.0 runtime dependencies. Debian-family packages use `libgtk-4-1` and `libwebkitgtk-6.0-4`; RPM and Arch package metadata contains the corresponding platform names.

Install a generated Debian package with the system package manager so runtime dependencies are resolved:

```bash
sudo apt install ./bin/solace.deb
```

An AppImage can be made executable and launched directly. It still relies on the display/session facilities available on the Linux desktop.

## Data location

The installation directory contains the application only. Journal data remains separate so upgrades and package removal do not overwrite it.

- Linux: `~/.local/share/solace`
- macOS: `~/Library/Application Support/solace`
- Windows: `%AppData%\solace`

The precise active path is shown read-only in **Settings → About**. Solace does not expose storage relocation because moving a live journal safely requires a verified migration workflow.

The data directory contains `solace.db`, imported media, thumbnails, backups, and logs. The privacy lock is an application screen; it does not encrypt these files.

## Back up and restore

Use **Settings → Data & safety → Backups** to create a complete local backup, preferably including attachments. Use **Export** for readable or portable copies. Use **Restore** to inspect and verify a complete archive before replacement; Solace creates a recovery archive before replacing a non-empty journal.

Before moving to another computer or uninstalling permanently:

1. Create a complete archive with attachments.
2. Copy it to another drive or a location outside the Solace data folder.
3. Verify it from the Restore screen before deleting the original profile.

## Uninstalling safely

Removing the application package does not intentionally delete the data directory. This prevents a normal uninstall from destroying a journal.

After uninstalling, retain the data directory if Solace may be reinstalled. Only delete the platform-specific Solace data directory after making and verifying an external archive. Package removal scripts must never remove user journal data.

