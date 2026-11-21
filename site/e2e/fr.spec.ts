import { expect, test } from "@playwright/test";
import { element, image, svgOf } from "./helpers";

test("m. the builder UI is French on the French page while the cards stay English", async ({ page, baseURL }) => {
  await page.emulateMedia({ reducedMotion: "reduce", colorScheme: "dark" });
  await page.goto(new URL("/ghstats/fr/builder/", baseURL).href);
  await expect(page.getByTestId("builder")).toBeVisible();
  await expect(element(page, "stats")).toBeVisible();
  await expect(page.locator("html")).toHaveAttribute("lang", "fr");

  await expect(page.getByTestId("section-theme").locator("summary")).toHaveText("Thème");
  await expect(page.getByTestId("section-cards").locator("summary")).toHaveText("Cartes");
  await expect(page.getByTestId("section-links").locator("summary")).toHaveText("Liens");
  await expect(page.getByTestId("tab-preview")).toHaveText("Aperçu");
  await expect(page.getByTestId("tab-yaml")).toHaveText("ghstats.yaml");
  await expect(page.getByTestId("mode-dark")).toHaveText("Sombre");
  await expect(page.getByTestId("data-source")).toContainText("données d'exemple");
  await expect(page.getByTestId("download-zip")).toHaveText("Tout télécharger (.zip)");
  await expect(page.getByTestId("selection-bar")).toContainText("Cliquez sur un élément");
  await expect(page.getByTestId("drop-newrow").first()).toContainText("Nouvelle ligne");

  await element(page, "stats").click();
  await expect(page.getByTestId("section-selection")).toBeVisible();
  await expect(page.getByTestId("section-selection")).toContainText("Carte stats");
  await expect(page.getByTestId("deselect")).toHaveText("Terminé");
  await expect(page.getByTestId("download-svg")).toHaveText("Télécharger le SVG");
  await expect(page.getByTestId("download-png")).toHaveText("Télécharger le PNG");
  await expect(page.getByTestId("selection-bar").getByRole("button", { name: "Désélectionner" })).toBeVisible();

  await page.getByTestId("tab-snippet").click();
  await expect(page.getByTestId("output-README.snippet.md")).toContainText("Collez-le dans le README de votre profil");
  await page.getByTestId("tab-preview").click();

  expect(await svgOf(image(page, "stats"))).toContain("<svg");
});
