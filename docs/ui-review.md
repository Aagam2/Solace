# UI review checkpoint

Last updated: 2026-11-20

## Scope

This checkpoint reviewed Solace before Part 6 implementation. The visual pass covered:

- Today dashboard and its empty/populated summary states.
- Entry list, empty editor, new-entry editor, rich-text toolbar, and metadata fields.
- Calendar, timeline, search, tags, favorites, trash, and settings navigation states.
- Light and dark themes.
- The standard 1240 x 780 desktop viewport and the 800 x 600 Wails minimum window size.
- The rebuilt production binary using an isolated temporary data root.

Browser preview was used for repeatable screenshots because it renders the same React/CSS frontend as the Wails WebView. The rebuilt native binary was then launched separately to verify that the updated production assets load under GTK4/WebKitGTK 6.0. No real journal database or external service was used.

## Findings and fixes

1. Light-theme secondary text was nearly invisible. Several text elements incorrectly used the Tailwind background token `--muted` as a foreground color. They now use `--muted-foreground`.
2. The shadcn/Tailwind tokens did not fully follow Solace's `data-theme` theme switch. Light and dark semantic tokens now share the warm paper-and-ink palette used by the custom interface.
3. Brand color was overloaded with the generic accent background token. Dedicated `--brand`, `--brand-strong`, and `--brand-foreground` tokens now keep interactive accents legible in both themes.
4. The 800 px minimum window produced a horizontal scrollbar and a cramped editor. At 820 px and below, the sidebar now becomes a compact icon rail and the page no longer overflows horizontally.
5. The disabled **Create tag** action could wrap onto two lines. Its label now remains on one line.
6. Keyboard focus was too dependent on browser defaults. Buttons, form controls, and the rich-text surface now receive a visible two-pixel focus ring.

The final pass found the visual hierarchy consistent across navigation, dashboard cards, the editor, and archive tools. Warm neutral surfaces, restrained shadows, serif editorial headings, and the brown accent remain the core Solace visual language.

## Lightweight design skills

The following project-local skills were added under `.agents/skills`:

| Skill | Installed size | Purpose |
| --- | ---: | --- |
| `web-design-guidelines` | 8 KB | Accessibility and interface review checklist |
| `frontend-design` | 28 KB | Visual direction and implementation guidance |
| `solace-ui` | 32 KB | Solace-specific desktop design system and QA checklist |
| `screenshot` | 92 KB | Screenshot capture guidance and helper scripts |
| `react-best-practices` | 424 KB | React implementation and performance guidance |

Total installed size is approximately 584 KB. No browser runtime, model, media pack, or large binary was downloaded.

The suggested community Wails skill was not installed because its Linux dependency guidance targeted GTK3/WebKit2GTK and conflicts with this verified Wails v3 project, which uses GTK4/WebKitGTK 6.0. Playwright browser downloads were also skipped because they can consume hundreds of megabytes and were unnecessary for this review.

## Verification

- `npm run typecheck`: passed.
- `npm run lint`: passed.
- `npm test -- --run`: 8 tests passed.
- `npm run build`: passed.
- `wails3 build`: passed.
- Rebuilt native binary launch with `XDG_DATA_HOME=/tmp/solace-visual-final`: passed.

## Remaining visual work

- Density and dialog follow-up completed on 2026-11-20: reduced the fixed page header to 62 px, replaced oversized Settings presentation with a compact tabbed work surface, tightened controls without shrinking body text below a comfortable desktop size, contained long dashboard previews, and replaced JavaScript confirms with an accessible Solace alert dialog. A fresh two-entry browser walkthrough covered all primary navigation destinations, search, calendar/timeline, history, reading/focus modes, light/dark themes, and Settings. At 800 x 600, the editor and Settings had no document-level horizontal overflow. The walkthrough also found and fixed a Tiptap read-mode false-dirty transition.

- Part 6 follow-up completed on 2026-11-18: Settings now uses a compact Export/Restore/Backups tab set rather than a long page. The dark-theme constrained-width pass confirmed readable hierarchy, no horizontal overflow, clear disabled browser-preview states, and contained internal scrolling; the production native window also launched successfully at its 1240 × 780 default with an 800 × 600 minimum.
- Part 7 follow-up completed on 2026-11-18: the fourth Privacy & safety tab and entry History tab were reviewed in light/dark rendered states and at the 800 × 600 viewport. The warning hierarchy, disabled browser-preview controls, revision rows, tab focus order, and internal scrolling remained clear. The isolated GTK application launched successfully, but this environment lacked `scrot`, `gnome-screenshot`, and `xdotool`, so native-window image capture was unavailable; native startup and embedded assets were verified separately from the rendered browser screenshots.
- Part 8 follow-up completed on 2026-11-18: Application Settings now uses General, Writing, Data & safety, and About tabs. Focus mode, wrapped formatting controls, live writing counts, writing-preference controls, storage-path disclosure, and light/dark palettes were rendered and reviewed. The constrained IAB pass was narrower than the supported 800 px minimum and still showed no application-level horizontal clipping; the 1240 × 780 and 800 × 600 shell baselines remained clean. The final GTK production binary served the reviewed assets and reopened from the isolated `/tmp/solace-part8-native` profile. Native-window image capture remains unavailable in this environment.
- Repeat the platform-specific visual pass on Windows and macOS before packaging because native file dialogs and font rendering differ.
- Perform a final platform pass before packaging on Windows and macOS, where native font rasterization, window chrome, pickers, and system theme behavior differ.
- Keep LLM/RAG interface concepts out of the application until that future feature is explicitly enabled.

## Scrolling and density follow-up

Completed: 2026-11-18

The follow-up pass reduced scrolling without turning every surface into nested navigation:

- Today now uses three keyboard-accessible tabs: **Today’s pages**, **On this day**, and **Recent**. The redundant full entry workspace was removed from beneath the dashboard; opening or creating an entry moves directly to the dedicated writing workspace.
- The welcome area is shorter and changes between horizontal and stacked layouts based on useful width.
- The entry editor now keeps the title and save/actions visible while separating **Writing** from **Details & media**. Dates, mood, location, tags, draft state, and attachments recede until needed, leaving a substantially larger writing surface.
- The application shell is fixed to the desktop window. The top bar and sidebar remain stable while long content scrolls inside the relevant content surface.
- At 800 x 600, the icon rail, entry list, and editor remain visible together; only the editor pane scrolls when its content genuinely exceeds the available height.
- Both tab sets implement `role="tab"`, selected state, controlled panels, roving tab focus, Home/End, and left/right arrow navigation.

The follow-up was visually verified at 1240 x 780 and 800 x 600 in browser preview, then packaged and launched in the native Wails application with the isolated data root `/tmp/solace-tabs-final`.
