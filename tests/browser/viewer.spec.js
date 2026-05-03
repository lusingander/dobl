const { execFileSync } = require("node:child_process");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { pathToFileURL } = require("node:url");
const { expect, test } = require("@playwright/test");

const repoRoot = path.resolve(__dirname, "../..");

test("static viewer loads summary JSON", async ({ page }) => {
  await page.goto(pathToFileURL(path.join(repoRoot, "examples/viewer/index.html")).href);
  await page.locator("#file-input").setInputFiles(path.join(repoRoot, "examples/viewer/sample-summary.json"));

  await expect(page.locator("#source-name")).toHaveText("sample-summary.json");
  await expect(page.locator("#metric-steps")).toHaveText("8");
  await expect(page.locator("#metric-warnings")).toHaveText("1");
  await expect(page.locator("#metric-errors")).toHaveText("1");
  await expect(page.locator("#visible-count")).toHaveText("8 shown");
  await expect(page.locator(".timeline-segment")).toHaveCount(8);
  await expect(page.locator("#detail-panel")).toContainText("[internal] load build definition from Dockerfile");
});

test("report output loads embedded summary", async ({ page }) => {
  const reportPath = path.join(os.tmpdir(), "dobl-browser-report.html");
  const html = execFileSync(
    "go",
    ["run", "./cmd/dobl", "report", "testdata/error_plain.log"],
    {
      cwd: repoRoot,
      env: {
        ...process.env,
        GOCACHE: process.env.GOCACHE || "/tmp/dobl-go-build"
      },
      encoding: "utf8"
    }
  );
  fs.writeFileSync(reportPath, html);

  await page.goto(pathToFileURL(reportPath).href);

  await expect(page.locator("#source-name")).toHaveText("testdata/error_plain.log");
  await expect(page.locator("#metric-steps")).toHaveText("3");
  await expect(page.locator("#metric-errors")).toHaveText("1");
  await expect(page.locator("#visible-count")).toHaveText("3 shown");
  await expect(page.locator(".timeline-segment")).toHaveCount(3);
  await expect(page.locator("#detail-panel")).toContainText("[internal] load build definition from Dockerfile");
});
