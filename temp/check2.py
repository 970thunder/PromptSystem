# -*- coding: utf-8 -*-
import io, sys
sys.stdout = io.TextIOWrapper(sys.stdout.buffer, encoding='utf-8')
path = r'D:\AMP\PromptSystem\docs\后端迭代任务清单.md'
with io.open(path, 'r', encoding='utf-8-sig') as f:
    c = f.read()
old = '- [ ] 派发给 AI：新增 docs/API契约.md'
new = "- [x] 派发给 AI：新增 docs/API契约.md"
if old in c:
    c = c.replace(old, new)
    print('REPLACED')
else:
    print('NOT FOUND')
with io.open(path, 'w', encoding='utf-8') as f:
    f.write(c)
print('done')
