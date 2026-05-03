const { defineConfig } = require("@playwright/test");

module.exports = defineConfig({
  testDir: "./tests/browser",
  reporter: "list",
  use: {
    browserName: "chromium"
  }
});
