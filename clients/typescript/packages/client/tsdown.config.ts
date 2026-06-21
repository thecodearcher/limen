import { defineConfig } from "tsdown";

export default defineConfig({
  format: ["esm"],
  dts: true,
  clean: true,
  treeshake: true,
  unbundle: true,
  entry: [
    "src/index.ts",
    "src/react/index.ts",
    "src/vue/index.ts",
    "src/svelte/index.ts",
    "src/solid/index.ts",
    "src/plugins/index.ts",
    "src/plugins/*/index.ts",
  ],
});
