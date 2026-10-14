import type { PromptTemplate, PromptTemplatesConfig } from '../api/system'

const serverText = Symbol('server-prompt-text')
type PromptField = 'content' | 'user'
type BrandedTemplate = PromptTemplate & { [serverText]?: Partial<Record<PromptField, string>> }

// Brand prose only. Code, paths, identifiers and placeholders are executable contracts.
export function brandPromptText(text: string): string {
  return text.replace(/```[\s\S]*?```|`[^`\n]*`|[^\s]*[\/\\][^\s]*|\{\{[\s\S]*?\}\}|\b[\w.-]*weknora[\w.-]*\b/gi,
    (token, offset: number) => /^weknora$/i.test(token) && text[offset - 1] !== '.' ? 'OPC' : token)
}

export function serverPromptText(template: PromptTemplate | undefined, field: PromptField): string | undefined {
  if (!template) return undefined
  return (template as BrandedTemplate)[serverText]?.[field] ?? template[field]
}

// Applied only to the server's system template catalog, never saved custom agent text.
export function brandPromptTemplates(config: PromptTemplatesConfig): PromptTemplatesConfig {
  const result = { ...config }
  for (const [group, list] of Object.entries(config)) {
    if (!Array.isArray(list)) continue
    const adapted = list.map((template: PromptTemplate) => {
      const copy: BrandedTemplate = { ...template }
      for (const field of ['name', 'description', 'content', 'user'] as const) {
        if (typeof copy[field] === 'string') copy[field] = brandPromptText(copy[field]!)
      }
      // Keep server semantics when serializing template refs. A branded body must
      // be saved inline until the server itself provides the same OPC template.
      if (copy.content !== template.content || copy.user !== template.user) {
        Object.defineProperty(copy, serverText, { configurable: true, value: { content: template.content, user: template.user } })
      }
      return copy
    })
    ;(result as unknown as Record<string, unknown>)[group] = adapted
  }
  return result
}
