import { defineConfig } from 'vite';
import { resolve } from 'path';

// import.meta.dirname rather than __dirname: this package is "type": "module",
// so __dirname only resolved because Vite pre-bundled the config through
// esbuild's CJS shim on every load — which writes a temp file into the project
// root that the config watcher then sees as a change. Loading natively removes
// that round trip. (Node 20.11+; this project runs 24.)
const root = import.meta.dirname;

export default defineConfig({
  // Absolute base, and this is load-bearing rather than cosmetic. The legacy
  // pages ship as flat files served extensionless via .htaccess, and Apache
  // also answers the trailing-slash variant (/services/) from the same file.
  // Under base: './' that variant would resolve assets against /services/ and
  // 404 every one of them. Absolute paths make the slash variant irrelevant.
  // Deployment target is Hostinger/Apache, not a GitHub Pages subpath.
  base: '/',
  server: {
    watch: {
      // Everything here lives inside the project root but is not source. The
      // build output alone is 129 files, and the screen recordings are 22MB —
      // all of it fed the watcher for nothing.
      ignored: [
        '**/dist/**',
        '**/content/**',
        '**/tools/**',
        '**/*.mp4',
        '**/.git/**'
      ]
    },
    // Transform the entry graphs up front so the first request after a start
    // is not a cold transform of three.js + gsap.
    warmup: {
      clientFiles: ['./src/js/main.js', './src/js/subpage.js']
    }
  },
  optimizeDeps: {
    // Heavy and stable. Declaring them keeps the dep cache deterministic
    // instead of Vite discovering them mid-request and re-optimizing, which
    // is what leaves a page showing unstyled HTML while it waits.
    include: ['three', 'gsap', 'gsap/ScrollTrigger', 'lenis']
  },
  build: {
    rollupOptions: {
      // Only the pages built from src/ are listed here. The 52 legacy pages
      // live in public/ and are copied verbatim, which keeps Vite away from
      // free-audit.html's 2,140 lines of inline JS.
      //
      // No 'index.html' entry: dist/index.html comes from public/index.html
      // (the legacy home page). experience.html carries the immersive build
      // until Phase 5 merges it onto / — see .migration/MIGRATION-PLAN.md.
      input: {
        // The merged immersive home page. It absorbed the legacy home page's
        // content and both schema nodes, so this entry now produces dist/index.html
        // and supersedes the copy that used to come from public/index.html.
        main: resolve(root, 'index.html'),
        works: resolve(root, 'works.html'),
        cognexa: resolve(root, 'cognexa.html')
      }
    }
  }
});
