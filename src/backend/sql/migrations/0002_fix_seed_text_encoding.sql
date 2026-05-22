UPDATE categories SET name = 'Image Generation' WHERE id = 1;
UPDATE categories SET name = 'Copywriting' WHERE id = 2;
UPDATE categories SET name = 'Coding' WHERE id = 3;
UPDATE categories SET name = 'Video Generation' WHERE id = 4;
UPDATE categories SET name = 'Agent Prompt' WHERE id = 5;
UPDATE categories SET name = 'Workflow' WHERE id = 6;
UPDATE categories SET name = 'Content Creation' WHERE id = 7;
UPDATE categories SET name = 'Ecommerce Ops' WHERE id = 8;
UPDATE categories SET name = 'Data Analysis' WHERE id = 9;
UPDATE categories SET name = 'AI Support' WHERE id = 10;
UPDATE categories SET name = 'Coding Automation' WHERE id = 11;
UPDATE categories SET name = 'Multi-Agent Collaboration' WHERE id = 12;

UPDATE prompts SET
    title = 'Brand Poster Prompt Builder',
    description = 'Turn a short slogan into polished Midjourney and SDXL prompt variants for campaign visuals and ecommerce launches.'
WHERE id = 101;

UPDATE prompts SET
    title = 'SaaS Landing Page Copy Rewrite',
    description = 'Input product positioning and competitor context, then generate homepage headlines, supporting copy, feature points, and CTA options.'
WHERE id = 102;

UPDATE prompts SET
    title = 'Code Review Assistant',
    description = 'Review a PR diff, prioritize risks, call out regressions, and suggest targeted follow-up tests for engineering teams.'
WHERE id = 103;

UPDATE prompts SET
    title = 'Short Video Script Factory',
    description = 'Generate 15s, 30s, and 60s short-form video scripts from product selling points, complete with hook and pacing guidance.'
WHERE id = 104;

UPDATE prompts SET
    title = 'Multi-Agent Research Coordinator',
    description = 'Break a research goal into parallel agent tasks, reporting contracts, and a practical merge plan for final synthesis.'
WHERE id = 105;

UPDATE prompts SET
    title = 'Customer Support SOP Workflow',
    description = 'Generate customer support flows, escalation rules, and reusable response templates from docs, FAQs, and tone guidance.'
WHERE id = 106;

DELETE FROM prompt_tags WHERE prompt_id IN (101, 102, 103, 104, 105, 106);

INSERT INTO prompt_tags (prompt_id, tag) VALUES
    (101, 'Brand'),
    (101, 'Poster'),
    (101, 'Ecommerce'),
    (102, 'SaaS'),
    (102, 'Conversion'),
    (102, 'Marketing'),
    (103, 'Code Review'),
    (103, 'Engineering'),
    (103, 'PR'),
    (104, 'Short Video'),
    (104, 'Script'),
    (104, 'Growth'),
    (105, 'Agent'),
    (105, 'Research'),
    (105, 'Collaboration'),
    (106, 'Support'),
    (106, 'SOP'),
    (106, 'Process');
