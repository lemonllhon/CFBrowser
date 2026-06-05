export type ProtoJSONPrimitive = string | number | boolean | null
export type ProtoJSONValue = ProtoJSONPrimitive | ProtoJSONObject | ProtoJSONArray
export type ProtoJSONObject = { [key: string]: ProtoJSONValue }
export type ProtoJSONArray = ProtoJSONValue[]

export function isProtoJSONObject(value: unknown): value is ProtoJSONObject {
  return !!value && typeof value === 'object' && !Array.isArray(value)
}

export function decodeProtoJSONObject(json: string, fallback: ProtoJSONObject = {}): ProtoJSONObject {
  if (!json) {
    return fallback
  }
  try {
    const parsed: unknown = JSON.parse(json)
    if (isProtoJSONObject(parsed)) {
      return parsed
    }
  } catch {
    // Ignore malformed diagnostic payloads from external services/backend logs.
  }
  return fallback
}
