import { defineConfig } from "astro/config";
import tailwindcss from "@tailwindcss/vite";
import sitemap from "@astrojs/sitemap";
import mdx from "@astrojs/mdx";
import {
  transformerNotationDiff,
  transformerNotationFocus,
  transformerMetaHighlight,
} from "@shikijs/transformers";

// https://astro.build/config
export default defineConfig({
  vite: {
    plugins: [tailwindcss()],
  },
  markdown: {
    drafts: true,
    syntaxHighlight: "shiki",
    shikiConfig: {
      theme: "css-variables",
      transformers: [
        transformerNotationDiff({
          // Make sure these match your CSS classes
          classLineAdd: "diff add",
          classLineRemove: "diff remove",
        }),
        transformerNotationFocus({
          classActiveLine: "focused",
        }),
        transformerMetaHighlight(),
      ],
      wrap: false,
    },
  },
  site: "https://httphatch.com",
  integrations: [
    sitemap(),
    mdx({
      syntaxHighlight: "shiki",
      shikiConfig: {
        theme: "css-variables",
        transformers: [
          transformerNotationDiff({
            classLineAdd: "diff add",
            classLineRemove: "diff remove",
          }),
          transformerNotationFocus({
            classActiveLine: "focused",
          }),
          transformerMetaHighlight(),
        ],
        wrap: true,
      },
    }),
  ],
});
