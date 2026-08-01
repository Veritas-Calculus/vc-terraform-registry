# Veritas Registry - Frontend

The web UI for Veritas Registry, built with React 19, Vite 8, Tailwind CSS 4, and
React Router 8.

## Development

```bash
npm install
npm run dev      # start the Vite dev server (http://localhost:5173)
npm run lint     # run ESLint
npm run build    # production build into dist/
npm run preview  # preview the production build
```

The dev server expects the backend API. Set `VITE_API_BASE_URL` to point at it
(defaults to `http://localhost:8080`). `VITE_REGISTRY_HOST` overrides the host
shown in generated Terraform CLI snippets (defaults to the current page host).

## Project layout

```
src/
  components/   shared components (layout, search, panels)
  contexts/     React contexts (authentication)
  pages/        route pages (home, providers, modules, mirror, docs, login)
  services/     API client
public/
  brand/        logo assets (mark.svg, icon.svg)
  favicon.svg   browser tab icon
```

See the repository [README](../README.md) and [docs](../docs) for deployment and
configuration.
