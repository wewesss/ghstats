const LOGIN = /^[A-Za-z0-9-]{1,39}$/;
const PATH = /^[A-Za-z0-9._/-]*$/;

export const DEFAULT_PATH = "ghstats/";

export interface DataSource {
  label: string;
  report: string | undefined;
}

export const SAMPLE: DataSource = { label: "sample data", report: undefined };

export function statsUrl(login: string, path: string): string | Error {
  if (!LOGIN.test(login)) return new Error("A GitHub login has 1 to 39 letters, digits or hyphens.");
  if (!PATH.test(path) || path.includes("..")) return new Error("The path may only hold letters, digits, dots, hyphens, underscores and slashes, and no \"..\".");
  const folder = path.replace(/^\/+/, "");
  const dir = folder === "" || folder.endsWith("/") ? folder : `${folder}/`;
  return `https://raw.githubusercontent.com/${login}/${login}/HEAD/${dir}stats.json`;
}

export async function loadFromGithub(login: string, path: string): Promise<DataSource> {
  const url = statsUrl(login, path);
  if (url instanceof Error) throw url;
  let response: Response;
  try {
    response = await fetch(url);
  } catch {
    throw new Error(`Could not reach ${url}.`);
  }
  if (response.status === 404) {
    throw new Error(`No stats.json at ${url}: run the ghstats workflow once, or open the file below.`);
  }
  if (!response.ok) throw new Error(`Could not load ${url}: ${response.status}`);
  return { label: `${login} (GitHub)`, report: await response.text() };
}

export async function loadFromFile(file: File): Promise<DataSource> {
  return { label: file.name, report: await file.text() };
}
