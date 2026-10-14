import assert from 'node:assert/strict'
import test from 'node:test'
import { reactive } from 'vue'
import { brandPromptTemplates, brandPromptText } from './promptTemplates'
import { hydrateAgentPromptRefs, serializeAgentPrompts } from '../utils/agentPromptTemplates'
import type { PromptTemplatesConfig } from '../api/system'

function catalog(brand = 'WeKnora'): PromptTemplatesConfig {
  return {
    agent_system_prompt: [{ id: 'data_analyst', name: `${brand} Data Analyst`, description: `For ${brand}.`, content: `You are ${brand} Data Analyst. Use {{knowledge_id}} and data_analysis.`, default: true }],
    system_prompt: [], context_template: [],
    rewrite: [{ id: 'rewrite', name: 'Rewrite', description: '', content: `Rewrite for ${brand}.`, user: '{{query}}', default: true }],
    fallback: [{ id: 'fallback', name: 'Fallback', description: '', content: `${brand} cannot answer.`, mode: 'model', default: true }],
  }
}

test('old server catalog uses OPC prose while preserving technical contracts', () => {
  const original = catalog()
  const copy = brandPromptTemplates(original)
  assert.equal(copy.agent_system_prompt![0]!.content, 'You are OPC Data Analyst. Use {{knowledge_id}} and data_analysis.')
  assert.equal(copy.agent_system_prompt![0]!.id, 'data_analyst')
  assert.equal(copy.agent_system_prompt![0]!.name, 'OPC Data Analyst')
  assert.equal(copy.agent_system_prompt![0]!.description, 'For OPC.')
  assert.ok(original.agent_system_prompt![0]!.content.includes('WeKnora'), 'must not mutate source response')
  const contracts = '.weknora/requirements.json https://github.com/Tencent/WeKnora WEKNORA_BASE_URL weknora-lite {{WeKnora}} `WeKnora` ```\nWeKnora\n```'
  assert.equal(brandPromptText(contracts), contracts)
  assert.equal(brandPromptText('你是WeKnora助手，weknora 和 WEKNORA。'), '你是OPC助手，OPC 和 OPC。')
})

test('applying an adapted template saves OPC inline instead of resolving an old server ID', () => {
  const templates = reactive(brandPromptTemplates(catalog()))
  const form = hydrateAgentPromptRefs({ agent_mode: 'smart-reasoning', system_prompt_id: 'data_analyst' }, templates)
  assert.ok(form.system_prompt!.startsWith('You are OPC'))
  const saved = serializeAgentPrompts({ ...form, rewrite_prompt_system: templates.rewrite[0]!.content, fallback_prompt: templates.fallback[0]!.content }, templates)
  assert.ok(saved.system_prompt!.startsWith('You are OPC'))
  assert.equal(saved.system_prompt_id, undefined)
  assert.equal(saved.rewrite_prompt_system, 'Rewrite for OPC.')
  assert.equal(saved.fallback_prompt, 'OPC cannot answer.')
  assert.equal(JSON.stringify(templates).includes('WeKnora'), false, 'original body metadata must not be persisted')
})

test('OPC server templates retain normal reference tracking and custom prompts stay unchanged', () => {
  const templates = brandPromptTemplates(catalog('OPC'))
  const saved = serializeAgentPrompts({ agent_mode: 'smart-reasoning', system_prompt_id: 'data_analyst', system_prompt: templates.agent_system_prompt![0]!.content }, templates)
  assert.equal(saved.system_prompt, '')
  assert.equal(saved.system_prompt_id, 'data_analyst')
  const custom = '用户自定义 WeKnora 描述'
  assert.equal(serializeAgentPrompts({ agent_mode: 'smart-reasoning', system_prompt: custom }, templates).system_prompt, custom)
})
