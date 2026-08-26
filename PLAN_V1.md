# Paperfolio Embedded PDF Viewer Plan (V1)

## 1. Objective

Replace Paperfolio's current external PDF opening behavior with an in-application PDF viewer. Imported PDFs should be rendered inside the Paperfolio desktop window, while existing paper management, highlights, notes, ideas, and local storage behavior remain unchanged.

The viewer should support the basic interactions expected when reading a research paper:

- Render existing PDF files inside the application.
- Navigate between pages.
- Zoom in and out.
- Scroll through document pages.
- Search document text where supported by the bundled PDF.js viewer.
- Keep PDF data local and avoid sending document contents to a network service.
- Continue supporting PDF import and replacement through the existing Fyne file dialogs.

## 2. Library assessment

### Boxes and Glue

`github.com/boxesandglue/boxesandglue` is not the correct dependency for this feature. It is a Go typesetting and PDF-generation library based on TeX-style layout algorithms. It creates PDFs from structured content; it is not intended to parse and display arbitrary existing PDFs in an interactive reader.

### PDF.js

PDF.js is the appropriate rendering engine for this requirement. It parses existing PDF files in JavaScript and renders pages to browser canvas elements. It also provides the established viewer UI for page navigation, zoom, text search, and document interaction.

PDF.js will be bundled locally with the application rather than loaded from a CDN. This keeps paper contents and viewer assets local and makes the application usable offline.

## 3. Proposed architecture

### Native application

The Go/Fyne layer remains responsible for:

- Importing and replacing PDFs.
- Storing PDFs in `Paperfolio_Data/uploads`.
- Resolving the selected paper's PDF path.
- Creating the reader view.
- Passing a safe local document URL or document bytes to the embedded viewer.
- Handling viewer lifecycle when switching papers or closing the application.

### Embedded viewer

A webview component will host a local PDF.js viewer page inside the Fyne window. The viewer assets will be packaged with the application or embedded into the Go binary, depending on the selected webview integration's asset-loading API.

The viewer will receive a PDF through one of these mechanisms, selected during implementation based on the chosen dependency's supported APIs:

1. A local application-controlled URL with a tightly scoped file handler.
2. A local HTTP server bound only to loopback and serving the selected PDF plus static PDF.js assets.
3. In-memory PDF bytes passed to JavaScript through a bridge or data URL, if supported without size or memory problems.

The preferred mechanism is a local, loopback-only document endpoint or an integration-native asset handler. Direct unrestricted `file://` access should be avoided where possible.

### Fyne integration

The current `Reader` tab will be changed from a text label plus “Open PDF in default reader” button to an embedded viewer container. The existing external `openPDF` function will be removed if it is no longer used.

Because Fyne does not include a cross-platform webview widget, the implementation must select and add a Go webview dependency compatible with the supported desktop platforms. The dependency must be verified against the project's Go version and Fyne's native window lifecycle.

## 4. Dependency and platform investigation

Before implementation:

1. Identify a maintained Go webview integration that supports the project's target platforms.
2. Confirm whether it uses WebKit on macOS/Linux and WebView2 on Windows, or equivalent platform runtimes.
3. Check whether it requires additional CGO flags, system libraries, or developer packages.
4. Verify whether it can embed into an existing Fyne window rather than opening a separate top-level window.
5. Verify how local HTML, JavaScript, CSS, worker scripts, fonts, and PDF assets are loaded.
6. Confirm licensing compatibility for the dependency and bundled PDF.js distribution.
7. Prefer an existing dependency over writing a custom browser/native bridge.

If no integration can reliably embed inside the existing Fyne content area across supported platforms, document the limitation before proceeding and choose the least disruptive fallback.

## 5. PDF.js asset strategy

Use a pinned PDF.js release rather than an unversioned CDN URL. The implementation should include only the assets needed by the viewer, normally:

- Viewer HTML.
- Viewer CSS.
- Viewer JavaScript.
- PDF.js core worker and runtime files.
- Required images, locale files, and fonts if the chosen viewer build needs them.

The PDF.js version must be recorded in the project documentation. Assets should be kept reproducible, and the build should not depend on network access at runtime.

