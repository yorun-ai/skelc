"use strict";

const fs = require("fs");
const path = require("path");

const root = path.resolve(__dirname, "..");
const jsonFiles = [
  "package.json",
  "language-configuration.json",
  "syntaxes/skel.tmLanguage.json",
  "themes/skel-dark-color-theme.json"
];

for (const file of jsonFiles) {
  JSON.parse(fs.readFileSync(path.join(root, file), "utf8"));
}

const extensionSource = fs.readFileSync(path.join(root, "src/extension.js"), "utf8");
new Function("require", "module", "exports", extensionSource);

console.log("vscode-skel files are syntactically valid");
