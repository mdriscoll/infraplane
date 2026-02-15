/**
 * Client-side directory picker and infrastructure file reader.
 *
 * Uses the File System Access API (showDirectoryPicker) to let the user
 * select a local folder, then reads infrastructure-relevant files and
 * returns their contents for uploading to the backend.
 *
 * This is needed because browsers cannot expose real filesystem paths
 * for security reasons — so we read the files client-side and upload
 * the contents to the server for LLM analysis.
 */

export interface FileContent {
  path: string
  content: string
}

/** Max file size to read (10KB) — matches backend analyzer limit. */
const MAX_FILE_SIZE = 10 * 1024

/** Max lines for README.md — matches backend analyzer limit. */
const MAX_README_LINES = 200

/** Infrastructure-relevant filenames to look for at root. */
const INFRA_FILES = new Set([
  'Dockerfile',
  'docker-compose.yml',
  'docker-compose.yaml',
  'package.json',
  'requirements.txt',
  'Pipfile',
  'go.mod',
  'Gemfile',
  'pom.xml',
  'build.gradle',
  'serverless.yml',
  'serverless.yaml',
  'Procfile',
  '.env.example',
  '.env.sample',
  'Makefile',
  'README.md',
])

/** Subdirectory patterns to search for infrastructure files. */
const INFRA_SUBDIRS: { dir: string; extensions: string[] }[] = [
  { dir: 'k8s', extensions: ['.yaml', '.yml'] },
  { dir: 'kubernetes', extensions: ['.yaml', '.yml'] },
  { dir: 'prisma', extensions: ['.prisma'] },
  { dir: 'config', extensions: ['.yml', '.yaml'] },
  { dir: 'deploy', extensions: ['.sh', '.yaml', '.yml', '.ts', '.js'] },
  { dir: 'scripts', extensions: ['.sh', '.yaml', '.yml', '.ts', '.js'] },
  { dir: 'infra', extensions: ['.tf', '.tfvars', '.yaml', '.yml', '.sh'] },
  { dir: '.github/workflows', extensions: ['.yaml', '.yml'] },
]

/** File extensions to look for at root (glob equivalent). */
const INFRA_ROOT_EXTENSIONS = new Set(['.tf', '.tfvars'])

/** Additional root files that match glob patterns. */
const INFRA_ROOT_NAMES = new Set([
  'knexfile.js',
  'drizzle.config.ts',
  'drizzle.config.js',
])

/**
 * Returns true if the File System Access API is supported in the browser.
 */
export function isDirectoryPickerSupported(): boolean {
  return 'showDirectoryPicker' in window
}

/**
 * Opens a native directory picker dialog and reads infrastructure-relevant
 * files from the selected folder.
 *
 * @returns Array of file contents, or null if the user cancelled.
 * @throws Error if the API is not supported.
 */
export async function pickAndReadDirectory(): Promise<FileContent[] | null> {
  if (!isDirectoryPickerSupported()) {
    throw new Error('File System Access API is not supported in this browser. Please use Chrome or Edge.')
  }

  let dirHandle: FileSystemDirectoryHandle
  try {
    dirHandle = await window.showDirectoryPicker({ mode: 'read' })
  } catch (err) {
    // User cancelled the dialog
    if (err instanceof DOMException && err.name === 'AbortError') {
      return null
    }
    throw err
  }

  const files: FileContent[] = []
  const seenPaths = new Set<string>()

  // Read root-level infrastructure files
  for await (const [name, handle] of dirHandle.entries()) {
    if (handle.kind !== 'file') continue

    const ext = getExtension(name)

    const isInfraFile =
      INFRA_FILES.has(name) ||
      INFRA_ROOT_NAMES.has(name) ||
      INFRA_ROOT_EXTENSIONS.has(ext)

    if (isInfraFile) {
      const content = await readFileHandle(handle as FileSystemFileHandle, name)
      if (content !== null && !seenPaths.has(name)) {
        seenPaths.add(name)
        files.push({ path: name, content })
      }
    }
  }

  // Read infrastructure files from known subdirectories
  for (const { dir, extensions } of INFRA_SUBDIRS) {
    try {
      // Support nested paths like ".github/workflows"
      let subHandle: FileSystemDirectoryHandle = dirHandle
      for (const segment of dir.split('/')) {
        subHandle = await subHandle.getDirectoryHandle(segment)
      }
      for await (const [name, handle] of subHandle.entries()) {
        if (handle.kind !== 'file') continue
        const ext = getExtension(name)
        if (extensions.includes(ext)) {
          const relPath = `${dir}/${name}`
          if (!seenPaths.has(relPath)) {
            const content = await readFileHandle(handle as FileSystemFileHandle, name)
            if (content !== null) {
              seenPaths.add(relPath)
              files.push({ path: relPath, content })
            }
          }
        }
      }
    } catch {
      // Subdirectory doesn't exist — skip
    }
  }

  return files
}

/**
 * Reads a file handle with size and line limits.
 */
async function readFileHandle(
  handle: FileSystemFileHandle,
  name: string,
): Promise<string | null> {
  try {
    const file = await handle.getFile()

    // Read up to MAX_FILE_SIZE bytes
    const blob = file.slice(0, MAX_FILE_SIZE)
    let content = await blob.text()

    // Limit README to first N lines
    if (name.toLowerCase() === 'readme.md') {
      const lines = content.split('\n')
      if (lines.length > MAX_README_LINES) {
        content = lines.slice(0, MAX_README_LINES).join('\n')
      }
    }

    return content
  } catch {
    return null
  }
}

function getExtension(name: string): string {
  const dot = name.lastIndexOf('.')
  return dot >= 0 ? name.slice(dot) : ''
}
