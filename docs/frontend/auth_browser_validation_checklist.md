# SignalOps Auth Browser Validation Checklist

Gate: G054
Audience: operator / frontend-agent
Date: 2026-07-09

Real-browser validation of the Syncratic IdP login/logout flow for the SignalOps
web app. This is the step deferred from G053/G054: automated/headless validation
is **blocked by the Imperva/Incapsula WAF** fronting the IdP auth endpoint
(`auth.syncratic.co`), so a real browser session with operator credentials is
required. The OIDC discovery document is reachable headlessly, but the
authorization endpoint is not.

## Preconditions

- Backend remains `SIGNALOPS_AUTH_ENABLED=false` for this validation (the frontend
  flow is validated with the backend not yet enforcing tokens).
- A real browser (Chrome/Firefox/Safari) and the initial operator account
  `lukeb` (member of `/signalops/admins`).

## 1. Build the auth-enabled frontend image

```bash
VITE_SIGNALOPS_AUTH_ENABLED=true \
VITE_SIGNALOPS_AUTH_ISSUER=https://auth.syncratic.co/realms/syncratic \
VITE_SIGNALOPS_AUTH_REALM=syncratic \
VITE_SIGNALOPS_AUTH_CLIENT_ID=signalops-web \
VITE_SIGNALOPS_AUTH_AUDIENCE=signalops-api \
docker compose -f compose.yaml -f compose.traefik.yaml build web
```

> The Traefik overlay (`-f compose.traefik.yaml`) is required: it attaches the
> `web` service to the Traefik network and adds the `Host(signalops.syncratic.io)`
> router labels. A plain `docker compose build/up web` strips those labels and
> takes the public site offline (404).

## 2. Redeploy the frontend for validation (backend still disabled)

```bash
docker compose -f compose.yaml -f compose.traefik.yaml up -d web
```

Only the `web` service is redeployed; the gateway keeps `SIGNALOPS_AUTH_ENABLED=false`.

## 3. Real-browser checklist

Open `https://signalops.syncratic.io` in a real browser and confirm each step.
Record the result (pass/fail + notes) for the audit.

- [ ] Unauthenticated user sees the compact SignalOps **Sign in** screen (brand
      mark + single primary action; no data loads).
- [ ] Clicking **Sign in** redirects to `auth.syncratic.co` (Syncratic IdP).
- [ ] Signing in as `lukeb` returns to SignalOps at the originally requested path
      (or `/`).
- [ ] The app shell shows the operator identity (`lukeb`, else email, else sub).
- [ ] The tenant resolves to `tenant-local` (dashboard/queries scoped to it).
- [ ] Role-derived lifecycle permissions are admin-capable (Acknowledge/Resolve/
      Suppress on Alerts and Review/Dismiss/Archive on Insights are enabled).
- [ ] In browser devtools (Network), `/v1/*` requests include
      `Authorization: Bearer ...`.
- [ ] Dashboard data loads (runs, raw events, provider usage, catalogs, signals,
      alerts, insights).
- [ ] The dashboard stream does **not** show a persistent auth-related error under
      the auth REST fallback (expect neutral **REST refresh** wording, not
      "reconnecting"/"disconnected").
- [ ] Health indicator reflects gateway health (green when healthy), not a stream
      alarm.
- [ ] **Sign out** redirects through the IdP logout flow, clears the session and
      query cache, and returns to the Sign in screen.
- [ ] Leave the app idle longer than `VITE_SIGNALOPS_AUTH_IDLE_TIMEOUT_MINUTES`; the
      next activity requires a new sign-in and no background renewal extends that
      idle period.
- [ ] Keep the app active across an access-token expiry; it continues without a
      login interruption (verify the IdP allows `/auth/silent-renew` when a
      refresh token is unavailable).
- [ ] After sign out, `/healthz`/`/readyz` remain reachable and no protected
      `/v1/*` request fires before the next sign in.

## 4. Restore the auth-disabled frontend image

Unless backend auth enablement is being coordinated immediately, restore the
default auth-disabled deployment:

```bash
docker compose -f compose.yaml -f compose.traefik.yaml build web
docker compose -f compose.yaml -f compose.traefik.yaml up -d web
```

Then confirm `https://signalops.syncratic.io/healthz` returns 200 and the
dashboard loads without login.

## Notes

- Headless automation (curl, Playwright without a real browser fingerprint, etc.)
  is blocked by the Imperva WAF on the IdP authorization endpoint — do not treat a
  scripted 403 as a client/redirect misconfiguration.
- If the public site 404s after a redeploy, the Traefik overlay was omitted; redeploy
  with both compose files (see step 2).
- Permanent `SIGNALOPS_AUTH_ENABLED=true` backend enablement is a separate,
  coordinated operator/Codex step and is **out of scope** for this checklist.
