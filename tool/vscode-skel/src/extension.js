"use strict";

const fs = require("fs");
const path = require("path");
const vscode = require("vscode");

const languageSelector = { language: "skel", scheme: "file" };

const builtins = new Set([
  "for",
  "as",
  "action",
  "all",
  "any",
  "auth",
  "binary",
  "bool",
  "check",
  "config",
  "credential",
  "data",
  "decimal",
  "domain",
  "duration",
  "enum",
  "event",
  "float",
  "import",
  "info",
  "input",
  "int",
  "json",
  "list",
  "localdate",
  "localdatetime",
  "localtime",
  "map",
  "method",
  "output",
  "payload",
  "permission",
  "permissioncode",
  "pub",
  "require",
  "resource",
  "service",
  "string",
  "task",
  "timestamp",
  "trigger",
  "uuid",
  "via",
  "web"
]);

function activate(context) {
  const provider = new SkelLanguageProvider();
  context.subscriptions.push(vscode.languages.registerDefinitionProvider(languageSelector, provider));
  context.subscriptions.push(vscode.languages.registerReferenceProvider(languageSelector, provider));
}

function deactivate() {}

class SkelLanguageProvider {
  async provideDefinition(document, position) {
    const context = await getSymbolContext(document, position);
    if (!context) {
      return undefined;
    }

    return resolveReference(context.index, document.uri, context.reference);
  }

  async provideReferences(document, position, referenceContext) {
    const context = await getSymbolContext(document, position);
    if (!context) {
      return undefined;
    }

    const definitions = definitionTargetsForContext(context.index, document.uri, context.reference, context.wordRange);
    if (!definitions.length) {
      return undefined;
    }

    const includeDeclaration = referenceContext ? referenceContext.includeDeclaration : true;
    const locations = [];
    const seen = new Set();
    for (const definition of definitions) {
      for (const location of referencesForDefinition(context.index, definition, includeDeclaration)) {
        const key = `${location.uri.toString()}:${location.range.start.line}:${location.range.start.character}`;
        if (!seen.has(key)) {
          seen.add(key);
          locations.push(location);
        }
      }
    }

    return locations.length ? locations : undefined;
  }
}

async function getSymbolContext(document, position) {
  const wordRange = document.getWordRangeAtPosition(position, /[A-Za-z_][A-Za-z0-9_]*/);
  if (!wordRange) {
    return undefined;
  }

  const word = document.getText(wordRange);
  if (builtins.has(word.toLowerCase())) {
    return undefined;
  }

  const workspaceFolder = vscode.workspace.getWorkspaceFolder(document.uri);
  if (!workspaceFolder) {
    return undefined;
  }

  const reference = getReferenceAtPosition(document, wordRange, word);
  const index = await buildIndex(workspaceFolder, document);
  return { index, reference, wordRange };
}

function getReferenceAtPosition(document, wordRange, word) {
  const lineText = document.lineAt(wordRange.start.line).text;
  const start = wordRange.start.character;
  const end = wordRange.end.character;
  let qualifier = "";

  if (start > 1 && lineText[start - 1] === ".") {
    const before = lineText.slice(0, start - 1);
    const match = before.match(/([A-Za-z_][A-Za-z0-9_]*)$/);
    if (match) {
      qualifier = match[1];
    }
  }

  if (!qualifier && end < lineText.length && lineText[end] === ".") {
    return { name: word, qualifier: "", aliasOnly: true };
  }

  return { name: word, qualifier, aliasOnly: false };
}

