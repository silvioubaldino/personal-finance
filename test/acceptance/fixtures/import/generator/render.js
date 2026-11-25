// Renderiza um HTML estático em PDF (Chromium via Playwright).
// Uso: node render.js in.html out.pdf
'use strict';

const path = require('path');
const { pathToFileURL } = require('url');
const { chromium } = require('playwright');

async function main() {
  const [input, output] = process.argv.slice(2);
  if (!input || !output) {
    console.error('uso: node render.js in.html out.pdf');
    process.exit(2);
  }

  const browser = await chromium.launch();
  try {
    const page = await browser.newPage();
    await page.goto(pathToFileURL(path.resolve(input)).href, { waitUntil: 'load' });
    await page.evaluate(() => document.fonts.ready);
    // 446pt x 632pt (o Playwright não aceita 'pt'; 1pt = 1/72in).
    await page.pdf({
      path: path.resolve(output),
      width: `${446 / 72}in`,
      height: `${632 / 72}in`,
      printBackground: true,
    });
  } finally {
    await browser.close();
  }
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
