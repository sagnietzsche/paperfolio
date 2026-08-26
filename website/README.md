# Paperfolio website

This directory contains Paperfolio's static landing page. It is a React + Vite
site with no backend. The desktop application it documents is built with Go and
Fyne; it is not a Tauri or browser application.

```bash
npm install
npm run dev
npm run build
```

## Release artifacts

Build the desktop executable from the repository root:

```bash
go build -o paperfolio .
```

Package the executable using the conventions of the target operating system,
then attach that artifact to the project's GitHub release. The landing page's
download links point to the latest release.

## Data and privacy

Paperfolio stores its SQLite database, imported PDFs, and markdown note mirrors
locally in `~/Documents/Paperfolio_Data/`. It has no account, backend, sync
service, or hosted data dependency.
