const textEncoder = new TextEncoder()
const textDecoder = new TextDecoder()

export const WireType = {
  Varint: 0,
  LengthDelimited: 2,
} as const

export type WireTypeValue = (typeof WireType)[keyof typeof WireType]

export type ProtoField = {
  fieldNumber: number
  wireType: WireTypeValue
  value: Uint8Array
}

export function concatBytes(parts: Uint8Array[]): Uint8Array {
  const length = parts.reduce((sum, part) => sum + part.length, 0)
  const result = new Uint8Array(length)
  let offset = 0
  for (const part of parts) {
    result.set(part, offset)
    offset += part.length
  }
  return result
}

export function encodeStringField(fieldNumber: number, value: string): Uint8Array {
  if (!value) {
    return new Uint8Array()
  }
  return encodeLengthDelimitedField(fieldNumber, textEncoder.encode(value))
}

export function encodeBytesField(fieldNumber: number, value: Uint8Array): Uint8Array {
  if (value.length === 0) {
    return new Uint8Array()
  }
  return encodeLengthDelimitedField(fieldNumber, value)
}

export function encodeInt32Field(fieldNumber: number, value: number): Uint8Array {
  if (value === 0) {
    return new Uint8Array()
  }
  const integer = Math.trunc(value)
  const encoded = integer < 0
    ? BigInt.asUintN(64, BigInt(integer))
    : BigInt(integer)
  return concatBytes([encodeTag(fieldNumber, WireType.Varint), encodeVarint(encoded)])
}

export function encodeBoolField(fieldNumber: number, value: boolean): Uint8Array {
  if (!value) {
    return new Uint8Array()
  }
  return concatBytes([encodeTag(fieldNumber, WireType.Varint), encodeVarint(1n)])
}

export function encodeInt64Field(fieldNumber: number, value: number): Uint8Array {
  if (value === 0) {
    return new Uint8Array()
  }
  const integer = Math.trunc(value)
  const encoded = integer < 0
    ? BigInt.asUintN(64, BigInt(integer))
    : BigInt(integer)
  return concatBytes([encodeTag(fieldNumber, WireType.Varint), encodeVarint(encoded)])
}

export function decodeString(value: Uint8Array): string {
  return textDecoder.decode(value)
}

export function decodeVarintField(value: Uint8Array): bigint {
  const decoded = decodeVarint(value, 0)
  if (decoded.nextOffset !== value.length) {
    throw new Error('Protobuf varint 字段包含多余字节')
  }
  return decoded.value
}

export function readFields(bytes: Uint8Array): ProtoField[] {
  const fields: ProtoField[] = []
  let offset = 0
  while (offset < bytes.length) {
    const tag = decodeVarint(bytes, offset)
    offset = tag.nextOffset
    const tagValue = Number(tag.value)
    const fieldNumber = tagValue >>> 3
    const wireType = (tagValue & 0x7) as WireTypeValue

    if (fieldNumber <= 0) {
      throw new Error('Protobuf 字段编号无效')
    }

    if (wireType === WireType.Varint) {
      const start = offset
      const decoded = decodeVarint(bytes, offset)
      offset = decoded.nextOffset
      fields.push({ fieldNumber, wireType, value: bytes.slice(start, offset) })
      continue
    }

    if (wireType === WireType.LengthDelimited) {
      const length = decodeVarint(bytes, offset)
      offset = length.nextOffset
      const end = offset + Number(length.value)
      if (end > bytes.length) {
        throw new Error('Protobuf length-delimited 字段越界')
      }
      fields.push({ fieldNumber, wireType, value: bytes.slice(offset, end) })
      offset = end
      continue
    }

    throw new Error(`暂不支持的 Protobuf wire type: ${wireType}`)
  }
  return fields
}

function encodeLengthDelimitedField(fieldNumber: number, value: Uint8Array): Uint8Array {
  return concatBytes([encodeTag(fieldNumber, WireType.LengthDelimited), encodeVarint(BigInt(value.length)), value])
}

function encodeTag(fieldNumber: number, wireType: WireTypeValue): Uint8Array {
  return encodeVarint(BigInt((fieldNumber << 3) | wireType))
}

function encodeVarint(value: bigint): Uint8Array {
  if (value < 0n) {
    throw new Error('Protobuf varint 不支持负数')
  }
  const bytes: number[] = []
  let cursor = value
  while (cursor >= 0x80n) {
    bytes.push(Number((cursor & 0x7fn) | 0x80n))
    cursor >>= 7n
  }
  bytes.push(Number(cursor))
  return new Uint8Array(bytes)
}

function decodeVarint(bytes: Uint8Array, startOffset: number): { value: bigint; nextOffset: number } {
  let value = 0n
  let shift = 0n
  let offset = startOffset
  while (offset < bytes.length) {
    const byte = bytes[offset]
    value |= BigInt(byte & 0x7f) << shift
    offset += 1
    if ((byte & 0x80) === 0) {
      return { value, nextOffset: offset }
    }
    shift += 7n
    if (shift > 63n) {
      throw new Error('Protobuf varint 超过 64 位')
    }
  }
  throw new Error('Protobuf varint 未结束')
}
