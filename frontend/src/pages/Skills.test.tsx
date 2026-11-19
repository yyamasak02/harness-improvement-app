import {render, screen} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {api} from '../api'
import Skills from './Skills'

async function fillAndGenerate(name: string) {
  const list = await screen.findByRole('list', {name: 'メモリ選択'})
  const boxes = await screen.findAllByRole('checkbox')
  expect(list).toBeInTheDocument()
  await userEvent.click(boxes[0])
  await userEvent.type(screen.getByLabelText('skill 名'), name)
  await userEvent.click(screen.getByRole('button', {name: '生成'}))
}

test('メモリと名前が揃うまで生成できない', async () => {
  render(<Skills />)
  await screen.findAllByRole('checkbox')
  expect(screen.getByRole('button', {name: '生成'})).toBeDisabled()
})

test('生成 → 差分表示 → 適用 → 履歴に残る。生成しただけでは書き込まれない', async () => {
  const before = (await api.listBackups()).length
  render(<Skills />)
  await fillAndGenerate('my-skill')
  expect(await screen.findByText('SKILL.md（新規）')).toBeInTheDocument()
  expect(screen.getByText('knowledge/notes.md（新規）')).toBeInTheDocument()
  expect(screen.getByText(/メモリに「応答は日本語」/)).toBeInTheDocument()
  expect((await api.listBackups()).length).toBe(before)
  await userEvent.click(screen.getByRole('button', {name: '適用'}))
  expect(await screen.findByText(/2 ファイルを適用しました/)).toBeInTheDocument()
  expect((await api.listBackups()).length).toBe(before + 2)
})

test('破棄すると何も書き込まれず、不正な名前はエラーになる', async () => {
  const before = (await api.listBackups()).length
  render(<Skills />)
  await fillAndGenerate('discard-me')
  await userEvent.click(await screen.findByRole('button', {name: '破棄'}))
  expect((await api.listBackups()).length).toBe(before)

  await userEvent.clear(screen.getByLabelText('skill 名'))
  await userEvent.type(screen.getByLabelText('skill 名'), 'Bad Name')
  await userEvent.click(screen.getByRole('button', {name: '生成'}))
  expect(await screen.findByRole('alert')).toHaveTextContent('skill 名が不正')
})

test('編集した内容が適用される', async () => {
  render(<Skills />)
  await fillAndGenerate('edited')
  await userEvent.click((await screen.findAllByRole('button', {name: '右側を編集'}))[0])
  await userEvent.type(screen.getByRole('textbox', {name: '改善案の編集'}), '追記')
  await userEvent.click(screen.getByRole('button', {name: '適用'}))
  await screen.findByText(/2 ファイルを適用しました/)
  const hist = await api.listBackups()
  const skillMd = hist.find((h) => h.path.endsWith('/edited/SKILL.md'))
  expect((await api.getBackup(skillMd!.id)).after).toContain('追記')
})