If repository policy permits checked-in vendor assets, place them in a clearly named directory such as `assets/pdfjs`. If assets are generated or copied during the build, add a deterministic Taskfile target and ensure the final application can locate them in development and packaged builds.

Avoid modifying vendored PDF.js source except where necessary for integration. Any local patch should be documented.

## 6. Local document serving and security

The viewer must not expose the user's entire filesystem to JavaScript.

Required safeguards:

- Only serve the currently selected PDF, or a narrowly scoped uploads directory.
- Reject path traversal and normalize all requested paths.
- Bind any local HTTP server to loopback only.
- Use an unpredictable per-viewer token if an HTTP endpoint is required.
- Do not accept arbitrary remote URLs from PDF metadata or user-controlled fields.
- Keep network access disabled or unnecessary for viewer assets.
- Set appropriate response content types and disposition headers.
- Shut down the local document server when the paper view or application closes.
- Avoid logging document contents or absolute user paths unnecessarily.

The PDF path already comes from Paperfolio's store. The viewer integration must still validate that the resolved path remains under the application's uploads directory before serving it.

## 7. UI behavior

### Paper with a PDF

When a paper has a PDF:

- Show the embedded PDF.js viewer in the Reader tab.
- Preserve the highlights panel or controls in the reader layout.
- Allow the user to read without leaving Paperfolio.
- Keep existing Add highlight behavior available.

### Paper without a PDF

When a paper has no PDF:

- Show the existing empty state.
- Keep the “Choose a PDF…” action.
- Do not create a viewer or local server unnecessarily.

### Replacement

After replacing a PDF:

- Close or refresh the old viewer document.
- Load the new PDF in the same Reader tab.
- Avoid leaving stale file handles, URLs, or viewer instances alive.

### Navigation

When switching papers:

- Dispose of the previous embedded viewer or navigate it to a blank local page.
- Ensure the new paper's PDF is loaded only after its path has been validated.
- Avoid retaining prior document contents in JavaScript memory longer than needed.

## 8. Code changes expected

Likely changes include:

- `go.mod` and `go.sum`: add the selected webview dependency.
- `app.go`: replace external PDF opening with an embedded reader widget and lifecycle handling.
- A new focused Go file, if needed, for viewer URL construction, local serving, path validation, and cleanup. Prefer this over expanding `app.go` substantially.
- PDF.js viewer assets under a dedicated assets directory, or a deterministic asset-copy/build arrangement.
- `Taskfile.yml`: add viewer asset preparation or verification tasks if needed, and ensure `task build` produces a runnable application with all viewer assets.
- `.gitignore`: ignore only generated viewer/build artifacts that are not checked in.
- `README.md`: update architecture, requirements, runtime behavior, and platform setup.
- Tests for path validation, URL/token construction, and viewer lifecycle helpers where they can be tested without starting a GUI.

No database schema changes are expected.

## 9. Build and packaging changes

The build must work from a clean checkout after dependencies/assets are prepared.

The Taskfile should provide targets equivalent to:

- `task assets`: fetch or prepare the pinned PDF.js assets, if the project chooses generated assets.
- `task build`: prepare required assets, then build the native application into `./bin/paperfolio`.
- `task test`: run unit tests, including non-GUI viewer helper tests.
- `task run`: run the application with assets available.
- `task clean`: remove generated build artifacts without deleting source or vendored assets.

The final packaging approach must account for relative paths. Running from the repository root and launching an installed binary should both resolve viewer assets correctly. If the webview integration supports Go `embed`, embedding static assets is preferred because it avoids installation-directory assumptions.

## 10. Testing strategy

### Unit tests

Add tests for:

- Accepting a PDF path located inside the configured uploads directory.
- Rejecting paths outside the uploads directory.
- Rejecting traversal attempts and malformed paths.
- Generating valid viewer document URLs or bridge payloads.
- Cleaning up viewer/server resources safely and idempotently.
- Handling missing or deleted PDF files with a user-visible error path.

### Build verification

Run:

```bash
go test ./...
go build -o /tmp/paperfolio-viewer-check .
```

Also verify the project Taskfile from a clean build state:

```bash
task test
task build
```

Confirm that `./bin/paperfolio` exists after `task build` and that no required viewer asset is missing.