async function buildIndex(workspaceFolder, currentDocument) {
  const files = await vscode.workspace.findFiles(
    new vscode.RelativePattern(workspaceFolder, "**/*.skel"),
    new vscode.RelativePattern(workspaceFolder, "**/{.git,node_modules}/**")
  );

  const docs = new Map();
  for (const uri of files) {
    docs.set(uri.toString(), loadDocumentText(uri, currentDocument));
  }

  if (currentDocument.uri.scheme === "file" && currentDocument.languageId === "skel") {
    docs.set(currentDocument.uri.toString(), currentDocument.getText());
  }

  const index = {
    byName: new Map(),
    byQualifiedName: new Map(),
    byUri: new Map(),
    files: new Map()
  };

  for (const [uriString, text] of docs) {
    const uri = vscode.Uri.parse(uriString);
    const parsed = parseSkelFile(uri, text);
    index.files.set(uriString, parsed);

    for (const def of parsed.definitions) {
      addDefinition(index.byName, def.name, def);
      addDefinition(index.byUri, uriString, def);
      if (parsed.domain) {
        addDefinition(index.byQualifiedName, `${parsed.domain}.${def.name}`, def);
      }
    }
  }

  return index;
}

function loadDocumentText(uri, currentDocument) {
  if (uri.toString() === currentDocument.uri.toString()) {
    return currentDocument.getText();
  }

  const openDocument = vscode.workspace.textDocuments.find((doc) => doc.uri.toString() === uri.toString());
  if (openDocument) {
    return openDocument.getText();
  }

  return fs.readFileSync(uri.fsPath, "utf8");
}

function parseSkelFile(uri, text) {
  const parsed = {
    uri,
    text,
    domain: "",
    imports: new Map(),
    definitions: []
  };

  const domainMatch = /^\s*domain\s+([A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)*)/m.exec(text);
  if (domainMatch) {
    parsed.domain = domainMatch[1];
  }

  const importPattern = /^\s*import\s+([A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)*)(?:\s+as\s+([A-Za-z_][A-Za-z0-9_]*))?/gm;
  for (const match of text.matchAll(importPattern)) {
    const domainName = match[1];
    const alias = match[2] || defaultImportAlias(domainName);
    parsed.imports.set(alias, domainName);
  }

  const entryPattern = /^\s*(?:pub\s+)?(enum|data|config|actor|resource|service|web|event|task)\s+([A-Za-z_][A-Za-z0-9_]*)/gm;
  for (const match of text.matchAll(entryPattern)) {
    const kind = match[1];
    const name = match[2];
    const nameOffset = match.index + match[0].lastIndexOf(name);
    const position = positionAt(text, nameOffset);
    parsed.definitions.push({
      name,
      kind,
      domain: parsed.domain,
      dir: dirname(uri),
      qualifiedName: parsed.domain ? `${parsed.domain}.${name}` : name,
      uri,
      range: new vscode.Range(position, position.translate(0, name.length))
    });
  }

  return parsed;
}

function resolveReference(index, sourceUri, reference) {
  if (reference.aliasOnly) {
    const sourceFile = index.files.get(sourceUri.toString());
    if (!sourceFile) {
      return undefined;
    }
    const domainName = sourceFile.imports.get(reference.name);
    return domainName ? domainLocations(index, domainName) : undefined;
  }

  if (reference.qualifier) {
    const sourceFile = index.files.get(sourceUri.toString());
    const importedDomain = sourceFile && sourceFile.imports.get(reference.qualifier);
    const qualifiedName = `${importedDomain || reference.qualifier}.${reference.name}`;
    return locationsFor(index.byQualifiedName.get(qualifiedName));
  }

  return locationsFor(preferSourceDirectory(index.byName.get(reference.name) || [], sourceUri));
}

function definitionTargetsForContext(index, sourceUri, reference, wordRange) {
  const definitionsAtPosition = definitionsAtRange(index, sourceUri, wordRange);
  if (definitionsAtPosition.length) {
    return definitionsAtPosition;
  }

  if (reference.aliasOnly) {
    return [];
  }

  if (reference.qualifier) {
    const sourceFile = index.files.get(sourceUri.toString());
    const importedDomain = sourceFile && sourceFile.imports.get(reference.qualifier);
    const qualifiedName = `${importedDomain || reference.qualifier}.${reference.name}`;
    return index.byQualifiedName.get(qualifiedName) || [];
  }

  return preferSourceDirectory(index.byName.get(reference.name) || [], sourceUri);
}

