export type Yaml = string | number | boolean | null | Yaml[] | { [key: string]: Yaml };

interface Line {
  indent: number;
  text: string;
}

function scalar(text: string): Yaml {
  if (text === "[]") return [];
  if (text === "{}") return {};
  if (text === "null" || text === "~") return null;
  if (text === "true") return true;
  if (text === "false") return false;
  if (text.length >= 2 && text.startsWith("'") && text.endsWith("'")) return text.slice(1, -1).replaceAll("''", "'");
  if (text.startsWith('"')) return JSON.parse(text) as string;
  if (/^-?\d+(\.\d+)?$/.test(text)) return Number(text);
  return text;
}

function splitKey(text: string): [string, string] | null {
  const match = /^([^'"\s-][^:]*|-[^\s:][^:]*):(?: (.*))?$/.exec(text);
  if (match === null) return null;
  return [match[1] ?? "", (match[2] ?? "").trim()];
}

function isItem(line: Line | undefined, indent: number): line is Line {
  return line !== undefined && line.indent === indent && (line.text === "-" || line.text.startsWith("- "));
}

function block(lines: Line[], start: number, indent: number): [Yaml, number] {
  return isItem(lines[start], indent) ? sequence(lines, start, indent) : mapping(lines, start, indent);
}

function sequence(lines: Line[], start: number, indent: number): [Yaml, number] {
  const items: Yaml[] = [];
  let i = start;
  for (;;) {
    const line = lines[i];
    if (!isItem(line, indent)) break;
    const rest = line.text.slice(1).trim();
    if (rest === "") {
      const next = lines[i + 1];
      if (next !== undefined && next.indent > indent) {
        const [value, after] = block(lines, i + 1, next.indent);
        items.push(value);
        i = after;
      } else {
        items.push(null);
        i += 1;
      }
    } else if (splitKey(rest) !== null) {
      lines[i] = { indent: indent + 2, text: rest };
      const [value, after] = mapping(lines, i, indent + 2);
      items.push(value);
      i = after;
    } else {
      items.push(scalar(rest));
      i += 1;
    }
  }
  return [items, i];
}

function mapping(lines: Line[], start: number, indent: number): [Yaml, number] {
  const out: { [key: string]: Yaml } = {};
  let i = start;
  for (;;) {
    const line = lines[i];
    if (line === undefined || line.indent !== indent) break;
    const pair = splitKey(line.text);
    if (pair === null) throw new Error(`unreadable YAML line: ${line.text}`);
    const [key, rest] = pair;
    if (rest !== "") {
      out[key] = scalar(rest);
      i += 1;
      continue;
    }
    const next = lines[i + 1];
    if (next !== undefined && next.indent > indent) {
      const [value, after] = block(lines, i + 1, next.indent);
      out[key] = value;
      i = after;
    } else {
      out[key] = null;
      i += 1;
    }
  }
  return [out, i];
}

export function parseYaml(source: string): Yaml {
  const lines: Line[] = source
    .split("\n")
    .filter((raw) => raw.trim() !== "" && !raw.trimStart().startsWith("#"))
    .map((raw) => ({ indent: raw.length - raw.trimStart().length, text: raw.trim() }));
  const [value] = block(lines, 0, lines[0]?.indent ?? 0);
  return value;
}