### Manual desktop verification

On each supported platform where the required webview runtime is available:

1. Launch Paperfolio.
2. Import a valid PDF.
3. Open the paper's Reader tab.
4. Confirm the first page renders inside the Paperfolio window.
5. Scroll through multiple pages.
6. Test zoom controls.
7. Test text search.
8. Replace the PDF and confirm the new document loads.
9. Switch between papers and confirm documents do not bleed into one another.
10. Open a paper without a PDF and confirm the empty state remains usable.
11. Delete a paper and confirm the viewer closes without a crash.
12. Quit and relaunch to verify local assets and stored PDFs still work offline.

## 11. Failure handling

The application should show a clear in-app error when:

- The selected PDF cannot be found.
- The embedded webview runtime is unavailable.
- PDF.js assets cannot be loaded.
- The document fails to parse or render.
- The local serving/bridge layer cannot be initialized.

Errors should not panic the application or silently fall back to opening arbitrary files. An optional “Open externally” fallback may be retained only if explicitly desired, and it should remain a secondary action rather than the primary reader behavior.

## 12. Risks and mitigations

### Native runtime availability

Webview integrations depend on platform browser runtimes and native libraries. Mitigate this by documenting prerequisites, selecting a maintained dependency, and testing on each supported platform.

### Fyne and webview window ownership

Embedding a native webview into a Fyne canvas may be difficult because both toolkits manage native windows differently. Validate this early with a minimal prototype before rewriting the reader tab.

### Asset packaging

PDF.js uses worker scripts and multiple related assets. Missing or incorrectly located worker files can cause blank documents. Pin and test the complete asset set, preferably using embedded assets or a controlled local handler.

### Memory usage

Large research PDFs can consume significant memory when rendered. Avoid duplicating full PDF bytes unnecessarily, release old viewer instances, and let PDF.js handle incremental page rendering where available.

### Licensing and upgrades

PDF.js and the selected webview library have separate licenses and upgrade paths. Record versions and licenses, and avoid silently updating either dependency.

## 13. Implementation milestones

### Milestone 1: Dependency spike

- Select and verify the webview integration.
- Build a minimal Fyne screen containing an embedded webview.
- Load a local HTML page and execute a basic JavaScript bridge or local URL.
- Confirm supported-platform build behavior.

### Milestone 2: PDF.js packaging

- Pin a PDF.js version.
- Add or generate viewer assets.
- Load the PDF.js viewer locally inside the webview.
- Render a known sample PDF without network access.

### Milestone 3: Paperfolio integration

- Connect the selected paper's validated local PDF path.
- Replace the external open button in the Reader tab.
- Handle missing PDFs, replacements, paper switching, and cleanup.
- Preserve highlights and existing reader controls.

### Milestone 4: Hardening

- Add path and lifecycle tests.
- Update Taskfile, README, and platform requirements.
- Run Go tests and builds.
- Perform manual desktop verification.
- Review for filesystem exposure, stale resources, and offline behavior.

## 14. V1 implementation status

- Embedded webview integration: implemented with `apptrix.org/components/widget/webview`.
- Loopback-only viewer server and selected-PDF validation: implemented.
- Bundled PDF.js runtime and worker: implemented with PDF.js 4.10.38 assets under `assets/pdfjs`.
- Full PDF.js stock viewer UI, text-layer search, and annotation tools: not yet included; the current local shell provides rendering, navigation, zoom, and a basic search affordance.
- Cross-platform manual verification: pending on each target platform's native web runtime.

## 15. Definition of done

The feature is complete when:

- `boxesandglue` is not used for PDF viewing.
- An imported PDF renders inside the Paperfolio application window.
- Basic navigation, scrolling, zoom, and search work through the bundled PDF.js viewer.
- No runtime CDN or external PDF reader is required for normal reading.
- PDF paths are restricted to Paperfolio's managed uploads area.
- Replacing, switching, and deleting papers correctly clean up viewer resources.
- `task build` writes a runnable application to `./bin/paperfolio`.
- `task test` and `go test ./...` pass.
- README requirements and platform setup instructions are accurate.
- Manual verification succeeds on the supported desktop platforms.
