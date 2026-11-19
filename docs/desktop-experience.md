# Desktop experience

Part 8 makes Solace usable as a daily desktop writing application while preserving the local-first data model.

## Keyboard shortcuts

Solace uses Command on macOS and Control on Linux and Windows.

| Action | Shortcut |
| --- | --- |
| New entry | `Cmd/Ctrl+N` |
| Save current entry | `Cmd/Ctrl+S` |
| Global search | `Cmd/Ctrl+K` |
| Go to Today | `Cmd/Ctrl+Shift+T` |
| Toggle sidebar | `Cmd/Ctrl+B` |
| Toggle reading mode | `Cmd/Ctrl+Shift+R` |
| Open Settings | `Cmd/Ctrl+,` |
| Exit focus or reading mode | `Escape` |
| Lock privacy screen | `Ctrl+Shift+L` |

The fixed shortcuts are listed in **Settings → About**. The settings model leaves room for later customization, but Part 8 does not expose editable shortcut bindings.

## Writing controls

- Focus mode hides the entry list and section tabs while keeping the current entry editable.
- Reading mode removes editing chrome and makes the entry read-only until explicitly exited.
- Word and character counts update with the editor content.
- Font size, prose width, and line spacing use CSS variables so changes do not alter stored Tiptap JSON.
- Autosave pace supports quick, balanced, and relaxed delays. Manual save always remains available.
- Prose width stays bounded for readable line length, even on wide displays.
- Scrolling is smooth unless the operating system requests reduced motion.

## Settings

The compact Settings workspace separates General, Writing, Data & safety, and About. It persists theme, date format, editor appearance, autosave pace, and default export preferences through the existing local settings service. Backup and privacy controls remain native-only where they require filesystem or database access.

Storage paths are informational. Solace does not offer unsafe in-place relocation.

