import { chmodSync, copyFileSync, existsSync, mkdirSync } from "node:fs";
import { join, resolve } from "node:path";

const site = resolve(import.meta.dir, "..");
const root = resolve(site, "..");
const out = join(site, "public");
mkdirSync(out, { recursive: true });

async function run(cmd: string[], cwd: string, env: Record<string, string> = {}): Promise<string> {
  const proc = Bun.spawn(cmd, {
    cwd,
    env: { ...process.env, ...env },
    stdout: "pipe",
    stderr: "inherit",
  });
  const text = await new Response(proc.stdout).text();
  if ((await proc.exited) !== 0) throw new Error(`${cmd.join(" ")} failed`);
  return text.trim();
}

await run(
  ["go", "build", "-trimpath", "-ldflags=-s -w", "-o", join(out, "ghstats.wasm"), "./cmd/wasm"],
  root,
  { GOOS: "js", GOARCH: "wasm" },
);

const goroot = await run(["go", "env", "GOROOT"], root);
const exec = [join(goroot, "lib", "wasm", "wasm_exec.js"), join(goroot, "misc", "wasm", "wasm_exec.js")].find(existsSync);
if (exec === undefined) throw new Error(`wasm_exec.js not found under ${goroot}`);
const dest = join(out, "wasm_exec.js");
// The Go module cache is read-only and copyFileSync keeps that mode, so the next build could not overwrite the copy.
if (existsSync(dest)) chmodSync(dest, 0o644);
copyFileSync(exec, dest);
chmodSync(dest, 0o644);
