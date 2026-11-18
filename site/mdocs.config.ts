import { defineConfig } from "@standarx/mdocs/config";

export default defineConfig({
  siteName: "ghstats",
  base: "/ghstats/",
  contentDir: "../docs",
  demosDir: "demos",
  apiDir: "api",
  locales: ["en", "fr"],
  headingSlugs: "github",
  stylesheets: ["@standarx/mdocs/styles/docs.css", "./site.css"],
  categories: [
    {
      slug: "guide",
      label: "Guide",
      blurb: "Set up ghstats and configure it.",
      labels: { en: "Guide", fr: "Guide" },
      blurbs: { en: "Set up ghstats and configure it.", fr: "Installer ghstats et le configurer." },
    },
    {
      slug: "cards",
      label: "Cards",
      blurb: "Every card ghstats renders.",
      labels: { en: "Cards", fr: "Cartes" },
      blurbs: { en: "Every card ghstats renders.", fr: "Toutes les cartes que ghstats produit." },
    },
    {
      slug: "builder",
      label: "Builder",
      blurb: "Try a configuration in the browser.",
      labels: { en: "Builder", fr: "Builder" },
      blurbs: { en: "Try a configuration in the browser.", fr: "Essayer une configuration dans le navigateur." },
    },
  ],
});
