import { cpSync, existsSync, readdirSync } from "node:fs";
import { join, resolve } from "node:path";

const site = resolve(import.meta.dir, "..");
const dist = join(site, "dist");
const publicDir = join(site, "public");

if (!existsSync(dist)) throw new Error("dist/ is missing: run mdocs build first");

for (const name of readdirSync(publicDir)) cpSync(join(publicDir, name), join(dist, name), { recursive: true });
cpSync(join(site, "..", "showcase"), join(dist, "showcase"), { recursive: true });
