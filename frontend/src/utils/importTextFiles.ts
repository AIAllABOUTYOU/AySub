export interface ImportTextFile {
  name: string
  content: string
}

export interface ImportTextSkippedFile {
  name: string
  reason: string
}

export interface ImportTextReadResult {
  loaded: ImportTextFile[]
  skipped: ImportTextSkippedFile[]
}

export interface ImportTextReadMessages {
  unsupportedFile: string
  invalidZip: string
  zipReadFailed: string
  zipUnsupportedBrowser: string
  unsupportedZipMethod: (method: number) => string
}

export interface ImportTextReadOptions {
  extensions: string[]
  messages: ImportTextReadMessages
}

const ZIP_EXTENSIONS = ['.zip']

export const readImportTextFiles = async (
  files: File[],
  options: ImportTextReadOptions
): Promise<ImportTextReadResult> => {
  const loaded: ImportTextFile[] = []
  const skipped: ImportTextSkippedFile[] = []

  for (const sourceFile of files) {
    const sourceName = sourceFile.webkitRelativePath || sourceFile.name
    if (isZipFile(sourceFile)) {
      try {
        const zipEntries = await readZipTextEntries(sourceFile, skipped, options)
        loaded.push(...zipEntries.map((entry) => ({
          name: `${sourceName}/${entry.name}`,
          content: entry.content
        })))
      } catch (error: any) {
        skipped.push({
          name: sourceName,
          reason: error?.message || options.messages.zipReadFailed
        })
      }
      continue
    }

    if (!hasSupportedExtension(sourceName, options.extensions)) {
      skipped.push({ name: sourceName, reason: options.messages.unsupportedFile })
      continue
    }

    loaded.push({
      name: sourceName,
      content: await readFileAsText(sourceFile)
    })
  }

  return { loaded, skipped }
}

export const readFileAsText = async (sourceFile: File): Promise<string> => {
  if (typeof sourceFile.text === 'function') {
    return sourceFile.text()
  }
  if (typeof sourceFile.arrayBuffer === 'function') {
    const buffer = await sourceFile.arrayBuffer()
    return new TextDecoder().decode(buffer)
  }
  return await new Promise<string>((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result ?? ''))
    reader.onerror = () => reject(reader.error || new Error('Failed to read file'))
    reader.readAsText(sourceFile)
  })
}

const isZipFile = (sourceFile: File): boolean => {
  return hasSupportedExtension(sourceFile.name, ZIP_EXTENSIONS) || sourceFile.type === 'application/zip'
}

const hasSupportedExtension = (name: string, extensions: string[]): boolean => {
  const lower = name.toLowerCase()
  return extensions.some((extension) => lower.endsWith(extension))
}

const readZipTextEntries = async (
  sourceFile: File,
  skipped: ImportTextSkippedFile[],
  options: ImportTextReadOptions
): Promise<ImportTextFile[]> => {
  const buffer = await sourceFile.arrayBuffer()
  const bytes = new Uint8Array(buffer)
  const view = new DataView(buffer)
  const eocdOffset = findZipEndOfCentralDirectory(view)
  if (eocdOffset < 0) {
    throw new Error(options.messages.invalidZip)
  }

  const entryCount = view.getUint16(eocdOffset + 10, true)
  let centralOffset = view.getUint32(eocdOffset + 16, true)
  const entries: ImportTextFile[] = []

  for (let index = 0; index < entryCount; index += 1) {
    if (view.getUint32(centralOffset, true) !== 0x02014b50) {
      throw new Error(options.messages.invalidZip)
    }

    const flags = view.getUint16(centralOffset + 8, true)
    const method = view.getUint16(centralOffset + 10, true)
    const compressedSize = view.getUint32(centralOffset + 20, true)
    const fileNameLength = view.getUint16(centralOffset + 28, true)
    const extraLength = view.getUint16(centralOffset + 30, true)
    const commentLength = view.getUint16(centralOffset + 32, true)
    const localOffset = view.getUint32(centralOffset + 42, true)
    const nameBytes = bytes.slice(centralOffset + 46, centralOffset + 46 + fileNameLength)
    const entryName = decodeZipName(nameBytes, flags)

    centralOffset += 46 + fileNameLength + extraLength + commentLength

    if (entryName.endsWith('/') || !hasSupportedExtension(entryName, options.extensions)) {
      continue
    }

    if (view.getUint32(localOffset, true) !== 0x04034b50) {
      throw new Error(options.messages.invalidZip)
    }

    const localFileNameLength = view.getUint16(localOffset + 26, true)
    const localExtraLength = view.getUint16(localOffset + 28, true)
    const dataStart = localOffset + 30 + localFileNameLength + localExtraLength
    const compressed = bytes.slice(dataStart, dataStart + compressedSize)
    let output: Uint8Array

    if (method === 0) {
      output = compressed
    } else if (method === 8) {
      output = await inflateRawZipEntry(compressed, options.messages.zipUnsupportedBrowser)
    } else {
      skipped.push({
        name: `${sourceFile.name}/${entryName}`,
        reason: options.messages.unsupportedZipMethod(method)
      })
      continue
    }

    entries.push({
      name: entryName,
      content: new TextDecoder().decode(output)
    })
  }

  return entries
}

const findZipEndOfCentralDirectory = (view: DataView): number => {
  const minOffset = Math.max(0, view.byteLength - 22 - 0xffff)
  for (let offset = view.byteLength - 22; offset >= minOffset; offset -= 1) {
    if (view.getUint32(offset, true) === 0x06054b50) {
      return offset
    }
  }
  return -1
}

const decodeZipName = (bytes: Uint8Array, flags: number): string => {
  const encoding = (flags & 0x0800) !== 0 ? 'utf-8' : 'utf-8'
  return new TextDecoder(encoding).decode(bytes)
}

const inflateRawZipEntry = async (input: Uint8Array, unsupportedMessage: string): Promise<Uint8Array> => {
  const DecompressionStreamCtor = (globalThis as typeof globalThis & {
    DecompressionStream?: new (format: string) => DecompressionStream
  }).DecompressionStream
  if (!DecompressionStreamCtor) {
    throw new Error(unsupportedMessage)
  }

  const stream = new Blob([input]).stream().pipeThrough(new DecompressionStreamCtor('deflate-raw'))
  return new Uint8Array(await new Response(stream).arrayBuffer())
}
