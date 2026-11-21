import { existsSync, statSync } from "node:fs";
import { join, normalize, resolve } from "node:path";

const root = resolve(process.env.GHSTATS_DIST ?? join(import.meta.dir, "..", "dist"));
const port = Number(process.env.PORT ?? 4317);
const base = "/ghstats/";

if (!existsSync(root)) throw new Error(`${root} is missing: run bun run build first`);

function locate(pathname: string): string | null {
  const relative = normalize(decodeURIComponent(pathname.slice(base.length)));
  if (relative.startsWith("..")) return null;
  const target = join(root, relative);
  if (existsSync(target) && statSync(target).isDirectory()) {
    const index = join(target, "index.html");
    return existsSync(index) ? index : null;
  }
  return existsSync(target) ? target : null;
}

Bun.serve({
  port,
  hostname: "127.0.0.1",
  fetch(request) {
    const { pathname } = new URL(request.url);
    if (!pathname.startsWith(base)) return new Response("not found", { status: 404 });
    const file = locate(pathname);
    if (file === null) return new Response("not found", { status: 404 });
    return new Response(Bun.file(file));
  },
});

console.log(`serving ${root} at http://127.0.0.1:${port}${base}`);
