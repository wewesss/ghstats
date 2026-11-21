import { t } from "./i18n";

const LOGIN = /^[A-Za-z0-9-]{1,39}$/;
const PATH = /^[A-Za-z0-9._/-]*$/;

export const DEFAULT_PATH = "ghstats/";

export interface DataSource {
  label: string;
  report: string | undefined;
}

export const SAMPLE: DataSource = {
  get label() {
    return t("sampleData");
  },
  report: undefined,
};

export function statsUrl(login: string, path: string): string | Error {
  if (!LOGIN.test(login)) return new Error(t("invalidLogin"));
  if (!PATH.test(path) || path.includes("..")) return new Error(t("invalidPath"));
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
    throw new Error(t("unreachable", { url }));
  }
  if (response.status === 404) {
    throw new Error(t("notFound", { url }));
  }
  if (!response.ok) throw new Error(t("loadFailed", { url, status: response.status }));
  return { label: `${login} (GitHub)`, report: await response.text() };
}

export async function loadFromFile(file: File): Promise<DataSource> {
  return { label: file.name, report: await file.text() };
}
