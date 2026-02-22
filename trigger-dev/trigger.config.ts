import { defineConfig } from "@trigger.dev/sdk/v3";

export default defineConfig({
  project: "proj_vfiplbbqbkdjjleodvmj", // Replaced by `make init`
  runtime: "node",
  logLevel: "log",
  retries: {
    enabledInDev: true,
    default: {
      maxAttempts: 3,
      minTimeoutInMs: 1000,
      maxTimeoutInMs: 10000,
      factor: 2,
    },
  },
  dirs: ["src/trigger"],
  maxDuration: 30000
});
