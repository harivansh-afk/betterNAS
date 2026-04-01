import { readdirSync, readFileSync, statSync } from "node:fs";
import path from "node:path";

const repoRoot = process.cwd();
const sourceExtensions = new Set([".js", ".mjs", ".cjs", ".ts", ".tsx"]);
const ignoredDirectories = new Set([
  ".git",
  ".next",
  ".turbo",
  "coverage",
  "dist",
  "node_modules",
]);

const laneRoots = [];
for (const baseDir of ["apps", "packages"]) {
  const absoluteBaseDir = path.join(repoRoot, baseDir);
  for (const entry of readdirSync(absoluteBaseDir, { withFileTypes: true })) {
    if (entry.isDirectory()) {
      laneRoots.push(path.join(absoluteBaseDir, entry.name));
    }
  }
}

const disallowedWorkspacePackages = new Set([
  "@betternas/sdk-ts",
  "@betternas/web",
  "@betternas/control-plane",
  "@betternas/node-agent",
  "@betternas/nextcloud-app",
]);

const importPattern =
  /\b(?:import|export)\b[\s\S]*?\bfrom\s*["']([^"']+)["']|import\s*\(\s*["']([^"']+)["']\s*\)/g;

const errors = [];

walk(path.join(repoRoot, "apps"));
walk(path.join(repoRoot, "packages"));

if (errors.length > 0) {
  console.error("Boundary check failed:\n");
  for (const error of errors) {
    console.error(`- ${error}`);
  }
  process.exit(1);
}

console.log("Boundary check passed.");

function walk(currentPath) {
  const stat = statSync(currentPath);
  if (stat.isDirectory()) {
    if (ignoredDirectories.has(path.basename(currentPath))) {
      return;
    }

    for (const entry of readdirSync(currentPath, { withFileTypes: true })) {
      walk(path.join(currentPath, entry.name));
    }
    return;
  }

  if (!sourceExtensions.has(path.extname(currentPath))) {
    return;
  }

  const fileContent = readFileSync(currentPath, "utf8");
  const fileRoot = findLaneRoot(currentPath);
  if (fileRoot === null) {
    return;
  }

  for (const match of fileContent.matchAll(importPattern)) {
    const specifier = match[1] ?? match[2];
    if (!specifier) {
      continue;
    }

    if (disallowedWorkspacePackages.has(specifier)) {
      errors.push(
        `${relativeToRepo(currentPath)} imports forbidden workspace package ${specifier}`,
      );
      continue;
    }

    if (!specifier.startsWith(".")) {
      continue;
    }

    const resolvedImport = path.resolve(path.dirname(currentPath), specifier);
    const targetRoot = findLaneRoot(resolvedImport);
    if (targetRoot !== null && targetRoot !== fileRoot) {
      errors.push(
        `${relativeToRepo(currentPath)} crosses lane boundary with relative import ${specifier}`,
      );
    }
  }
}

function findLaneRoot(targetPath) {
  const normalizedTargetPath = path.normalize(targetPath);
  for (const laneRoot of laneRoots) {
    const normalizedLaneRoot = path.normalize(laneRoot);
    if (
      normalizedTargetPath === normalizedLaneRoot ||
      normalizedTargetPath.startsWith(`${normalizedLaneRoot}${path.sep}`)
    ) {
      return normalizedLaneRoot;
    }
  }
  return null;
}

function relativeToRepo(targetPath) {
  return path.relative(repoRoot, targetPath) || ".";
}
