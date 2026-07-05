import { build, $ } from 'bun';
import { existsSync, rmSync, readFileSync, writeFileSync } from 'fs';
import { SveltePlugin } from 'bun-plugin-svelte';

if (existsSync('dist')) {
  rmSync('dist', { recursive: true });
}
await $`mkdir -p dist`;

console.log('building CSS...');
await $`npx tailwindcss -i src/app.css -o dist/app.css --minify`;

// Skeleton generates :root [data-theme=wintry] (descendant selector)
// which doesn't match <html data-theme="wintry"> since html IS :root.
// Fix: change to [data-theme=wintry] so it matches the html element directly.
let css = readFileSync('dist/app.css', 'utf-8');
css = css.replace(/:root \[data-theme=wintry\]/g, '[data-theme=wintry]');
writeFileSync('dist/app.css', css);

console.log('bundling JS...');
await build({
  entrypoints: ['./src/main.js'],
  outdir: './dist',
  target: 'browser',
  minify: true,
  plugins: [SveltePlugin()],
});

console.log('generating index.html...');
await Bun.write(
  'dist/index.html',
  `<!doctype html>
<html lang="en" class="dark" data-theme="wintry">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Quant Dashboard</title>
  <link rel="stylesheet" href="./app.css" />
</head>
<body class="bg-surface-50 dark:bg-surface-900 text-surface-900 dark:text-surface-50 antialiased">
  <div id="app"></div>
  <script type="module" src="./main.js"></script>
</body>
</html>`,
);

console.log('build complete');
