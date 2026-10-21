# Infinite Canvas integration

Upstream: https://github.com/basketikun/infinite-canvas (MIT).
Pinned revision: `9f2bd9f1b27c45fd1b8f5323072b07243f9e0fac`.

The main frontend build verifies canvas asset hashes and rebuilds stale assets.
Install Bun 1.3.14 and Git for the first build. To rebuild explicitly, run
`node integrations/infinite-canvas/build.mjs` before the main frontend build.
The script prepares the upstream source under `.dev/`, applies these overlays,
and builds `frontend/public/canvas-app`. The production frontend embeds that
directory in the existing application image. No additional production container,
database migration, or upstream account credential is needed.

The authenticated `/canvas` host loads the user's own keys and model catalog.
The iframe starts only after a same-origin message from its parent, and receives
only the selected user API key, never the login token or an administrator key.
The API key stays in memory; upstream config persistence strips channel keys.
Local storage and IndexedDB are scoped by user ID. This is browser-local storage,
not server-enforced storage isolation or a cloud gallery.

The embedded build exposes canvases and assets. Remote plugins, third-party
channel setup, telemetry, and local Agent initialization are not enabled by the
host. Image calls continue through Sub2API's existing routing, access controls,
usage records, and billing. One image per generation is the initial default.
Preserve the upstream MIT notice and attribution when redistributing.

The workspace uses a persistent bottom composer: enter a prompt and generate
without creating an empty node first. Select an image's **Use as reference** action
or choose multiple images in the reference picker. Generation creates a separate
result node and connects each reference automatically, preserving the originals.
The model, aspect ratio and advanced settings stay in the composer; generation
still uses the upstream request and storage pipeline. Duplicate submissions are
locked while a request runs, and stopping is a separate action. Failed cards show
the upstream error and retain the existing retry action.

`simple-workspace.tsx`, `workspace.css` and `apply-workspace.mjs` implement this
layout and behavior as overlays on the pinned source. The elements drawer starts
closed; advanced text/config node editors temporarily replace the bottom composer.
Prompt/settings/reference drafts are stored separately for each canvas in the
user-scoped local storage. The host sidebar starts compact only on this page,
without changing the user's navigation preference elsewhere. Appearance changes
follow the host without reloading the active canvas.

`canvas-theme.ts` and `app-theme.ts` share Sub2API's neutral surfaces and teal
accent across canvas rendering and Ant Design controls. `embedded.css` styles
the workspace navigation, tools, and prompt panel through scoped data hooks.
Keep visual customizations here so rebuilding the pinned upstream preserves them.

The production entry is `https://sub.gpt-buss.xyz/canvas`. The temporary
`image.gpt-buss.xyz` proxy entry has been removed. The canvas document permits
`connect-src blob:` so connected nodes can read browser-local reference images;
other pages retain their existing CSP.
