import {mockApi} from './mockApi'

const target = '/home/demo/app/CLAUDE.md'

test('適用 → 履歴 → ロールバック', async () => {
  const api = mockApi()
  const tc = await api.readTarget(target)
  const p = await api.generateProposal({targetPath: target, project: '', days: 30, instruction: ''})
  expect(p.output).toContain('go vet')
  await api.applyProposal(target, p.output, tc.hash)
  expect((await api.readTarget(target)).content).toBe(p.output)
  const hist = await api.listBackups()
  expect(hist).toHaveLength(1)
  await api.rollback(hist[0].id)
  expect((await api.readTarget(target)).content).toBe(tc.content)
  expect(await api.listBackups()).toHaveLength(2)
})

test('ハッシュ不一致と許可外パスは拒否する', async () => {
  const api = mockApi()
  await expect(api.applyProposal(target, 'x', 'stale')).rejects.toThrow('変更されています')
  await expect(api.applyProposal('/etc/passwd', 'x', 'h')).rejects.toThrow('許可されていない')
  await expect(api.readTarget('/etc/passwd')).rejects.toThrow()
})

test('生成はキャンセルできる', async () => {
  const api = mockApi()
  const p = api.generateProposal({targetPath: target, project: '', days: 30, instruction: ''})
  await api.cancelGeneration()
  await expect(p).rejects.toThrow('キャンセル')
})

test('不正な正規表現の設定は保存できない', async () => {
  const api = mockApi()
  const c = await api.settings()
  await expect(api.saveSettings({...c, maskPatterns: [{name: 'x', regex: '('}]})).rejects.toThrow()
})
