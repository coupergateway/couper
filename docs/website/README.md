# Couper Documentation Website

Built with [Nuxt 3](https://v3.nuxtjs.org).

## Setup

Make sure to install the dependencies:

```bash
# npm
npm install
```

## Generate Reference and Update Search Index

```bash
(cd ../.. && make generate-docs)
```

## Development Server

Start the development server on http://localhost:3000:

```bash
npm run dev
```

## Production

Build the application for production:

```bash
npm run build
```

Preview the production build locally:

```bash
npm run generate && couper run -f couper.hcl
```
