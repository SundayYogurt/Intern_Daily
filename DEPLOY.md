# Deploy

Build the frontend with `npm ci` and `npm run build` in `web`. Build the API with `go build -o internship-journal .` in `server` on the deployment OS. The bundled Windows exe cannot run on Linux.

Run the Go server behind an HTTPS reverse proxy. The same server serves the frontend and `/api`; there are no required frontend environment variables.

| Variable | Production value | Purpose |
| --- | --- | --- |
| `ADDR` | `0.0.0.0:8080` | Listen address; match the platform's port |
| `DATA_DIR` | `/var/lib/internship-journal` | Writable persistent volume for SQLite; survives deploys |
| `WEB_DIST` | `/app/web/dist` | Absolute path to the built frontend |
| `APP_SECURE_COOKIE` | `1` | Send login cookies only over HTTPS; use `0` for local HTTP |
| `PDF_BROWSER` | `/usr/bin/chromium` | Actual installed Chrome/Chromium executable for PDF generation |

The `.env.example` file documents values only: configure them in your hosting platform or process manager. The application does not automatically read `.env`. There is no `DATABASE_URL` or required API key. Run one instance with a persistent disk; this SQLite app is not configured for shared storage across replicas. Back up the database regularly. Install Chromium and its system dependencies and run the application as a user that can launch it; verify PDF export on the target host.

Use the public HTTPS domain when copying mentor invitations. `127.0.0.1` links only work on the sender's computer. Keep the frontend and API on the same origin. Forward the public Host header through the reverse proxy.

UI redesign guidance: [Unslop by Damir Sokolovsky](https://github.com/heydamir/unslop), CC BY 4.0.
