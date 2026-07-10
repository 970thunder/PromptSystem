// 文件作用：从提示词正文中提取只读展示用的 Few-shot 示例和流程步骤。
import type { PromptExample, PromptWorkflowStep } from '@/types'

const knownExampleLabels = ['input', '输入', '用户', 'user', 'output', '输出', '结果', 'assistant']

export function extractPromptExamples(content: string): PromptExample[] {
  const blocks = splitSection(content, /(few[-\s]?shot|示例|案例)/i)
  return blocks
    .slice(0, 3)
    .map((block, index) => ({
      title: `示例 ${index + 1}`,
      input: extractLabeledText(block, ['input', '输入', '用户', 'user']),
      output: extractLabeledText(block, ['output', '输出', '结果', 'assistant'])
    }))
    .filter((item) => item.input || item.output)
}

export function extractPromptWorkflow(content: string): PromptWorkflowStep[] {
  const blocks = splitSection(content, /(workflow|流程|步骤|sop)/i)
  return blocks
    .flatMap((block) => block.split(/\r?\n/))
    .map((line) => line.replace(/^[-*\d.\s、)）]+/, '').trim())
    .filter(Boolean)
    .slice(0, 6)
    .map((line, index) => ({ title: `步骤 ${index + 1}`, detail: line }))
}

function splitSection(content: string, headingPattern: RegExp): string[] {
  const lines = content.split(/\r?\n/)
  const blocks: string[] = []
  let collecting = false
  let current: string[] = []

  for (const line of lines) {
    const heading = normalizeHeading(line)
    if (heading && headingPattern.test(heading)) {
      pushBlock(blocks, current)
      collecting = true
      current = []
      continue
    }

    if (collecting && heading) {
      pushBlock(blocks, current)
      collecting = false
      current = []
      continue
    }

    if (collecting) {
      current.push(line)
    }
  }

  pushBlock(blocks, current)
  return blocks
}

function normalizeHeading(line: string): string {
  const trimmed = line.trim()
  if (!trimmed) {
    return ''
  }

  const markdownHeading = trimmed.match(/^#{1,6}\s+(.+)$/)
  if (markdownHeading) {
    return markdownHeading[1].trim()
  }

  const bracketHeading = trimmed.match(/^【(.+)】$/)
  if (bracketHeading) {
    return bracketHeading[1].trim()
  }

  return ''
}

function pushBlock(blocks: string[], lines: string[]) {
  const block = lines.join('\n').trim()
  if (block) {
    blocks.push(block)
  }
}

function extractLabeledText(block: string, labels: string[]): string {
  const lines = block.split(/\r?\n/)
  const result: string[] = []
  let collecting = false

  for (const line of lines) {
    const matchedLabel = findLineLabel(line)
    if (matchedLabel) {
      if (collecting && !labels.includes(matchedLabel)) {
        break
      }

      collecting = labels.includes(matchedLabel)
      if (collecting) {
        result.push(stripLineLabel(line))
      }
      continue
    }

    if (collecting) {
      result.push(line.trim())
    }
  }

  return result.filter(Boolean).join('\n').trim()
}

function findLineLabel(line: string): string {
  const normalized = line.trim().toLowerCase()
  for (const label of knownExampleLabels) {
    if (normalized.startsWith(`${label.toLowerCase()}:`) || normalized.startsWith(`${label.toLowerCase()}：`)) {
      return label
    }
  }

  return ''
}

function stripLineLabel(line: string): string {
  return line.replace(/^[^:：]+[:：]\s*/, '').trim()
}