function definitionsAtRange(index, uri, range) {
  const definitions = index.byUri.get(uri.toString()) || [];
  return definitions.filter((definition) => rangesEqual(definition.range, range));
}

function preferSourceDirectory(definitions, sourceUri) {
  const sourceDir = dirname(sourceUri);
  const sameDirectory = definitions.filter((definition) => definition.dir === sourceDir);
  return sameDirectory.length ? sameDirectory : definitions;
}

function dirname(uri) {
  return path.dirname(uri.fsPath);
}

function referencesForDefinition(index, definition, includeDeclaration) {
  const locations = [];
  for (const file of index.files.values()) {
    const maskedText = maskCommentsAndStrings(file.text);
    if (file.domain === definition.domain && dirname(file.uri) === definition.dir) {
      locations.push(...unqualifiedReferenceLocations(file.uri, maskedText, definition.name));
    }

    for (const [alias, domainName] of file.imports) {
      if (domainName === definition.domain) {
        locations.push(...qualifiedReferenceLocations(file.uri, maskedText, alias, definition.name));
      }
    }
  }

  if (includeDeclaration) {
    return locations;
  }

  return locations.filter((location) => {
    return location.uri.toString() !== definition.uri.toString() || !rangesEqual(location.range, definition.range);
  });
}

function unqualifiedReferenceLocations(uri, text, name) {
  const locations = [];
  const pattern = new RegExp(`\\b${escapeRegExp(name)}\\b`, "g");
  for (const match of text.matchAll(pattern)) {
    const startOffset = match.index;
    const endOffset = startOffset + name.length;
    if (text[startOffset - 1] === "." || text[endOffset] === ".") {
      continue;
    }
    locations.push(locationAt(uri, text, startOffset, name.length));
  }
  return locations;
}

function qualifiedReferenceLocations(uri, text, qualifier, name) {
  const locations = [];
  const pattern = new RegExp(`\\b${escapeRegExp(qualifier)}\\.${escapeRegExp(name)}\\b`, "g");
  for (const match of text.matchAll(pattern)) {
    const nameOffset = match.index + qualifier.length + 1;
    locations.push(locationAt(uri, text, nameOffset, name.length));
  }
  return locations;
}

function locationAt(uri, text, offset, length) {
  const start = positionAt(text, offset);
  return new vscode.Location(uri, new vscode.Range(start, start.translate(0, length)));
}

function domainLocations(index, domainName) {
  const locations = [];
  for (const file of index.files.values()) {
    if (file.domain === domainName) {
      locations.push(new vscode.Location(file.uri, new vscode.Position(0, 0)));
    }
  }
  return locations.length ? locations : undefined;
}

function rangesEqual(left, right) {
  return left.start.line === right.start.line &&
    left.start.character === right.start.character &&
    left.end.line === right.end.line &&
    left.end.character === right.end.character;
}

function maskCommentsAndStrings(text) {
  return text.replace(/\/\/[^\n\r]*|\/\*[\s\S]*?\*\/|"""[\s\S]*?"""|"([^"\\]|\\.)*"/g, (match) => {
    return match.replace(/[^\n\r]/g, " ");
  });
}

function escapeRegExp(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

function addDefinition(map, key, def) {
  const defs = map.get(key) || [];
  defs.push(def);
  map.set(key, defs);
}

function locationsFor(defs) {
  if (!defs || defs.length === 0) {
    return undefined;
  }
  return defs.map((def) => new vscode.Location(def.uri, def.range));
}

function defaultImportAlias(domainName) {
  const parts = domainName.split(".");
  return parts[parts.length - 1];
}

function positionAt(text, offset) {
  const prefix = text.slice(0, offset);
  const lines = prefix.split(/\r\n|\r|\n/);
  return new vscode.Position(lines.length - 1, lines[lines.length - 1].length);
}

module.exports = {
  activate,
  deactivate,
  _test: {
    defaultImportAlias,
    maskCommentsAndStrings,
    parseSkelFile,
    positionAt
  }
};
